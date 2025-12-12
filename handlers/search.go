package handlers

import (
	"log"
	"net/http"

	dbx "devops-valgfag/internal/db"
	"devops-valgfag/internal/metrics"
	"devops-valgfag/internal/scraper"

	"github.com/prometheus/client_golang/prometheus"
)

var useFTSSearch bool
var externalEnabled = true

const (
	pageLimit       = 50
	rowsCloseErrMsg = "rows.Close error:"
)

// EnableFTSSearch toggles FTS usage for search endpoints.
func EnableFTSSearch(on bool) {
	useFTSSearch = on
}

// EnableExternalSearch toggles external Wikipedia enrichment.
func EnableExternalSearch(on bool) {
	externalEnabled = on
}

// SearchResult represents a single search result row.
type SearchResult struct {
	ID          int
	Title       string
	URL         string
	Language    string
	Description string
}

// -----------------------------------------------------------------------------
// WEB PAGE SEARCH HANDLER (PostgreSQL-compatible + SQLite compatible)
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

	// BASIC SEARCH (ILIKE replaced with LOWER(...) LIKE LOWER(...) for SQLite compatibility)
	if q != "" {
		rows, err := db.Query(
			`SELECT id, title, url, language, content
			 FROM pages
			 WHERE language = $1 
			   AND (
					LOWER(title) LIKE LOWER($2)
					OR LOWER(content) LIKE LOWER($2)
				   )
			 LIMIT $3`,
			language, "%"+q+"%", pageLimit,
		)
		if err == nil {
			defer func() {
				if err := rows.Close(); err != nil {
					log.Println(rowsCloseErrMsg, err)
				}
			}()

			for rows.Next() {
				var it SearchResult
				if err := rows.Scan(&it.ID, &it.Title, &it.URL, &it.Language, &it.Description); err == nil {
					results = append(results, it)
				}
			}
		}

		if externalEnabled {
			if !dbx.ExternalExists(db, q, language) {
				scraped, err := scraper.WikipediaSearch(q, 10)
				if err != nil {
					log.Println("WikipediaSearch error:", err)
				} else if len(scraped) > 0 {
					store := []dbx.ExternalResult{}
					for _, s := range scraped {
						store = append(store, dbx.ExternalResult{
							Title:   s.Title,
							URL:     s.URL,
							Snippet: s.Snippet,
						})
					}
					if err := dbx.InsertExternal(db, q, language, store); err != nil {
						log.Println("InsertExternal error:", err)
					}
				}
			}

			external, err := dbx.GetExternal(db, q, language)
			if err == nil {
				for _, e := range external {
					results = append(results, SearchResult{
						ID:          0,
						Title:       e.Title,
						URL:         e.URL,
						Language:    language,
						Description: e.Snippet,
					})
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
// API SEARCH HANDLER (PostgreSQL-compatible + SQLite compatible)
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
		// FTS SEARCH (unchanged)
		// ---------------------------------------------------------------------
		if useFTSSearch {
			ftsQuery := `
				WITH q AS (SELECT plainto_tsquery($2) AS query)
				SELECT id, title, url, language, content
				FROM pages, q
				WHERE language = $1
				  AND content_tsv @@ q.query
				ORDER BY ts_rank(content_tsv, q.query) DESC
				LIMIT $3 OFFSET $4;
			`

			rows, err := db.Query(ftsQuery, language, q, limit, offset)
			if err != nil {
				// Fallback: LOWER(...) LIKE LOWER(...)
				log.Println("FTS search error, falling back to LIKE:", err)
				rows, err = db.Query(
					`SELECT id, title, url, language, content
					 FROM pages
					 WHERE language = $1
					   AND (
							LOWER(title) LIKE LOWER($2)
							OR LOWER(content) LIKE LOWER($2)
						   )
					 LIMIT $3 OFFSET $4`,
					language, "%"+q+"%", limit, offset,
				)
			}

			if err == nil {
				defer func() {
					if err := rows.Close(); err != nil {
						log.Println(rowsCloseErrMsg, err)
					}
				}()
				for rows.Next() {
					var it SearchResult
					if err := rows.Scan(&it.ID, &it.Title, &it.URL, &it.Language, &it.Description); err == nil {
						results = append(results, it)
					}
				}
			} else {
				log.Println("FTS + fallback LIKE search failed:", err)
			}

		} else {
			// -----------------------------------------------------------------
			// BASIC SEARCH (ILIKE replaced with LOWER(...) LIKE LOWER(...))
			// -----------------------------------------------------------------
			rows, err := db.Query(
				`SELECT id, title, url, language, content
				 FROM pages
				 WHERE language = $1
				   AND (
						LOWER(title) LIKE LOWER($2)
						OR LOWER(content) LIKE LOWER($2)
					   )
				 LIMIT $3 OFFSET $4`,
				language, "%"+q+"%", limit, offset,
			)

			if err == nil {
				defer func() {
					if err := rows.Close(); err != nil {
						log.Println(rowsCloseErrMsg, err)
					}
				}()
				for rows.Next() {
					var it SearchResult
					if err := rows.Scan(&it.ID, &it.Title, &it.URL, &it.Language, &it.Description); err == nil {
						results = append(results, it)
					}
				}
			} else {
				log.Println("Basic LIKE search failed:", err)
			}
		}

		if externalEnabled {
			if !dbx.ExternalExists(db, q, language) {
				scraped, err := scraper.WikipediaSearch(q, 10)
				if err != nil {
					log.Println("WikipediaSearch error:", err)
				} else if len(scraped) > 0 {
					store := []dbx.ExternalResult{}
					for _, s := range scraped {
						store = append(store, dbx.ExternalResult{
							Title:   s.Title,
							URL:     s.URL,
							Snippet: s.Snippet,
						})
					}
					if err := dbx.InsertExternal(db, q, language, store); err != nil {
						log.Println("InsertExternal error:", err)
					}
				}
			}

			external, err := dbx.GetExternal(db, q, language)
			if err != nil {
				log.Println("GetExternal error:", err)
			} else {
				for _, e := range external {
					results = append(results, SearchResult{
						ID:          0,
						Title:       e.Title,
						URL:         e.URL,
						Language:    language,
						Description: e.Snippet,
					})
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
