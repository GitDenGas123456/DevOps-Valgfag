package handlers

import (
	"net/http"
)

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

	renderTemplate(w, "search", map[string]any{
		"Title":   "Search",
		"Query":   q,
		"Results": results,
	})
}

// APISearchHandler returns JSON-formatted search results
func APISearchHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
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

	writeJSON(w, http.StatusOK, map[string]any{"search_results": results})
}
