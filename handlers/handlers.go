package handlers

// Imports
import (
	"database/sql"
	"encoding/json"
	"html/template"
	"net/http"
	"os"

	"github.com/gorilla/sessions"
)

// values for database, template and session/cookies
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

// renders template
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
	if err := tmpl.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, "template exec error: "+err.Error(), http.StatusInternalServerError)
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

// function parsing string to JSON
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
