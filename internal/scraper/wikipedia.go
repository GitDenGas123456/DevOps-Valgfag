package scraper

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
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

// WikipediaSearch queries Wikipedia API for a search term
func WikipediaSearch(query string, limit int) ([]ScrapedResult, error) {
	endpoint := "https://en.wikipedia.org/w/api.php"
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

	req.Header.Set("User-Agent", "WhoKnowsBot/1.0 (+https://example.com)")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Wikipedia API returned status %d", resp.StatusCode)
	}

	var data wikiResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	results := []ScrapedResult{}
	for _, r := range data.Query.Search {
		results = append(results, ScrapedResult{
			Title:   r.Title,
			URL:     fmt.Sprintf("https://en.wikipedia.org/?curid=%d", r.PageID),
			Snippet: r.Snippet,
		})
	}

	log.Printf("WikipediaSearch: found %d results for query '%s'\n", len(results), query)
	return results, nil
}
