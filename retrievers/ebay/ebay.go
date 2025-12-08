package ebay

import (
	"card-collectors-uk/analysis"
	"card-collectors-uk/card"
	"card-collectors-uk/database"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"golang.org/x/text/unicode/norm"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
)

const (
	numberOfWorkers = 10
)

type Retriever struct {
	name              string
	searchURL         string
	clientID          string
	clientSecret      string
	playersToRetrieve []database.PlayerRow
	setsToRetrieve    []database.SetRow
	client            *Client
}

type job struct {
	Set     database.SetRow
	Players []database.PlayerRow
}

type listingResult struct {
	Set      database.SetRow
	Players  []database.PlayerRow
	Listings []Item
	Err      error
}

func InitialiseRetriever(ctx context.Context, sets []database.SetRow, players []database.PlayerRow) (Retriever, error) {
	var retriever Retriever
	retriever.name = "Ebay"
	retriever.searchURL = "https://api.ebay.com/buy/browse/v1/item_summary/search"
	retriever.setsToRetrieve = append(retriever.setsToRetrieve, sets...)
	retriever.playersToRetrieve = append(retriever.playersToRetrieve, players...)

	apiCredentials, err := database.GetAPICredentialsByMarketplaceName(ctx, retriever.Name())
	if err != nil {
		return Retriever{}, fmt.Errorf("failed to retrieve api credentials: %w", err)
	}

	decryptedPassword, err := database.DecryptSecret(apiCredentials.EncryptedPassword)
	if err != nil {
		return Retriever{}, fmt.Errorf("failed to decrypt api credentials: %w", err)
	}

	retriever.clientID = apiCredentials.Username
	retriever.clientSecret = decryptedPassword

	accessToken, err := retriever.getEbayAccessToken()
	if err != nil {
		return Retriever{}, fmt.Errorf("failed to retrieve access token: %w", err)
	}

	rateLimits, err := getRateLimits(accessToken, browseAPIName, buyAPIContext)
	if err != nil {
		return Retriever{}, fmt.Errorf("failed to retrieve rate limits: %w", err)
	}

	remaining, resetAt, timeWindow := extractBrowseLimits(rateLimits)
	retriever.client = &Client{
		AccessToken: accessToken,
		Limiter: &rateLimiter{
			remaining:  remaining,
			resetAt:    resetAt,
			timeWindow: timeWindow,
		},
	}

	log.Printf("Retriever initialised: {Name: %s, Limiter: {Remaining: %d, ResetAt: %v, TimeWindow: %d}}",
		retriever.name,
		retriever.client.Limiter.remaining,
		retriever.client.Limiter.resetAt,
		retriever.client.Limiter.timeWindow,
	)

	retriever.startLimitRefresher(ctx, limitRefreshInterval)

	return retriever, nil
}

func (r Retriever) Name() string {
	return r.name
}

func (r Retriever) GetAndStoreListings(ctx context.Context) {
	jobs := make(chan job)
	output := make(chan listingResult)

	var wg sync.WaitGroup
	for i := 0; i < numberOfWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				listings, err := r.fetchListings(j.Set, j.Players)
				output <- listingResult{
					Set:      j.Set,
					Players:  j.Players,
					Listings: listings,
					Err:      err,
				}
			}
		}()
	}

	go func() {
		batchedJobs := r.makeJobs()
		for _, jb := range batchedJobs {
			select {
			case jobs <- jb:
			case <-ctx.Done():
				close(jobs)
				return
			}
		}
		close(jobs)
	}()

	go func() {
		wg.Wait()
		close(output)
	}()

	for res := range output {
		if res.Err != nil {
			log.Printf("error fetching results: %v", res.Err)
			continue
		}

		for _, item := range res.Listings {
			canonicalCard, analysisListing, err := r.processListing(item)
			if err != nil {
				log.Printf("error processing %s: %v", item.Title, err)
				continue
			}

			listingID, err := database.InsertListingWithCanonicalCard(ctx, canonicalCard.Key(), analysisListing)
			if err != nil {
				log.Printf("error inserting listing %s: %v", canonicalCard.Key(), err)
				continue
			}

			log.Printf("ListingID %d successfully inserted", listingID)
		}
	}
}

