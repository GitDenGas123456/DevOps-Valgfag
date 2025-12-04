package scraper

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

type ScrapedResult struct {
	Title   string
	URL     string
	Snippet string
}

type wikiResponse struct {
	Query struct {
		Search []struct {
			Title   string `json:"title"`
			Snippet string `json:"snippet"`
			PageID  int    `json:"pageid"`
		} `json:"search"`
	} `json:"query"`
}

// WikipediaSearch queries the Wikipedia API for a search term.
func WikipediaSearch(query string, limit int) ([]ScrapedResult, error) {
	endpoint := "https://en.wikipedia.org/w/api.php"

	// Simple sanity clamp for limit
	if limit <= 0 {
		limit = 10
	} else if limit > 50 {
		limit = 50
	}

	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	q.Add("action", "query")
	q.Add("list", "search")
	q.Add("srsearch", query)
	q.Add("format", "json")
	q.Add("srlimit", fmt.Sprintf("%d", limit))
	req.URL.RawQuery = q.Encode()

	ua := strings.TrimSpace(os.Getenv("WIKI_USER_AGENT"))
	if ua == "" {
		ua = "WhoKnowsBot/1.0 (+https://github.com/GitDenGas123456/DevOps-Valgfag)"
	}
	req.Header.Set("User-Agent", ua)

	// Use a client with timeout instead of http.DefaultClient
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("wikipedia API returned status %d", resp.StatusCode)
	}

	var data wikiResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	results := make([]ScrapedResult, 0, len(data.Query.Search))
	for _, r := range data.Query.Search {
		results = append(results, ScrapedResult{
			Title:   r.Title,
			URL:     fmt.Sprintf("https://en.wikipedia.org/?curid=%d", r.PageID),
			Snippet: r.Snippet, // html/template escapes on render
		})
	}

	log.Printf("WikipediaSearch: found %d results for query %q\n", len(results), query)
	return results, nil
}
