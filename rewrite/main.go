package main

import (
	"database/sql"
	"encoding/json"
	"html/template"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	_ "modernc.org/sqlite"
)

// ---------- Main ----------
func main() {
	db, err := sql.Open("sqlite", "../whoknows.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Ensure the users table exists (optional safety)
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT NOT NULL UNIQUE,
			email TEXT NOT NULL UNIQUE,
			password TEXT NOT NULL
		)
	`)
	if err != nil {
		log.Fatal("failed to create users table: ", err)
	}

	r := mux.NewRouter()

	// Serve static files (CSS, JS, images)
	fs := http.FileServer(http.Dir("static"))
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", fs))

	// HTML form routes
	r.HandleFunc("/register", RegisterFormHandler).Methods("GET")
	r.HandleFunc("/register", RegisterFormSubmitHandler(db)).Methods("POST")

	// JSON API
	r.HandleFunc("/api/register", RegisterAPIHandler(db)).Methods("POST")

	addr := ":8080"
	log.Printf("Register service running at http://localhost%s", addr)
	log.Fatal(http.ListenAndServe(addr, r))
}

// ---------- Data Models ----------
type RegisterRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// ---------- Handlers ----------

// Show register form (HTML)
func RegisterFormHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("templates/register.html")
	if err != nil {
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, nil)
}

// Handle form POST (HTML form submission)
func RegisterFormSubmitHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid form", http.StatusBadRequest)
			return
		}

		username := r.FormValue("username")
		email := r.FormValue("email")
		password := r.FormValue("password")

		_, err := db.Exec("INSERT INTO users (username, email, password) VALUES (?, ?, ?)",
			username, email, password)
		if err != nil {
			http.Error(w, "database error: "+err.Error(), http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/register?success=1", http.StatusSeeOther)
	}
}

// JSON API endpoint
func RegisterAPIHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req RegisterRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid input", http.StatusBadRequest)
			return
		}

		_, err := db.Exec(
			"INSERT INTO users (username, email, password) VALUES (?, ?, ?)",
			req.Username, req.Email, req.Password,
		)
		if err != nil {
			http.Error(w, "database error: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("User registered successfully"))
	}
}