func (r Retriever) fetchListings(set database.SetRow, players []database.PlayerRow) ([]Item, error) {
	var listings []Item
	offset := 0
	limit := 200

	names := make([]string, len(players))
	for i, p := range players {
		names[i] = cleanName(fmt.Sprintf("%s %s", p.FirstName, p.SecondName))
	}

	playersQuery := strings.Join(names, " OR ")
	query := fmt.Sprintf("%s %s %s", set.CompanyName, set.SetName, playersQuery)

	for {
		r.waitBeforeCall()

		body, err := r.searchEbay(r.client.AccessToken, query, offset, limit)
		if err != nil {
			return nil, err
		}

		r.reduceRemainingRequests()

		var response SearchResponse
		err = json.Unmarshal(body, &response)
		if err != nil {
			return nil, err
		}

		listings = append(listings, response.Items...)

		offset += limit
		if offset >= response.TotalResultCount {
			break
		}

		if offset+limit > response.TotalResultCount {
			limit = response.TotalResultCount - offset
		}
	}

	log.Printf("Query %s: listings count: %d", query, len(listings))

	return listings, nil
}

func (r Retriever) processListing(item Item) (card.CanonicalCard, analysis.Listing, error) {
	canonicalCard, err := card.NewCanonicalCardFromListing(item.Title, item.ShortDescription, r.setsToRetrieve, r.playersToRetrieve)
	if err != nil {
		return card.CanonicalCard{}, analysis.Listing{}, err
	}

	price, err := strconv.ParseFloat(item.Price.Value, 32)
	if err != nil {
		return card.CanonicalCard{}, analysis.Listing{}, fmt.Errorf("failed to parse price: %+v", err)
	}

	shipping := -1.0
	for _, opt := range item.ShippingOptions {
		v, err := strconv.ParseFloat(opt.ShippingCost.Value, 32)
		if err == nil {
			if shipping == -1.0 || v < shipping {
				shipping = v
			}
		}
	}

	return canonicalCard, analysis.NewListing(
		item.ItemID,
		r.Name(),
		item.Title,
		item.ShortDescription,
		item.ItemWebURL,
		price,
		item.Price.CurrencyCode,
		shipping,
		item.StartDate,
		item.EndDate,
	), nil
}

type SearchResponse struct {
	TotalResultCount int    `json:"total"`
	Items            []Item `json:"itemSummaries"`
}

type Item struct {
	ItemID           string            `json:"itemId"`
	Title            string            `json:"title"`
	Price            Price             `json:"price"`
	ShippingOptions  []ShippingOptions `json:"shippingOptions"`
	ItemWebURL       string            `json:"itemWebUrl"`
	ShortDescription string            `json:"shortDescription"`
	StartDate        time.Time         `json:"itemCreationDate"`
	EndDate          time.Time         `json:"itemEndDate"`
}

type Price struct {
	Value        string `json:"value"`
	CurrencyCode string `json:"currency"`
}

type ShippingOptions struct {
	ShippingCost Price `json:"shippingCost"`
}

func (r Retriever) searchEbay(accessToken string, query string, offset int, limit int) ([]byte, error) {
	endpoint := fmt.Sprintf("%s?q=%s&limit=%d&offset=%d&filter=deliveryCountry:GB", r.searchURL, url.QueryEscape(query), limit, offset)

	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-EBAY-C-MARKETPLACE-ID", "EBAY_GB")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	err = resp.Body.Close()
	if err != nil {
		return nil, err
	}

	return respBody, nil
}

type getEbayAccessTokenResponse struct {
	AccessToken string `json:"access_token"`
}

func (r Retriever) getEbayAccessToken() (string, error) {
	auth := base64.StdEncoding.EncodeToString([]byte(r.clientID + ":" + r.clientSecret))

	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	data.Set("scope", "https://api.ebay.com/oauth/api_scope")

	req, err := http.NewRequest("POST", "https://api.ebay.com/identity/v1/oauth2/token", strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Authorization", "Basic "+auth)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}

	var response getEbayAccessTokenResponse
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	err = json.Unmarshal(body, &response)
	if err != nil {
		return "", err
	}

	err = resp.Body.Close()
	if err != nil {
		return "", err
	}

	return response.AccessToken, nil
}

func (r Retriever) makeJobs() []job {
	var jobs []job
	var maxBatchSize = 1

	batches := batchPlayers(r.playersToRetrieve, maxBatchSize)

	for _, set := range r.setsToRetrieve {
		for _, batch := range batches {
			jobs = append(jobs, job{
				Set:     set,
				Players: batch,
			})
		}
	}

	return jobs
}

func batchPlayers(players []database.PlayerRow, maxBatchSize int) [][]database.PlayerRow {
	var batches [][]database.PlayerRow
	for i := 0; i < len(players); i += maxBatchSize {
		end := i + maxBatchSize
		if end > len(players) {
			end = len(players)
		}
		batches = append(batches, players[i:end])
	}
	return batches
}

func cleanName(name string) string {
	t := norm.NFD.String(name)
	return strings.Map(func(r rune) rune {
		if unicode.Is(unicode.Mn, r) { // remove accents
			return -1
		}
		return r
	}, t)
}
