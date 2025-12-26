package handlers

import (
	"database/sql"
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/sessions"
)

// Shared dependencies injected from main.go.
// These are package-level variables to avoid passing
// database, templates, and session store through every handler.
var (
	db           *sql.DB
	tmpl         *template.Template
	sessionStore *sessions.CookieStore
)

// Init injects shared dependencies into the handlers package.
//
// It must be called once from main.go during application startup.
// This avoids global initialization logic and keeps handlers testable.
func Init(database *sql.DB, templates *template.Template, store *sessions.CookieStore) {
	db = database
	tmpl = templates
	sessionStore = store
}

// renderTemplate executes an HTML template with common default data.
//
// It ensures that:
// - Content-Type is set correctly
// - Title always exists
// - Authentication state is available to templates
//
// This function is internal to the handlers package.
func renderTemplate(w http.ResponseWriter, r *http.Request, name string, data map[string]any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if data == nil {
		data = map[string]any{}
	}
	if _, ok := data["Title"]; !ok {
		data["Title"] = ""
	}
	data["LoggedIn"] = isAuthenticated(r)

	if err := tmpl.ExecuteTemplate(w, name, data); err != nil {
		// Cannot safely call http.Error if template wrote some content
		log.Println("template exec error:", err)
	}
}

// isAuthenticated checks whether the current request
// belongs to a logged-in user by inspecting the session.
func isAuthenticated(r *http.Request) bool {
	if sessionStore == nil {
		return false
	}
	sess, err := sessionStore.Get(r, "session")
	if err != nil {
		return false
	}
	_, ok := sess.Values["user_id"]
	return ok
}

// writeJSON writes a JSON response with the given HTTP status code.
//
// It is used by API handlers to return structured JSON responses.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// InitSchema initializes the database schema for tests/CI by executing schema.sql.
// It creates tables/indexes but does not insert demo data.
//
// Note: tests run with working directory = ./tests, so the relative path is correct.
func InitSchema(database *sql.DB) error {
	raw, err := os.ReadFile("../internal/db/schema.sql")
	if err != nil {
		return err
	}
	_, err = database.Exec(string(raw))
	return err
}

// SeedDB is kept for backward compatibility (older tests/branches may still call it).
// Prefer InitSchema going forward.
func SeedDB(database *sql.DB) error {
	return InitSchema(database)
}