package db

import (
	"database/sql"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB

func InitDB() {
	var err error
	DB, err = sql.Open("sqlite3", "data/whoknows.db")
	if err != nil {
		log.Fatal("Failed to open DB:", err)
	}

	createTables()
}

func createTables() {
	query := `
CREATE TABLE IF NOT EXISTS external_results (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    query TEXT NOT NULL,
    language TEXT NOT NULL,
    title TEXT,
    url TEXT,
    snippet TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_query_lang_unique
ON external_results (query, language, title);
`
	_, err := DB.Exec(query)
	if err != nil {
		log.Fatal("Failed creating tables:", err)
	}
}

func InsertExternalResult(query, lang, title, url, snippet string) error {
	_, err := DB.Exec(`
INSERT INTO external_results (query, language, title, url, snippet)
VALUES (?, ?, ?, ?, ?)`,
		query, lang, title, url, snippet)

	return err
}

func GetCachedResults(query, lang string) ([]map[string]string, error) {
	rows, err := DB.Query(`
SELECT title, url, snippet
FROM external_results
WHERE query = ? AND language = ?`,
		query, lang)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []map[string]string

	for rows.Next() {
		var title, url, snippet string
		rows.Scan(&title, &url, &snippet)

		results = append(results, map[string]string{
			"Title":   title,
			"URL":     url,
			"Snippet": snippet,
		})
	}

	return results, nil
}