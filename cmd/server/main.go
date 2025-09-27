package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	_ "modernc.org/sqlite"
	"golang.org/x/crypto/bcrypt"
)

var (
	// TODO: move these into a small app/context struct later
	db           *sql.DB
	tmpl         *template.Template
	sessionStore *sessions.CookieStore
)

// --- Models (move to internal/models later) ---

type User struct {
	ID       int
	Username string
	Email    string
	Password string // bcrypt hash
}

type SearchResult struct {
	ID       int
	Language string
	Content  string
}

// --- main ---

func main() {
	port := getenv("PORT", "8080")
	dbPath := getenv("DATABASE_PATH", "internal/db/whoknows.db")
	sessionKey := getenv("SESSION_KEY", "development key") // dev fallback

	// DB
	var err error
	db, err = sql.Open("sqlite", dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Templates
	// Expect files like:
	//   templates/layout.html            (defines {{define "base"}} ... {{block "content" .}}{{end}} ... {{end}})
	//   templates/page-search.html       (defines "search" + overrides "content")
	//   templates/page-about.html        (defines "about"  + overrides "content")
	//   templates/page-login.html        (defines "login"  + overrides "content")
	//   templates/page-register.html     (defines "register" + overrides "content")
	tmpl = template.Must(template.ParseGlob("./templates/*.html"))

	// Sessions
	sessionStore = sessions.NewCookieStore([]byte(sessionKey))

	// Router
	r := mux.NewRouter()

	// Static
	fs := http.FileServer(http.Dir("static"))
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", fs))

	// Pages
	r.HandleFunc("/", SearchPageHandler).Methods("GET")
	r.HandleFunc("/about", AboutPageHandler).Methods("GET")
	r.HandleFunc("/login", LoginPageHandler).Methods("GET")
	r.HandleFunc("/register", RegisterPageHandler).Methods("GET")

	// API
	r.HandleFunc("/api/login", APILoginHandler).Methods("POST")
	r.HandleFunc("/api/register", APIRegisterHandler).Methods("POST")
	r.HandleFunc("/api/logout", APILogoutHandler).Methods("POST", "GET")
	r.HandleFunc("/api/search", APISearchHandler).Methods("POST")

	fmt.Printf("Server running on :%s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}

// =====================
// Page Handlers
// =====================

func SearchPageHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	language := r.URL.Query().Get("language")
	if language == "" {
		language = "en"
	}

	var results []SearchResult
	if q != "" {
		rows, err := db.Query(
			`SELECT id, language, content FROM pages WHERE language = ? AND content LIKE ?`,
			language, "%"+q+"%",
		)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var it SearchResult
				if err := rows.Scan(&it.ID, &it.Language, &it.Content); err == nil {
					results = append(results, it)
				}
			}
		}
	}

	// Execute the WRAPPER template "search"
	renderTemplate(w, "search", map[string]any{
		"Title":   "",
		"Query":   q,
		"Results": results,
	})
}

func AboutPageHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, "about", map[string]any{
		"Title": "About",
	})
}

func LoginPageHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, "login", map[string]any{
		"Title": "Sign In",
	})
}

func RegisterPageHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, "register", map[string]any{
		"Title": "Sign Up",
	})
}

// =====================
// API Handlers
// =====================

func APISearchHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	language := r.URL.Query().Get("language")
	if language == "" {
		language = "en"
	}

	var results []SearchResult
	if q != "" {
		rows, err := db.Query(
			`SELECT id, language, content FROM pages WHERE language = ? AND content LIKE ?`,
			language, "%"+q+"%",
		)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var it SearchResult
				if err := rows.Scan(&it.ID, &it.Language, &it.Content); err == nil {
					results = append(results, it)
				}
			}
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{"search_results": results})
}

func APILoginHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad form"})
		return
	}
	username := r.FormValue("username")
	password := r.FormValue("password")

	u := User{}
	err := db.QueryRow(
		`SELECT id, username, email, password FROM users WHERE username = ?`,
		username,
	).Scan(&u.ID, &u.Username, &u.Email, &u.Password)
	if err != nil {
		renderTemplate(w, "login", map[string]any{"Title": "Sign In", "error": "Invalid username"})
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password)) != nil {
		renderTemplate(w, "login", map[string]any{"Title": "Sign In", "error": "Invalid password", "username": username})
		return
	}

	sess, _ := sessionStore.Get(r, "session")
	sess.Values["user_id"] = u.ID
	_ = sess.Save(r, w)

	http.Redirect(w, r, "/", http.StatusFound)
}

func APIRegisterHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad form"})
		return
	}
	username := r.FormValue("username")
	email := r.FormValue("email")
	pw1 := r.FormValue("password")
	pw2 := r.FormValue("password2")

	if username == "" || email == "" || pw1 == "" {
		renderTemplate(w, "register", map[string]any{"Title": "Sign Up", "error": "All fields required"})
		return
	}
	if pw1 != pw2 {
		renderTemplate(w, "register", map[string]any{"Title": "Sign Up", "error": "Password do not match"})
		return
	}

	var exists int
	_ = db.QueryRow(`SELECT COUNT(*) FROM users WHERE username = ?`, username).Scan(&exists)
	if exists > 0 {
		renderTemplate(w, "register", map[string]any{"Title": "Sign Up", "error": "Registration failed, Username already in use"})
		return
	}

	hash, _ := bcrypt.GenerateFromPassword([]byte(pw1), bcrypt.DefaultCost)
	_, err := db.Exec(
		`INSERT INTO users (username, email, password) VALUES (?, ?, ?)`,
		username, email, string(hash),
	)
	if err != nil {
		renderTemplate(w, "register", map[string]any{"Title": "Sign Up", "error": "Registration failed"})
		return
	}

	http.Redirect(w, r, "/login", http.StatusFound)
}

func APILogoutHandler(w http.ResponseWriter, r *http.Request) {
	sess, _ := sessionStore.Get(r, "session")
	delete(sess.Values, "user_id")
	_ = sess.Save(r, w)
	http.Redirect(w, r, "/", http.StatusFound)
}

// =====================
// Helpers
// =====================

func renderTemplate(w http.ResponseWriter, page string, data map[string]any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if data == nil {
		data = map[string]any{}
	}
	if _, ok := data["Title"]; !ok {
		data["Title"] = ""
	}
	// Execute the page wrapper ("search", "about", "login", "register")
	if err := tmpl.ExecuteTemplate(w, page, data); err != nil {
		http.Error(w, "template exec error: "+err.Error(), http.StatusInternalServerError)
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		http.Error(w, "json encode error: "+err.Error(), http.StatusInternalServerError)
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}