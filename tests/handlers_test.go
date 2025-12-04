package tests

import (
	"database/sql"
	"html/template"
	"testing"

	"github.com/gorilla/sessions"
	_ "modernc.org/sqlite"

	h "devops-valgfag/handlers"
)

// setupTestHandlers laver en test-db, templates og session store,
// og kalder h.Init, så handlers er klar til brug i tests.
func setupTestHandlers(t *testing.T) *sql.DB {
	t.Helper()

	// In-memory SQLite (kun i RAM, forsvinder efter test)
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open in-memory sqlite: %v", err)
	}

	// Minimal schema (samme som migrations/0001_create_core_tables.sql)
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

	// Enkle templates – bare tekst, ikke rigtige html-filer.
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

	// Init fra handlers/handlers.go – samme som main gør
	h.Init(db, tmpl, store)

	return db
}

// Ensure db is closed after test
func closeDB(t *testing.T, db *sql.DB) {
	t.Helper()
	if err := db.Close(); err != nil {
		t.Fatalf("failed to close db: %v", err)
	}
}
