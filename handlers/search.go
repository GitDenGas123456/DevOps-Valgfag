package handlers

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	dbx "devops-valgfag/internal/db"
	"devops-valgfag/internal/metrics"
	"devops-valgfag/internal/scraper"

	"github.com/prometheus/client_golang/prometheus"
)

var useFTSSearch atomic.Bool
var externalEnabled atomic.Bool

func init() {
	externalEnabled.Store(true)
}

const (
	pageLimit       = 50
	apiLimit        = 10
	requestTimeout  = 2 * time.Second
	snippetLen      = 200
	rowsCloseErrMsg = "rows.Close error:"
)

// EnableFTSSearch toggles FTS usage for search endpoints.
func EnableFTSSearch(on bool) { 
	useFTSSearch.Store(on) 
}

// EnableExternalSearch toggles external Wikipedia enrichment.
func EnableExternalSearch(on bool) { 
	externalEnabled.Store(on) 
}

// SearchResult represents a single search result row.
type SearchResult struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	URL         string `json:"url"`
	Language    string `json:"language"`
	Description string `json:"description"`
}

// APISearchResponse is the JSON payload returned by the search API.
type APISearchResponse struct {
	SearchResults []SearchResult `json:"search_results"`
}

// HomePageHandler renders the landing page and redirects searches to /search.
func HomePageHandler(w http.ResponseWriter, r *http.Request) {
	if q := r.URL.Query().Get("q"); q != "" {
		target := "/search"
		if raw := r.URL.RawQuery; raw != "" {
			target += "?" + raw
		}
		http.Redirect(w, r, target, http.StatusFound)
		return
	}

	renderTemplate(w, r, "search", map[string]any{
		"Title":   "Home",
		"Query":   "",
		"Results": []SearchResult{},
	})
}

// -----------------------------------------------------------------------------
// WEB PAGE SEARCH HANDLER
// -----------------------------------------------------------------------------
func SearchPageHandler(w http.ResponseWriter, r *http.Request) {
	if db == nil {
		http.Error(w, "database not configured", http.StatusInternalServerError)
		return
	}

	q := r.URL.Query().Get("q")
	lang := getLanguage(r)

	results := runSearch(r, q, lang, pageLimit, true)

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
// API SEARCH HANDLER
// -----------------------------------------------------------------------------
// APISearchHandler godoc
// @Summary      Search content
// @Description  Search stored pages and cached external results.
// @Tags         Search
// @Produce      json
// @Param        q          query  string  false  "Search query"
// @Param        language   query  string  false  "Language code (default en)"
// @Success      200  {object}  APISearchResponse  "Search results"
// @Router       /api/search [get]
func APISearchHandler(w http.ResponseWriter, r *http.Request) {
	if db == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "database not configured"})
		return
	}

	// Enforce session auth for API (matches your Postman negative test).
	if !isAuthenticated(r) {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "unauthorized"})
		return
	}

	q := r.URL.Query().Get("q")
	lang := getLanguage(r)

	results := runSearch(r, q, lang, apiLimit, false)

	if len(results) > 0 {
		metrics.SearchWithResult.Inc()
	}

	writeJSON(w, http.StatusOK, APISearchResponse{SearchResults: results})
}

// -----------------------------------------------------------------------------
// Shared search runner (metrics + timeout + best-effort behavior)
// -----------------------------------------------------------------------------
func runSearch(r *http.Request, q, lang string, limit int, includeExternal bool) []SearchResult {
	q = strings.TrimSpace(q)
	if q == "" {
		return []SearchResult{}
	}

	metrics.SearchTotal.Inc()
	timer := prometheus.NewTimer(metrics.SearchLatency)
	defer timer.ObserveDuration()

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	local, err := queryLocal(ctx, q, lang, limit)
	if err != nil {
		log.Println("search local error:", err)
		local = []SearchResult{}
	}

	if includeExternal && externalEnabled.Load() {
		ext := loadExternalBestEffort(q, lang)
		local = append(local, ext...)
	}
	if len(local) > limit {
		local = local[:limit]
	}

	return local
}

