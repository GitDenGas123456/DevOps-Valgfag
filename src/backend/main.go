package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"

	"github.com/GitDenGas123456/DevOps-Valgfag/src/database"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"golang.org/x/crypto/bcrypt"
)

var (
	tmpl         *template.Template
	sessionStore = sessions.NewCookieStore([]byte("development key"))
)

// --- Models ---
type User struct {
	ID       int
	Username string
	Email    string
	Password string
}

type Page struct {
	ID       int
	Language string
	Content  string
}

// --- Main ---
func main() {
	// Initialize DB from the database package
	database.InitDB("../database/whoknows.db")
	defer database.DB.Close()

	// Parse templates including layout
	tmpl = template.Must(template.ParseGlob("templates/*.html"))

	r := mux.NewRouter()

	// Static files
	fs := http.FileServer(http.Dir("./static"))
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", fs))

	// Page routes
	r.HandleFunc("/", searchHandler).Methods("GET")
	r.HandleFunc("/about", aboutHandler).Methods("GET")
	r.HandleFunc("/login", loginHandler).Methods("GET")
	r.HandleFunc("/register", registerHandler).Methods("GET")

	// API routes
	r.HandleFunc("/api/login", apiLoginHandler).Methods("POST")
	r.HandleFunc("/api/register", apiRegisterHandler).Methods("POST")
	r.HandleFunc("/api/logout", apiLogoutHandler).Methods("POST")
	r.HandleFunc("/api/search", apiSearchHandler).Methods("POST")

	fmt.Println("Server running on :8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}

// --- Template Helper ---
func renderTemplate(w http.ResponseWriter, r *http.Request, tmplName string, data map[string]interface{}) {
	if data == nil {
		data = map[string]interface{}{}
	}

	// Add logged-in user if present
	sess, _ := sessionStore.Get(r, "session")
	if userID, ok := sess.Values["user_id"].(int); ok {
		user := User{}
		err := database.DB.QueryRow("SELECT id, username, email FROM users WHERE id=?", userID).
			Scan(&user.ID, &user.Username, &user.Email)
		if err == nil {
			data["User"] = user
		}
	}

	// Execute layout template with child blocks
	err := tmpl.ExecuteTemplate(w, "layout.html", data)
	if err != nil {
		log.Printf("Template %s error: %v", tmplName, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// --- Page Handlers ---
func loginHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, r, "layout.html", map[string]interface{}{
		"Page": "login",
	})
}

func registerHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, r, "layout.html", map[string]interface{}{
		"Page": "register",
	})
}

func aboutHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, r, "layout.html", map[string]interface{}{
		"Page": "about",
	})
}

func searchHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	results := []Page{}
	if q != "" {
		rows, _ := database.DB.Query("SELECT id, language, content FROM pages WHERE content LIKE ?", "%"+q+"%")
		defer rows.Close()
		for rows.Next() {
			var p Page
			rows.Scan(&p.ID, &p.Language, &p.Content)
			results = append(results, p)
		}
	}

	renderTemplate(w, r, "layout.html", map[string]interface{}{
		"Page":          "search",
		"Query":         q,
		"SearchResults": results,
	})
}

// --- API Handlers ---
func apiSearchHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	q := r.FormValue("q")
	language := r.FormValue("language")
	if language == "" {
		language = "en"
	}

	var results []Page
	if q != "" {
		rows, err := database.DB.Query("SELECT id, language, content FROM pages WHERE language=? AND content LIKE ?", language, "%"+q+"%")
		if err != nil {
			log.Println("DB query error:", err)
		} else {
			defer rows.Close()
			for rows.Next() {
				var p Page
				rows.Scan(&p.ID, &p.Language, &p.Content)
				results = append(results, p)
			}
		}
	}

	renderTemplate(w, r, "search.html", map[string]interface{}{
		"SearchResults": results,
		"Query":         q,
	})
}

func apiLoginHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	username := r.FormValue("username")
	password := r.FormValue("password")

	user := User{}
	err := database.DB.QueryRow("SELECT id, username, email, password FROM users WHERE username=?", username).
		Scan(&user.ID, &user.Username, &user.Email, &user.Password)
	if err != nil {
		renderTemplate(w, r, "login.html", map[string]interface{}{"Error": "Invalid username"})
		return
	}
	if !checkPassword(user.Password, password) {
		renderTemplate(w, r, "login.html", map[string]interface{}{"Error": "Invalid password"})
		return
	}

	sess, _ := sessionStore.Get(r, "session")
	sess.Values["user_id"] = user.ID
	sess.Save(r, w)

	http.Redirect(w, r, "/", http.StatusFound)
}

func apiRegisterHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	username := r.FormValue("username")
	email := r.FormValue("email")
	password := r.FormValue("password")
	password2 := r.FormValue("password2")

	// Validation
	if username == "" || email == "" || password == "" || password2 == "" {
		renderTemplate(w, r, "layout.html", map[string]interface{}{
			"Page":  "register",
			"Error": "All fields are required",
		})
		return
	}

	if password != password2 {
		renderTemplate(w, r, "layout.html", map[string]interface{}{
			"Page":  "register",
			"Error": "Passwords do not match",
		})
		return
	}

	// Check if username exists
	var exists int
	err := database.DB.QueryRow("SELECT COUNT(*) FROM users WHERE username=?", username).Scan(&exists)
	if err != nil {
		log.Println("DB error:", err)
		renderTemplate(w, r, "layout.html", map[string]interface{}{
			"Page":  "register",
			"Error": "Internal error",
		})
		return
	}
	if exists > 0 {
		renderTemplate(w, r, "layout.html", map[string]interface{}{
			"Page":  "register",
			"Error": "Username already in use",
		})
		return
	}

	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Println("Password hash error:", err)
		renderTemplate(w, r, "layout.html", map[string]interface{}{
			"Page":  "register",
			"Error": "Internal error",
		})
		return
	}

	// Insert user
	_, err = database.DB.Exec("INSERT INTO users (username, email, password) VALUES (?, ?, ?)", username, email, string(hash))
	if err != nil {
		log.Println("DB insert error:", err)
		renderTemplate(w, r, "layout.html", map[string]interface{}{
			"Page":  "register",
			"Error": "Registration failed",
		})
		return
	}

	http.Redirect(w, r, "/login", http.StatusFound)
}

func apiLogoutHandler(w http.ResponseWriter, r *http.Request) {
	sess, _ := sessionStore.Get(r, "session")
	delete(sess.Values, "user_id")
	sess.Save(r, w)
	http.Redirect(w, r, "/", http.StatusFound)
}

// --- Security ---
func checkPassword(hash string, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
