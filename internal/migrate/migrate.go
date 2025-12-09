package migrate

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// RunMigrations applies all pending .sql migrations found in the migrations/ folder.
// Each migration is executed in a transaction and recorded in schema_migrations.
func RunMigrations(db *sql.DB) error {
	// Ensure schema_migrations exists
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version VARCHAR(255) PRIMARY KEY
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

	// Ensure deterministic order (e.g. 0001_..., 0002_...)
	sort.Strings(files)

	for _, file := range files {
		version := filepath.Base(file)
		version = strings.TrimSuffix(version, filepath.Ext(version))

		// Check if migration already applied
		var count int
		err = db.QueryRow(
			"SELECT COUNT(*) FROM schema_migrations WHERE version = $1",
			version,
		).Scan(&count)
		if err != nil {
			return fmt.Errorf("failed to query schema_migrations: %w", err)
		}

		if count > 0 {
			fmt.Println("Skipping migration:", version)
			continue
		}

		// Load SQL file
		content, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to read migration %s: %w", file, err)
		}

		fmt.Println("Applying migration:", version)

		// Execute migration in a transaction for atomicity
		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("failed to begin transaction for migration %s: %w", version, err)
		}

		if _, err := tx.Exec(string(content)); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("migration %s failed: %w", version, err)
		}

		if _, err := tx.Exec(
			"INSERT INTO schema_migrations (version) VALUES ($1)",
			version,
		); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("failed to insert migration record %s: %w", version, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit transaction for migration %s: %w", version, err)
		}
	}

	fmt.Println("All migrations applied successfully.")
	return nil
}
