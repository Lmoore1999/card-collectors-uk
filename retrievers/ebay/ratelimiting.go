package ebay

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	browseAPIName        string = "browse"
	buyAPIContext        string = "buy"
	limitRefreshInterval        = time.Minute * 5
)

type Client struct {
	AccessToken string
	Limiter     *rateLimiter
}

type rateLimiter struct {
	mu         sync.Mutex
	remaining  int
	resetAt    time.Time
	timeWindow time.Duration
}

type Rate struct {
	Name       string `json:"name"`
	Count      int    `json:"count"`
	Limit      int    `json:"limit"`
	Remaining  int    `json:"remaining"`
	Reset      string `json:"reset"`
	TimeWindow int    `json:"timeWindow"`
}

type Resource struct {
	Name  string `json:"name"`
	Rates []Rate `json:"rates"`
}

type LimitBlock struct {
	ApiContext string     `json:"apiContext"`
	ApiName    string     `json:"apiName"`
	ApiVersion string     `json:"apiVersion"`
	Resources  []Resource `json:"resources"`
}

type RateLimitsResponse struct {
	RateLimits []LimitBlock `json:"rateLimits"`
}

func (r Retriever) waitBeforeCall() {
	var interval time.Duration

	r.client.Limiter.mu.Lock()
	if r.client.Limiter.remaining <= 0 {
		sleep := time.Until(r.client.Limiter.resetAt)
		r.client.Limiter.mu.Unlock()
		if sleep > 0 {
			fmt.Printf("Rate limit reached, sleeping %v\n", sleep)
			time.Sleep(sleep)
		}
		return
	}

	interval = r.client.Limiter.timeWindow / time.Duration(r.client.Limiter.remaining)
	r.client.Limiter.mu.Unlock()

	time.Sleep(interval)
}

func (r Retriever) reduceRemainingRequests() {
	r.client.Limiter.mu.Lock()
	r.client.Limiter.remaining--
	r.client.Limiter.mu.Unlock()
}

func (r Retriever) startLimitRefresher(ctx context.Context, refreshInterval time.Duration) {
	go func() {
		ticker := time.NewTicker(refreshInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				fmt.Println("Rate limit refresher stopped")
				return
			case <-ticker.C:
				if err := r.client.refreshLimits(); err != nil {
					fmt.Printf("Failed to refresh rate limits: %v\n", err)
				} else {
					r.client.Limiter.mu.Lock()
					fmt.Printf("Rate limits refreshed: remaining=%d, resetAt=%s\n",
						r.client.Limiter.remaining,
						r.client.Limiter.resetAt.Format(time.RFC3339))
					r.client.Limiter.mu.Unlock()
				}

			}
		}
	}()
}

func (c *Client) refreshLimits() error {
	rateLimits, err := getRateLimits(c.AccessToken, browseAPIName, buyAPIContext)
	if err != nil {
		return err
	}

	remaining, reset, window := extractBrowseLimits(rateLimits)
	c.Limiter.mu.Lock()
	c.Limiter.remaining = remaining
	c.Limiter.resetAt = reset
	c.Limiter.timeWindow = window
	c.Limiter.mu.Unlock()
	return nil
}

func getRateLimits(token, apiName, apiContext string) (*RateLimitsResponse, error) {
	endpoint := "https://api.ebay.com/developer/analytics/v1_beta/rate_limit/"

	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	q := req.URL.Query()
	if apiName != "" {
		q.Add("api_name", apiName)
	}
	if apiContext != "" {
		q.Add("api_context", apiContext)
	}
	req.URL.RawQuery = q.Encode()

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	var result RateLimitsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	err = resp.Body.Close()
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func extractBrowseLimits(resp *RateLimitsResponse) (remaining int, reset time.Time, window time.Duration) {
	for _, rl := range resp.RateLimits {
		if strings.EqualFold(rl.ApiName, browseAPIName) {
			for _, resource := range rl.Resources {
				if strings.EqualFold(resource.Name, "buy.browse") {
					for _, rate := range resource.Rates {
						resetTime, _ := time.Parse(time.RFC3339, rate.Reset)
						return rate.Remaining, resetTime, time.Duration(rate.TimeWindow) * time.Second
					}
				}
			}
		}
	}
	return 0, time.Now(), 0
}
