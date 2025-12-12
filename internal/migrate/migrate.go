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

	statements := splitSQLStatements(string(content))
	for _, stmt := range statements {
		if _, err := tx.Exec(stmt); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("migration %s failed: %w", version, err)
		}
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

// splitSQLStatements breaks a migration file into individual statements.
// It keeps semicolons inside single quotes and dollar-quoted blocks intact.
func splitSQLStatements(content string) []string {
	var (
		statements []string
		buf        strings.Builder
		inSingle   bool
		inDollar   bool
		dollarTag  string
	)

	for i := 0; i < len(content); i++ {
		ch := content[i]

		// Handle dollar-quoted strings: $$...$$ or $tag$...$tag$
		if !inSingle && ch == '$' {
			tagEnd := i + 1
			for tagEnd < len(content) && isDollarTagChar(content[tagEnd]) {
				tagEnd++
			}
			if tagEnd < len(content) && content[tagEnd] == '$' {
				tag := content[i+1 : tagEnd]
				if inDollar && tag == dollarTag {
					inDollar = false
					dollarTag = ""
				} else if !inDollar {
					inDollar = true
					dollarTag = tag
				}
				for j := i; j <= tagEnd; j++ {
					buf.WriteByte(content[j])
				}
				i = tagEnd
				continue
			}
		}

		// Handle single-quoted strings, respecting escaped quotes.
		if !inDollar && ch == '\'' {
			buf.WriteByte(ch)
			if inSingle {
				if i+1 < len(content) && content[i+1] == '\'' {
					buf.WriteByte(content[i+1])
					i++
					continue
				}
				inSingle = false
			} else {
				inSingle = true
			}
			continue
		}

		// Statement boundary: semicolon outside strings/blocks.
		if ch == ';' && !inSingle && !inDollar {
			stmt := strings.TrimSpace(buf.String())
			if stmt != "" {
				statements = append(statements, stmt)
			}
			buf.Reset()
			continue
		}

		buf.WriteByte(ch)
	}

	if tail := strings.TrimSpace(buf.String()); tail != "" {
		statements = append(statements, tail)
	}

	return statements
}

func isDollarTagChar(b byte) bool {
	return b == '_' ||
		(b >= 'a' && b <= 'z') ||
		(b >= 'A' && b <= 'Z') ||
		(b >= '0' && b <= '9')
}