// -----------------------------------------------------------------------------
// Local DB search (FTS preferred + fallback)
// -----------------------------------------------------------------------------
func queryLocal(ctx context.Context, q, lang string, limit int) ([]SearchResult, error) {
	if useFTSSearch.Load() {
		res, err := queryFTS(ctx, q, lang, limit)
		if err == nil {
			return res, nil
		}
		log.Println("FTS search error, falling back to ILIKE:", err)
	}
	return queryILIKE(ctx, q, lang, limit)
}

func queryFTS(ctx context.Context, q, lang string, limit int) ([]SearchResult, error) {
	const sqlFTS = `
WITH qq AS (SELECT plainto_tsquery('simple', $2) AS query)
SELECT id, title, url, language, LEFT(content, $3) AS snippet
FROM pages, qq
WHERE language = $1
  AND content_tsv @@ qq.query
ORDER BY ts_rank(content_tsv, qq.query) DESC, id DESC
LIMIT $4;`

	rows, err := db.QueryContext(ctx, sqlFTS, lang, q, snippetLen, limit)
	if err != nil {
		return nil, err
	}
	return scanRows(rows)
}

func queryILIKE(ctx context.Context, q, lang string, limit int) ([]SearchResult, error) {
	const sqlILIKE = `
SELECT id, title, url, language, LEFT(content, $3) AS snippet
FROM pages
WHERE language = $1
  AND (title ILIKE $2 OR content ILIKE $2)
ORDER BY last_updated DESC NULLS LAST, id DESC
LIMIT $4;`

	rows, err := db.QueryContext(ctx, sqlILIKE, lang, "%"+q+"%", snippetLen, limit)
	if err != nil {
		return nil, err
	}
	return scanRows(rows)
}

func scanRows(rows *sql.Rows) ([]SearchResult, error) {
	defer func() {
		if err := rows.Close(); err != nil {
			log.Println(rowsCloseErrMsg, err)
		}
	}()

	out := make([]SearchResult, 0, 16)
	for rows.Next() {
		var it SearchResult
		if err := rows.Scan(&it.ID, &it.Title, &it.URL, &it.Language, &it.Description); err != nil {
			log.Println("rows.Scan error:", err)
			continue
		}
		out = append(out, it)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// -----------------------------------------------------------------------------
// External enrichment (Wikipedia) - best effort + cache
// -----------------------------------------------------------------------------
func loadExternalBestEffort(q, lang string) []SearchResult {
	// ensure cache (best effort)
	if !dbx.ExternalExists(db, q, lang) {
		scraped, err := scraper.WikipediaSearch(q, 10)
		if err != nil {
			log.Println("WikipediaSearch error:", err)
		} else if len(scraped) > 0 {
			store := make([]dbx.ExternalResult, 0, len(scraped))
			for _, s := range scraped {
				store = append(store, dbx.ExternalResult{
					Title:   s.Title,
					URL:     s.URL,
					Snippet: s.Snippet,
				})
			}
			if err := dbx.InsertExternal(db, q, lang, store); err != nil {
				log.Println("InsertExternal error:", err)
			}
		}
	}

	ext, err := dbx.GetExternal(db, q, lang)
	if err != nil {
		log.Println("GetExternal error:", err)
		return nil
	}

	out := make([]SearchResult, 0, len(ext))
	for _, e := range ext {
		out = append(out, SearchResult{
			ID:          0,
			Title:       e.Title,
			URL:         e.URL,
			Language:    lang,
			Description: e.Snippet,
		})
	}
	return out
}

func getLanguage(r *http.Request) string {
	lang := r.URL.Query().Get("language")
	if lang == "" {
		return "en"
	}
	return lang
}

// Silence unused import warning if sql is referenced by build tags elsewhere.
var _ = sql.ErrNoRows
