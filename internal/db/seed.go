// NOTE:
// This seeding helper is **SQLite-only** and intended for local demos/tests.
// It uses SQLite-specific syntax (e.g. INSERT OR IGNORE) and is **not**
// called from the PostgreSQL runtime code path.
package db

import (
	"bufio"
	"bytes"
	"database/sql"
	"os"
	"strings"
)

// Seed loads internal/db/schema.sql (if present) and applies statements.
// Keep gated by env to avoid unintended data changes.
func Seed(database *sql.DB) error {
	raw, err := os.ReadFile("internal/db/schema.sql")
	if err != nil {
		// If schema is missing, do nothing.
		return nil
	}

	appEnv := strings.ToLower(strings.TrimSpace(os.Getenv("APP_ENV")))
	if appEnv == "prod" || appEnv == "production" {
		// Skip seeding in production / Postgres runtime
		return nil
	}

	// Parse SQL file
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	var b strings.Builder
	for scanner.Scan() {
		line := scanner.Text()
		t := strings.TrimSpace(line)
		if t == "" || strings.HasPrefix(t, "--") {
			continue
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}
	if err := scanner.Err(); err != nil {
		return err
	}

	// Execute schema
	stmts := strings.Split(b.String(), ";")
	for _, s := range stmts {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if _, err := database.Exec(s); err != nil {
			return err
		}
	}

	// No default admin user seeded anymore – avoids hard-coded bcrypt hash.

	return nil
}
