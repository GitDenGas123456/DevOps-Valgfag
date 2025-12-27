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

// Feature flags toggled at startup (typically from env vars in main).
// atomic.Bool allows safe concurrent reads from HTTP handlers without locks.
var useFTSSearch atomic.Bool    // Prefer PostgreSQL FTS over ILIKE when enabled.
var externalEnabled atomic.Bool // Allow optional Wikipedia enrichment (disabled in tests/CI for determinism).

func init() {
	// Default behavior: allow external enrichment.
	// Tests/CI can disable this with EnableExternalSearch(false).
	externalEnabled.Store(true)
}

const (
	// UI vs API limits: UI is for humans (more results), API is for machines (smaller payload).
	pageLimit = 50
	apiLimit  = 10

	// Upper bound on search execution time (primarily DB calls via QueryContext).
	requestTimeout = 2 * time.Second

	// Max length of snippet text returned per result.
	snippetLen = 200

	rowsCloseErrMsg = "rows.Close error:"
)

// EnableFTSSearch toggles PostgreSQL full-text search (FTS) usage.
// When enabled, queryLocal() tries FTS first and falls back to ILIKE if needed.
func EnableFTSSearch(on bool) {
	useFTSSearch.Store(on)
}

// EnableExternalSearch toggles external Wikipedia enrichment.
// Keep this OFF in tests to avoid network calls and flaky CI.
func EnableExternalSearch(on bool) {
	externalEnabled.Store(on)
}

// SearchResult is the normalized result shape used by both UI and API.
// Local DB results use a real ID; external cached results set ID=0.
type SearchResult struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	URL         string `json:"url"`
	Language    string `json:"language"`
	Description string `json:"description"` // Snippet (local content or external snippet)
}

// APISearchResponse is the stable JSON contract returned by /api/search.
type APISearchResponse struct {
	SearchResults []SearchResult `json:"search_results"`
}

// HomePageHandler renders the landing page.
// If the user provides a query (?q=...), we redirect to /search so search logic lives in one place.
func HomePageHandler(w http.ResponseWriter, r *http.Request) {
	if q := r.URL.Query().Get("q"); q != "" {
		target := "/search"
		if raw := r.URL.RawQuery; raw != "" {
			target += "?" + raw
		}
		http.Redirect(w, r, target, http.StatusFound)
		return
	}

	// Empty search page (no query, no results).
	renderTemplate(w, r, "search", map[string]any{
		"Title":   "Home",
		"Query":   "",
		"Results": []SearchResult{},
	})
}

// -----------------------------------------------------------------------------
// WEB PAGE SEARCH HANDLER
// -----------------------------------------------------------------------------

// SearchPageHandler serves HTML search results for the web UI.
// It allows optional external enrichment to improve user experience.
func SearchPageHandler(w http.ResponseWriter, r *http.Request) {
	// Defensive check: avoid nil pointer panics if DB wiring/configuration fails.
	if db == nil {
		http.Error(w, "database not configured", http.StatusInternalServerError)
		return
	}

	q := r.URL.Query().Get("q")
	lang := getLanguage(r)

	// Shared search pipeline (UI settings: pageLimit + includeExternal).
	results := runSearch(r, q, lang, pageLimit, true)

	// Used for calculating "hit rate" (searches that return at least one result).
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
// @Description  Search stored pages (local database). Requires session auth.
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

	// API requires an authenticated session
	if !isAuthenticated(r) {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "unauthorized"})
		return
	}

	q := r.URL.Query().Get("q")
	lang := getLanguage(r)

	// API settings: smaller limit + no external enrichment for predictability and stability.
	results := runSearch(r, q, lang, apiLimit, false)

	if len(results) > 0 {
		metrics.SearchWithResult.Inc()
	}

	writeJSON(w, http.StatusOK, APISearchResponse{SearchResults: results})
}

// -----------------------------------------------------------------------------
// Shared search runner (metrics + timeout + best-effort behavior)
// -----------------------------------------------------------------------------

// runSearch is the shared search pipeline used by both UI and API.
// It handles:
//   - input sanitization
//   - metrics (count + latency)
//   - request-scoped timeout
//   - local DB search (FTS preferred, ILIKE fallback)
//   - optional external enrichment
//   - final result capping for predictable response sizes
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

	// Optional enrichment: only for UI and only if enabled.
	if includeExternal && externalEnabled.Load() {
		ext := loadExternalBestEffort(q, lang)
		local = append(local, ext...)
	}

	// Enforce final cap (external results should not expand response beyond the configured limit).
	if len(local) > limit {
		local = local[:limit]
	}

	return local
}

// -----------------------------------------------------------------------------
// Local DB search (FTS preferred + fallback)
// -----------------------------------------------------------------------------

// queryLocal performs the local DB search.
// If FTS is enabled, it tries FTS first and falls back to ILIKE if we get a FTS error.
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

// queryFTS performs ranked PostgreSQL full-text search against pages.content_tsv.
// NOTE: 'simple' config matches the migration that builds content_tsv using to_tsvector('simple', ...).
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

// queryILIKE is a simple substring search fallback.
// It is used when FTS is disabled or unavailable (e.g., missing migration/index).
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

// scanRows converts SQL rows to []SearchResult and guarantees rows.Close() is called.
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
// External enrichment (Wikipedia)
// -----------------------------------------------------------------------------

// loadExternalBestEffort returns cached Wikipedia results for (query, lang).
// If no cache exists, it performs a scrape and stores results in the DB.
// Failures are logged but do not fail the request (best-effort enrichment).
func loadExternalBestEffort(q, lang string) []SearchResult {
	// Ensure cache exists (best effort).
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

// getLanguage reads the requested language code.
// Default is "en" for predictable behavior.
func getLanguage(r *http.Request) string {
	lang := r.URL.Query().Get("language")
	if lang == "" {
		return "en"
	}
	return lang
}

// Silence unused import warning if sql is referenced by build tags elsewhere.
var _ = sql.ErrNoRows
