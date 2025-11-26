package handlers

import (
	"database/sql"
	"encoding/json"
	"html/template"
	"net/http"
	"os"

	"github.com/gorilla/sessions"
)

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

func renderTemplate(w http.ResponseWriter, name string, data map[string]any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if data == nil {
		data = map[string]any{}
	}
	if _, ok := data["Title"]; !ok {
		data["Title"] = ""
	}
	if err := tmpl.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, "template exec error: "+err.Error(), http.StatusInternalServerError)
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// SeedDB loads schema.sql into the given DB
func SeedDB(db *sql.DB) error {
	// Use project-root relative path
	raw, err := os.ReadFile("../internal/db/schema.sql") // <- adjust as needed
	if err != nil {
		return err
	}
	_, err = db.Exec(string(raw))
	return err
}
