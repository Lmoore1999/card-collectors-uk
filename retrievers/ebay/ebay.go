package ebay

import (
	"card-collectors-uk/analysis"
	"card-collectors-uk/card"
	"card-collectors-uk/database"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	NumberOfWorkers int = 10
)

type Retriever struct {
	name              string
	searchURL         string
	clientID          string
	clientSecret      string
	playersToRetrieve []database.PlayerRow
	setsToRetrieve    []database.SetRow
}

type job struct {
	Set    database.SetRow
	Player database.PlayerRow
}

type listingResult struct {
	Set      database.SetRow
	Player   database.PlayerRow
	Listings []Item
	Err      error
}

func InitialiseRetriever(ctx context.Context, sets []database.SetRow, players []database.PlayerRow) (Retriever, error) {
	var retriever Retriever
	retriever.name = "Ebay"

	apiCredentials, err := database.GetAPICredentialsByMarketplaceName(ctx, retriever.Name())
	if err != nil {
		return Retriever{}, fmt.Errorf("failed to retrieve api credentials: %w", err)
	}

	decryptedPassword, err := database.DecryptSecret(apiCredentials.EncryptedPassword)
	if err != nil {
		return Retriever{}, fmt.Errorf("failed to decrypt api credentials: %w", err)
	}

	return Retriever{
		name:              "Ebay",
		searchURL:         "https://api.ebay.com/buy/browse/v1/item_summary/search",
		clientID:          apiCredentials.Username,
		clientSecret:      decryptedPassword,
		playersToRetrieve: players,
		setsToRetrieve:    sets,
	}, nil
}

func (r Retriever) Name() string {
	return r.name
}

func (r Retriever) GetListings(ctx context.Context) (map[card.CanonicalCard][]analysis.Listing, error) {
	accessToken, err := r.getEbayAccessToken()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve access token: %w", err)
	}

	results := make(map[card.CanonicalCard][]analysis.Listing)
	var mu sync.Mutex

	numberOfWorkers := 1
	jobs := make(chan job)
	output := make(chan listingResult)

	var workers sync.WaitGroup
	for i := 0; i < numberOfWorkers; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for j := range jobs {
				listings, err := r.fetchListings(accessToken, j.Set, j.Player)
				if err != nil {
					log.Printf("Failed to fetch listings: %v", err)
				}
				output <- listingResult{
					Set:      j.Set,
					Player:   j.Player,
					Listings: listings,
					Err:      err,
				}
			}
		}()
	}

	go func() {
		for _, set := range r.setsToRetrieve {
			for _, player := range r.playersToRetrieve {
				select {
				case jobs <- job{Set: set, Player: player}:
				case <-ctx.Done():
					close(jobs)
					return
				}
			}
		}

		close(jobs)
	}()

	go func() {
		workers.Wait()
		close(output)
	}()

	for res := range output {
		if res.Err != nil {
			log.Printf("error fetching %s %s: %v", res.Set.SetName, res.Player.SecondName, res.Err)
			continue
		}

		for _, item := range res.Listings {
			canonicalCard, analysisListing, ok := r.processListing(item)
			if !ok {
				log.Printf("error processing %s %s: %v", res.Set.SetName, res.Player.SecondName, res.Err)
				continue
			}

			mu.Lock()
			results[canonicalCard] = append(results[canonicalCard], analysisListing)
			mu.Unlock()
		}
	}

	return results, nil
}

func (r Retriever) fetchListings(accessToken string, set database.SetRow, player database.PlayerRow) ([]Item, error) {
	var listings []Item
	offset := 0
	limit := 200

	query := fmt.Sprintf("%s %s %s %s", set.CompanyName, set.SetName, player.FirstName, player.SecondName)

	for {
		body, err := r.searchEbay(accessToken, query, offset, limit)
		if err != nil {
			return nil, err
		}

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

	fmt.Printf("%d listings found for query: %s\n", len(listings), query)
	return listings, nil
}

func (r Retriever) processListing(item Item) (card.CanonicalCard, analysis.Listing, bool) {
	canonicalCard, err := card.NewCanonicalCardFromListing(item.Title, item.ShortDescription, r.setsToRetrieve, r.playersToRetrieve)
	if err != nil {
		return card.CanonicalCard{}, analysis.Listing{}, false
	}

	price, err := strconv.ParseFloat(item.Price.Value, 32)
	if err != nil {
		return card.CanonicalCard{}, analysis.Listing{}, false
	}

	shipping := 0.0
	for _, opt := range item.ShippingOptions {
		v, err := strconv.ParseFloat(opt.ShippingCost.Value, 32)
		if err == nil {
			if shipping == 0 || v < shipping {
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
	), true
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
	endpoint := fmt.Sprintf(fmt.Sprintf("%s?q=%s&limit=%d&offset=%d&filter=deliveryCountry:GB", r.searchURL, url.QueryEscape(query), limit, offset))

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
