package handlers

import (
	"net/http"

	"devops-valgfag/internal/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

var useFTSSearch bool

// EnableFTSSearch toggles FTS usage for search endpoints.
func EnableFTSSearch(on bool) {
	useFTSSearch = on
}

// SearchResult represents a single search result row
type SearchResult struct {
	ID          int
	Title       string
	URL         string
	Language    string
	Description string
}

// SearchPageHandler renders the web search page and results
func SearchPageHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	var timer *prometheus.Timer
	if q != "" {
		metrics.SearchTotal.Inc()
		timer = prometheus.NewTimer(metrics.SearchLatency)
		defer timer.ObserveDuration()
	}

	language := r.URL.Query().Get("language")
	if language == "" {
		language = "en"
	}

	var results []SearchResult
	if q != "" {
		rows, err := db.Query(
			`SELECT id, title, url, language, content
			 FROM pages
			 WHERE language = ? AND (title LIKE ? OR content LIKE ?)`,
			language, "%"+q+"%", "%"+q+"%",
		)
		if err == nil {
			defer func() { _ = rows.Close() }()
			for rows.Next() {
				var it SearchResult
				if err := rows.Scan(&it.ID, &it.Title, &it.URL, &it.Language, &it.Description); err == nil {
					results = append(results, it)
				}
			}
		}
	}

	if len(results) > 0 {
		metrics.SearchWithResult.Inc()
	}

	renderTemplate(w, r, "search", map[string]any{
		"Title":   "Search",
		"Query":   q,
		"Results": results,
	})
}

// APISearchHandler returns JSON-formatted search results
func APISearchHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	var timer *prometheus.Timer
	if q != "" {
		metrics.SearchTotal.Inc()
		timer = prometheus.NewTimer(metrics.SearchLatency)
		defer timer.ObserveDuration()
	}

	language := r.URL.Query().Get("language")
	if language == "" {
		language = "en"
	}

	var results []SearchResult
	if q != "" {
		const limit = 10
		const offset = 0

		if useFTSSearch {
			// FTS-powered search when enabled
			ftsQuery := `
SELECT p.id,
       p.title,
       p.url,
       p.language,
       p.content,
       bm25(pages_fts) AS rank
FROM pages_fts
JOIN pages p ON p.id = pages_fts.rowid
WHERE p.language = ? AND pages_fts MATCH ?
ORDER BY rank ASC
LIMIT ? OFFSET ?;`
			rows, err := db.Query(ftsQuery, language, q, limit, offset)
			if err == nil {
				defer func() { _ = rows.Close() }()
				for rows.Next() {
					var it SearchResult
					var rank float64
					if err := rows.Scan(&it.ID, &it.Title, &it.URL, &it.Language, &it.Description, &rank); err == nil {
						results = append(results, it)
					}
				}
			}
		} else {
			rows, err := db.Query(
				`SELECT id, title, url, language, content
			 FROM pages
			 WHERE language = ? AND (title LIKE ? OR content LIKE ?)`,
				language, "%"+q+"%", "%"+q+"%",
			)
			// Error catch for no result
			if err == nil {
				defer func() { _ = rows.Close() }()
				for rows.Next() {
					var it SearchResult
					if err := rows.Scan(&it.ID, &it.Title, &it.URL, &it.Language, &it.Description); err == nil {
						results = append(results, it)
					}
				}
			}
		}
	}

	if len(results) > 0 {
		metrics.SearchWithResult.Inc()
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"search_results": results,
	})
}
