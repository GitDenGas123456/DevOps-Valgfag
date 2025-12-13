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
	if err := ensureSchemaMigrationsTable(db); err != nil {
		return err
	}

	files, err := loadMigrationFiles("migrations")
	if err != nil {
		return err
	}

	for _, file := range files {
		version := migrationVersionFromFile(file)

		applied, err := migrationApplied(db, version)
		if err != nil {
			return err
		}
		if applied {
			fmt.Println("Skipping migration:", version)
			continue
		}

		if err := applyMigrationFile(db, version, file); err != nil {
			return err
		}
	}

	fmt.Println("All migrations applied successfully.")
	return nil
}

// ensureSchemaMigrationsTable makes sure the schema_migrations table exists.
func ensureSchemaMigrationsTable(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version VARCHAR(255) PRIMARY KEY
		);
	`)
	if err != nil {
		return fmt.Errorf("failed to ensure schema_migrations: %w", err)
	}
	return nil
}

// loadMigrationFiles returns a sorted list of .sql migration files in the given directory.
func loadMigrationFiles(dir string) ([]string, error) {
	pattern := filepath.Join(dir, "*.sql")

	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to read migrations folder: %w", err)
	}

	// Ensure deterministic order (e.g. 0001_..., 0002_...)
	sort.Strings(files)
	return files, nil
}

// migrationVersionFromFile extracts the migration version from the filename.
func migrationVersionFromFile(path string) string {
	base := filepath.Base(path)
	return strings.TrimSuffix(base, filepath.Ext(base))
}

// migrationApplied checks whether a given migration version is already recorded.
func migrationApplied(db *sql.DB, version string) (bool, error) {
	var count int
	err := db.QueryRow(
		"SELECT COUNT(*) FROM schema_migrations WHERE version = $1",
		version,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to query schema_migrations: %w", err)
	}
	return count > 0, nil
}

// applyMigrationFile runs a single migration file inside a transaction and records it.
func applyMigrationFile(db *sql.DB, version, file string) error {
	content, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("failed to read migration %s: %w", file, err)
	}

	fmt.Println("Applying migration:", version)

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

	return nil
}
