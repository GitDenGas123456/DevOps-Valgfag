package handlers

import (
	"database/sql"
	"encoding/json"
	"html/template"
	"log"
	"net/http"

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

	// Execute template
	err := tmpl.ExecuteTemplate(w, name, data)
	if err != nil {
		// Cannot safely call http.Error if template wrote some content
		log.Println("template exec error:", err)
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
