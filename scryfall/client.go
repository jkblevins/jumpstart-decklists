// Package scryfall provides a client for the Scryfall card search API
// with local file caching and automatic rate limiting.
package scryfall

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
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
	baseURL   = "https://api.scryfall.com/cards/named"
	userAgent = "jumpforge/1.0 (MTG Jumpstart decklist formatter)"
	cacheTTL  = 7 * 24 * time.Hour // 1 week
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
		httpClient: &http.Client{Timeout: 10 * time.Second},
		cacheDir:   cacheDir(),
		log:        log.With().Str("component", "scryfall").Logger(),
	}
}

// FetchCard fetches a single card by exact name, using the cache when possible.
func (c *Client) FetchCard(name string) (*Card, error) {
	safe := safeFileName(name)

	// Check cache first.
	if data, ok := readCacheFile(c.cacheDir, safe, cacheTTL); ok {
		var card Card
		if err := json.Unmarshal(data, &card); err == nil {
			c.log.Debug().Str("card", name).Msg("cache hit")
			return &card, nil
		}
	}

	// Rate limit: 100ms between requests per Scryfall guidelines.
	if !c.lastReq.IsZero() {
		elapsed := time.Since(c.lastReq)
		if wait := 100*time.Millisecond - elapsed; wait > 0 {
			c.log.Debug().Dur("wait", wait).Msg("rate limit delay")
			time.Sleep(wait)
		}
	}

	return c.fetchFromAPI(name, safe)
}

func (c *Client) fetchFromAPI(name, safeName string) (*Card, error) {
	u := fmt.Sprintf("%s?exact=%s", baseURL, url.QueryEscape(name))

	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/json")

	start := time.Now()
	c.lastReq = start
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("scryfall request failed for %q: %w", name, err)
	}
	defer resp.Body.Close()

	// Handle 429: wait 2s and retry once.
	if resp.StatusCode == http.StatusTooManyRequests {
		resp.Body.Close()
		c.log.Warn().Str("card", name).Msg("rate limited (429), retrying in 2s")
		time.Sleep(2 * time.Second)

		// URL is already validated above; error can be safely ignored.
		req2, _ := http.NewRequest("GET", u, nil)
		req2.Header.Set("User-Agent", userAgent)
		req2.Header.Set("Accept", "application/json")
		start = time.Now()
		c.lastReq = start
		resp, err = c.httpClient.Do(req2)
		if err != nil {
			return nil, fmt.Errorf("scryfall retry failed for %q: %w", name, err)
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusTooManyRequests {
			return nil, fmt.Errorf("scryfall rate limited for %q after retry", name)
		}
	}

	duration := time.Since(start)

	if resp.StatusCode == http.StatusNotFound {
		c.log.Warn().Str("card", name).Msg("not found on Scryfall")
		return nil, fmt.Errorf("card not found: %q", name)
	}

	if resp.StatusCode != http.StatusOK {
		c.log.Error().Str("card", name).Int("status", resp.StatusCode).Msg("unexpected API response")
		return nil, fmt.Errorf("scryfall returned %d for %q", resp.StatusCode, name)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Cache the response (non-fatal if it fails).
	if err := writeCacheFile(c.cacheDir, safeName, body); err != nil {
		c.log.Warn().Err(err).Str("card", name).Msg("failed to write cache")
	}

	var card Card
	if err := json.Unmarshal(body, &card); err != nil {
		return nil, fmt.Errorf("failed to parse scryfall response for %q: %w", name, err)
	}

	c.log.Info().Str("card", name).Dur("duration", duration).Msg("fetched from API")
	return &card, nil
}

// FetchCards fetches all unique card names and returns a map of name to Card.
// Cards not found are logged and skipped.
func (c *Client) FetchCards(names []string) (map[string]*Card, error) {
	seen := make(map[string]bool)
	result := make(map[string]*Card)

	for _, name := range names {
		if seen[name] {
			continue
		}
		seen[name] = true

		card, err := c.FetchCard(name)
		if err != nil {
			continue // already logged in FetchCard/fetchFromAPI
		}
		result[name] = card
	}

	c.log.Info().Int("fetched", len(result)).Int("total", len(seen)).Msg("card fetch complete")
	return result, nil
}
