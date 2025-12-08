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

// -----------------------------------------------------------------------------
// WEB PAGE SEARCH HANDLER (PostgreSQL-compatible)
// -----------------------------------------------------------------------------
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

	// BASIC SEARCH (ILIKE for PostgreSQL)
	if q != "" {
		rows, err := db.Query(
			`SELECT id, title, url, language, content
			 FROM pages
			 WHERE language = $1 
			   AND (title ILIKE $2 OR content ILIKE $2)`,
			language, "%"+q+"%",
		)

		if err == nil {
			defer rows.Close()

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

// -----------------------------------------------------------------------------
// API SEARCH HANDLER (PostgreSQL-compatible)
// -----------------------------------------------------------------------------
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

		// ---------------------------------------------------------------------
		// FTS SEARCH (PostgreSQL content_tsv)
		// ---------------------------------------------------------------------
		if useFTSSearch {
			ftsQuery := `
				SELECT id, title, url, language, content
				FROM pages
				WHERE language = $1
				  AND content_tsv @@ plainto_tsquery($2)
				ORDER BY ts_rank(content_tsv, plainto_tsquery($2)) DESC
				LIMIT $3 OFFSET $4;
			`

			rows, err := db.Query(ftsQuery, language, q, limit, offset)
			if err == nil {
				defer rows.Close()
				for rows.Next() {
					var it SearchResult
					if err := rows.Scan(&it.ID, &it.Title, &it.URL, &it.Language, &it.Description); err == nil {
						results = append(results, it)
					}
				}
			}

		} else {
			// -----------------------------------------------------------------
			// BASIC SEARCH
			// -----------------------------------------------------------------
			rows, err := db.Query(
				`SELECT id, title, url, language, content
				 FROM pages
				 WHERE language = $1
				   AND (title ILIKE $2 OR content ILIKE $2)
				 LIMIT $3 OFFSET $4`,
				language, "%"+q+"%", limit, offset,
			)

			if err == nil {
				defer rows.Close()
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
