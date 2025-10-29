package handlers

import "net/http"

type SearchResult struct {
	ID       int
	Language string
	Content  string
}

func SearchPageHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	language := r.URL.Query().Get("language")
	if language == "" {
		language = "en"
	}

	var results []SearchResult
	if q != "" {
		rows, err := db.Query(
			`SELECT id, language, content FROM pages WHERE language = ? AND content LIKE ?`,
			language, "%"+q+"%",
		)
		if err == nil {
			defer func() { _ = rows.Close() }()
			for rows.Next() {
				var it SearchResult
				if err := rows.Scan(&it.ID, &it.Language, &it.Content); err == nil {
					results = append(results, it)
				}
			}
		}
	}

	renderTemplate(w, "search", map[string]any{
		"Title":   "",
		"Query":   q,
		"Results": results,
	})
}

func APISearchHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	language := r.URL.Query().Get("language")
	if language == "" {
		language = "en"
	}

	var results []SearchResult
	if q != "" {
		rows, err := db.Query(
			`SELECT id, language, content FROM pages WHERE language = ? AND content LIKE ?`,
			language, "%"+q+"%",
		)
		if err == nil {
			defer func() { _ = rows.Close() }()
			for rows.Next() {
				var it SearchResult
				if err := rows.Scan(&it.ID, &it.Language, &it.Content); err == nil {
					results = append(results, it)
				}
			}
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{"search_results": results})
}
