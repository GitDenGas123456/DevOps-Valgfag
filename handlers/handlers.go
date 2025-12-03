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

// Values for database, template and session/cookies
var (
	db           *sql.DB
	tmpl         *template.Template
	sessionStore *sessions.CookieStore
)

// Init wires shared dependencies for this package.
func Init(database *sql.DB, templates *template.Template, store *sessions.CookieStore) {
	db = database
	tmpl = templates
	sessionStore = store
}

// Renders template
func renderTemplate(w http.ResponseWriter, r *http.Request, name string, data map[string]any) {
	// Header
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if data == nil {
		data = map[string]any{}
	}
	if _, ok := data["Title"]; !ok {
		data["Title"] = ""
	}
	data["LoggedIn"] = isAuthenticated(r)

	// Execute template
	err := tmpl.ExecuteTemplate(w, name, data)
	if err != nil {
		// Cannot safely call http.Error if template wrote some content
		log.Println("template exec error:", err)
	}
}

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

// Function parsing struct to JSON
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// SeedDB loads schema.sql into the given DB
func SeedDB(db *sql.DB) error {
	// Use project-root relative path
	raw, err := os.ReadFile("../internal/db/schema.sql")
	if err != nil {
		return err
	}
	_, err = db.Exec(string(raw))
	return err
}
