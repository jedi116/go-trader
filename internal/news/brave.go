package news

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type BraveClient struct {
	APIKey  string
	BaseURL string
	http    *http.Client
}

type BraveNewsItem struct {
	Title     string `json:"title"`
	Url       string `json:"url"`
	Snippet   string `json:"snippet"`
	Source    string `json:"source"`
	Published string `json:"published"`
}

func NewBraveClient(apiKey, baseURL string) *BraveClient {
	if baseURL == "" {
		baseURL = "https://api.search.brave.com"
	}
	return &BraveClient{
		APIKey:  apiKey,
		BaseURL: baseURL,
		http:    &http.Client{Timeout: 15 * time.Second},
	}
}

func (b *BraveClient) SearchNews(ctx context.Context, query string, count int) ([]BraveNewsItem, error) {
	if count <= 0 {
		count = 10
	}
	endpoint := fmt.Sprintf("%s/res/v1/news/search", b.BaseURL)
	q := url.Values{}
	q.Set("q", query)
	q.Set("count", fmt.Sprintf("%d", count))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint+"?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	if b.APIKey != "" {
		req.Header.Set("X-Subscription-Token", b.APIKey)
	}
	resp, err := b.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("brave api status %d", resp.StatusCode)
	}
	var payload struct {
		Results []struct {
			Title       string `json:"title"`
			Url         string `json:"url"`
			Description string `json:"description"`
			Source      struct {
				Name string `json:"name"`
			} `json:"source"`
			Published string `json:"published"`
		} `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	items := make([]BraveNewsItem, 0, len(payload.Results))
	for _, r := range payload.Results {
		items = append(items, BraveNewsItem{
			Title:     r.Title,
			Url:       r.Url,
			Snippet:   r.Description,
			Source:    r.Source.Name,
			Published: r.Published,
		})
	}
	return items, nil
}
