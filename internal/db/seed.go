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
	return nil
}
