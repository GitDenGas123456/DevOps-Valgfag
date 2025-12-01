package handlers

import (
	"log"
	"net/http"

	dbx "devops-valgfag/internal/db"
	"devops-valgfag/internal/scraper"
)

type SearchResult struct {
	ID       int
	Language string
	Title    string
	URL      string
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
		// Stage 1 — Search local pages table
		rows, err := db.Query(
			`SELECT id, language, title, url, content FROM pages WHERE language = ? AND content LIKE ?`,
			language, "%"+q+"%",
		)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var r SearchResult
				if err := rows.Scan(&r.ID, &r.Language, &r.Title, &r.URL, &r.Content); err == nil {
					results = append(results, r)
				}
			}
		}

		// Stage 2 — Wikipedia search if not in external_results
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

		// Stage 3 — Load external results from DB
		external, err := dbx.GetExternal(db, q, language)
		if err == nil {
			for _, e := range external {
				results = append(results, SearchResult{
					ID:       0,
					Language: language,
					Title:    e.Title,
					URL:      e.URL,
					Content:  e.Snippet,
				})
			}
		}
	}

	renderTemplate(w, "search", map[string]any{
		"Title":   "Search",
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
		// Local pages
		rows, err := db.Query(
			`SELECT id, language, title, url, content FROM pages WHERE language = ? AND content LIKE ?`,
			language, "%"+q+"%",
		)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var r SearchResult
				if err := rows.Scan(&r.ID, &r.Language, &r.Title, &r.URL, &r.Content); err == nil {
					results = append(results, r)
				}
			}
		}

		// Wikipedia results from cache
		external, _ := dbx.GetExternal(db, q, language)
		for _, e := range external {
			results = append(results, SearchResult{
				ID:       0,
				Language: language,
				Title:    e.Title,
				URL:      e.URL,
				Content:  e.Snippet,
			})
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{"search_results": results})
}
