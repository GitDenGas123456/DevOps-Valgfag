package tests

import (
	"database/sql"
	"html/template"
	"testing"

	h "devops-valgfag/handlers"
	"github.com/gorilla/sessions"
	_ "modernc.org/sqlite"
)

// setupTestHandlers creates an in-memory DB, templates, and session store, then wires handlers.Init.
func setupTestHandlers(t *testing.T) *sql.DB {
	t.Helper()

	// In-memory SQLite (only in RAM; disappears after the test)
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open in-memory sqlite: %v", err)
	}

	// Minimal schema (mirrors migrations/0001_create_core_tables.sql)
	schema := `
CREATE TABLE IF NOT EXISTS users (
  id        INTEGER PRIMARY KEY AUTOINCREMENT,
  username  TEXT    NOT NULL UNIQUE,
  email     TEXT    NOT NULL UNIQUE,
  password  TEXT    NOT NULL
);

CREATE TABLE IF NOT EXISTS pages (
  id           INTEGER PRIMARY KEY AUTOINCREMENT,
  title        TEXT    NOT NULL UNIQUE,
  url          TEXT    NOT NULL UNIQUE,
  language     TEXT    NOT NULL CHECK(language IN ('en','da')) DEFAULT 'en',
  last_updated TIMESTAMP,
  content      TEXT    NOT NULL
);
`
	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}

	// Simple templates for tests (no real HTML files).
	tmpl := template.Must(template.New("base").Parse(`
{{define "search"}}search page {{.Query}}{{end}}
{{define "search.html"}}search page {{.Query}}{{end}}

{{define "login"}}login page{{end}}
{{define "login.html"}}login page{{end}}

{{define "register"}}register page{{end}}
{{define "register.html"}}register page{{end}}
`))

	// Test session store
	store := sessions.NewCookieStore([]byte("test-session-key"))

	// Init from handlers/handlers.go (same as main)
	h.Init(db, tmpl, store)

	return db
}
