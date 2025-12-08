package migrate

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
)

func RunMigrations(db *sql.DB) error {
	// Ensure schema_migrations exists
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY
		);
	`)
	if err != nil {
		return fmt.Errorf("failed to ensure schema_migrations: %w", err)
	}

	// Load all .sql files in migrations/
	files, err := filepath.Glob("migrations/*.sql")
	if err != nil {
		return fmt.Errorf("failed to read migrations folder: %w", err)
	}

	for _, file := range files {
		version := filepath.Base(file)
		version = strings.TrimSuffix(version, filepath.Ext(version))

		// Check if applied
		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM schema_migrations WHERE version=$1", version).Scan(&count)
		if err != nil {
			return fmt.Errorf("failed to query schema_migrations: %w", err)
		}

		if count > 0 {
			fmt.Println("Skipping migration:", version)
			continue
		}

		// Load SQL file
		content, err := ioutil.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to read migration %s: %w", file, err)
		}

		// Execute migration
		fmt.Println("Applying migration:", version)
		_, err = db.Exec(string(content))
		if err != nil {
			return fmt.Errorf("migration %s failed: %w", version, err)
		}

		// Mark migration as applied
		_, err = db.Exec("INSERT INTO schema_migrations (version) VALUES ($1)", version)
		if err != nil {
			return fmt.Errorf("failed to insert migration record %s: %w", version, err)
		}
	}

	fmt.Println("All migrations applied successfully.")
	return nil
}
