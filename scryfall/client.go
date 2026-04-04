// Package scryfall provides a client for the Scryfall card search API
// with local file caching and automatic rate limiting.
package scryfall

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/rs/zerolog"
)

// Card holds the Scryfall data needed for decklist rendering.
type Card struct {
	Name          string   `json:"name"`
	TypeLine      string   `json:"type_line"`
	ManaCost      string   `json:"mana_cost"`
	CMC           float64  `json:"cmc"`
	ColorIdentity []string `json:"color_identity"`
	Colors        []string `json:"colors"`
}

const (
	baseURL   = "https://api.scryfall.com"
	userAgent = "jumpstart-decklists/1.0 (MTG Jumpstart decklist formatter)"
	cacheTTL  = 30 * 24 * time.Hour // 1 month
	batchSize = 75                   // Scryfall /cards/collection max per request
)

// Client fetches card data from the Scryfall API with caching and rate limiting.
type Client struct {
	httpClient *http.Client
	cacheDir   string
	lastReq    time.Time
	log        zerolog.Logger
}

// NewClient creates a Scryfall client with default settings.
func NewClient(log zerolog.Logger) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		cacheDir:   cacheDir(),
		log:        log.With().Str("component", "scryfall").Logger(),
	}
}

// FetchCards looks up multiple cards by exact name. It serves what it can from
// cache and fetches the rest via Scryfall's /cards/collection endpoint (up to
// 75 cards per request). Returns a map keyed by card name.
func (c *Client) FetchCards(names []string) (map[string]*Card, error) {
	result := make(map[string]*Card, len(names))
	var uncached []string

	for _, name := range names {
		if data, ok := readCacheFile(c.cacheDir, safeFileName(name), cacheTTL); ok {
			var card Card
			if err := json.Unmarshal(data, &card); err == nil {
				result[card.Name] = &card
				c.log.Debug().Str("card", name).Msg("cache hit")
				continue
			}
		}
		uncached = append(uncached, name)
	}

	if len(uncached) == 0 {
		return result, nil
	}

	c.log.Info().Int("cached", len(result)).Int("fetching", len(uncached)).Msg("batch fetch from Scryfall")

	for i := 0; i < len(uncached); i += batchSize {
		end := i + batchSize
		if end > len(uncached) {
			end = len(uncached)
		}

		cards, notFound, err := c.fetchCollection(uncached[i:end])
		if err != nil {
			return nil, fmt.Errorf("fetch collection: %w", err)
		}

		for i := range cards {
			card := &cards[i]
			result[card.Name] = card
			if data, err := json.Marshal(card); err == nil {
				if werr := writeCacheFile(c.cacheDir, safeFileName(card.Name), data); werr != nil {
					c.log.Warn().Err(werr).Str("card", card.Name).Msg("failed to write cache")
				}
			}
		}

		for _, name := range notFound {
			c.log.Warn().Str("card", name).Msg("card not found on Scryfall")
		}
	}

	return result, nil
}

// collectionResponse is the response from POST /cards/collection.
type collectionResponse struct {
	Data     []Card             `json:"data"`
	NotFound []json.RawMessage  `json:"not_found"`
}

// fetchCollection calls POST /cards/collection for a batch of card names.
func (c *Client) fetchCollection(names []string) ([]Card, []string, error) {
	type identifier struct {
		Name string `json:"name"`
	}

	ids := make([]identifier, len(names))
	for i, name := range names {
		ids[i] = identifier{Name: name}
	}

	body, err := json.Marshal(struct {
		Identifiers []identifier `json:"identifiers"`
	}{Identifiers: ids})
	if err != nil {
		return nil, nil, err
	}

	respBody, err := c.doRequest("POST", baseURL+"/cards/collection", body)
	if err != nil {
		return nil, nil, err
	}

	var resp collectionResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, nil, fmt.Errorf("parse collection response: %w", err)
	}

	var notFound []string
	for _, raw := range resp.NotFound {
		var id identifier
		if json.Unmarshal(raw, &id) == nil {
			notFound = append(notFound, id.Name)
		}
	}

	return resp.Data, notFound, nil
}

// doRequest performs an HTTP request with rate limiting, retry on 429, and
// caching support. The body parameter is optional (nil for GET requests).
func (c *Client) doRequest(method, url string, body []byte) ([]byte, error) {
	respBody, err := c.executeWithRetry(method, url, body)
	if err != nil {
		return nil, err
	}
	return respBody, nil
}

// executeWithRetry sends an HTTP request with rate limiting.
// Retries up to 3 times on 429 with exponential backoff.
func (c *Client) executeWithRetry(method, url string, body []byte) ([]byte, error) {
	// Rate limit: 1 request per second to Scryfall.
	if wait := time.Second - time.Since(c.lastReq); wait > 0 {
		time.Sleep(wait)
	}

	respBody, status, err := c.sendRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	backoff := 2 * time.Second
	for attempt := 0; status == http.StatusTooManyRequests && attempt < 3; attempt++ {
		c.log.Warn().Dur("backoff", backoff).Msg("rate limited (429), retrying")
		time.Sleep(backoff)
		backoff *= 2
		respBody, status, err = c.sendRequest(method, url, body)
		if err != nil {
			return nil, err
		}
	}

	if status == http.StatusTooManyRequests {
		return nil, fmt.Errorf("rate limited after 3 retries")
	}

	if status != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d from %s", status, url)
	}

	return respBody, nil
}

// sendRequest performs a single HTTP request with standard headers.
func (c *Client) sendRequest(method, url string, body []byte) ([]byte, int, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	c.lastReq = time.Now()
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("request failed for %s: %w", url, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to read response for %s: %w", url, err)
	}

	return respBody, resp.StatusCode, nil
}
