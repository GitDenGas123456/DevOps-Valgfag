package db

import (
	"database/sql"
	"log"
)

type ExternalResult struct {
	Title   string
	URL     string
	Snippet string
}

// ExternalExists checks if results already exist for a query+language.
func ExternalExists(database *sql.DB, query, language string) bool {
	var count int
	err := database.QueryRow(
		`SELECT COUNT(*) FROM external_results WHERE query = ? AND language = ?`,
		query, language,
	).Scan(&count)
	if err != nil {
		log.Println("ExternalExists error:", err)
		return false
	}
	return count > 0
}

// InsertExternal saves scraped results to the database.
func InsertExternal(database *sql.DB, query, lang string, items []ExternalResult) error {
	if len(items) == 0 {
		return nil
	}

	tx, err := database.Begin()
	if err != nil {
		return err
	}





	
	stmt, err := tx.Prepare(`
INSERT OR IGNORE INTO external_results (query, language, title, url, snippet)
VALUES (?, ?, ?, ?, ?)`)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	defer func() {
		_ = stmt.Close()
	}()

	for _, r := range items {
		if _, err := stmt.Exec(query, lang, r.Title, r.URL, r.Snippet); err != nil {
			log.Println("InsertExternal exec error:", err)
			_ = tx.Rollback()
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

// GetExternal loads external results from the database.
func GetExternal(database *sql.DB, query, lang string) ([]ExternalResult, error) {
	rows, err := database.Query(
		`SELECT title, url, snippet
         FROM external_results
         WHERE query = ? AND language = ?`,
		query, lang,
	)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	var all []ExternalResult
	for rows.Next() {
		var r ExternalResult
		if err := rows.Scan(&r.Title, &r.URL, &r.Snippet); err != nil {
			log.Println("GetExternal scan error:", err)
			continue
		}
		all = append(all, r)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return all, nil
}
