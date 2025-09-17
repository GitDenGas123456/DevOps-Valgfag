package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"

	_ "github.com/mattn/go-sqlite3"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"golang.org/x/crypto/bcrypt"

)

var (
	db *sql.DB
	tmpl *template.Template
	sessionStore = sessions.NewCookieStore([]byte("development key"))

)


//Structure used to make and asses login
type User struct {
	ID         int
	Username   string
	Email      string
	Password   string

}

// INSERT
type Page struct {
	ID         int
	Language   string
	Content    string

}

func main() {
	var err error
	db, err = sql.Open("sqlite3", "../whoknows.db")
	if err !=nil {
		log.Fatal(err)
	}
	defer db.Close()

	tmpl = template.Must(template.ParseGlob("templates/*.html"))

	r := mux.NewRouter()

	//Routes for pages
	r.HandleFunc("/", searchHandler).Methods("GET")
	r.HandleFunc("/about", aboutHandler).Methods("GET")
	r.HandleFunc("/login", loginHandler).Methods("GET")
	r.HandleFunc("/register", registerHandler).Methods("GET")
	r.HandleFunc("/api/login", apiLoginHandler).Methods("POST")
	r.HandleFunc("/api/register", apiRegisterHandler).Methods("POST")
	r.HandleFunc("/api/logout", apiLogoutHandler).Methods("POST")
	r.HandleFunc("/api/search", apiSearchHandler).Methods("POST")

	fmt.Println("Server running on :8080")
	http.ListenAndServe(":8080", r)

}

//Handlers 
func searchHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	language := r.URL.Query().Get("language")
	if language == "" {
		language = "en"
	}
	var results []Page
	if q != "" {
		rows, err := db.Query("SELECT id, language, content FROM pages WHERE language = ? AND conten LIKE ?", language, "%"+q+"%")
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var p Page
				rows.Scan(&p.ID, &p.Language, &p.Content)
				results = append(results, p)
			}
		}
	}

	tmpl.ExecuteTemplate(w, "search.hmtl", map[string]interface{}{
		"search_results": results,
		"query":          q,
	})
}

func aboutHandler (w http.ResponseWriter, r *http.Request) {
	tmpl.ExecuteTemplate(w, "about.html", nil)
}

func loginHandler (w http.ResponseWriter, r *http.Request) {
	tmpl.ExecuteTemplate(w, "login.html", nil)
}

func registerHandler (w http.ResponseWriter, r *http.Request) {
	tmpl.ExecuteTemplate(w, "register.html", nil)
}

//API handlers
func apiSearchHandler (w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	language := r.URL.Query().Get("language")
	if language == "" {
		language = "en"
	}
	var results []Page
	if q != "" {
		rows, err := db.Query("SELECT id, language, content FROM page WHERE language = ? AND content LIKE ?", language, "%"+q+"%")
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var p Page
				rows.Scan(&p.ID, &p.Language, &p.Content)
				results = append(results, p)
			}
		}
	}
	json.NewEncoder(w).Encode(map[string]interface{}{"search_resilts": results})
}


func apiLoginHandler (w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	username := r.FormValue("username")
	password := r.FormValue("password")

	user := User{}
	err := db.QueryRow("SELECT id, username, email, password FROM user WHERE username = ?", username).Scan(&user.ID, &user.Username, &user.Email, &user.Password)
	if err != nil {
		tmpl.ExecuteTemplate(w, "login.html", map[string]string{"error": "Invalid username"})
		return
	}
	if !checkPassword(user.Password, password) {
		tmpl.ExecuteTemplate(w, "login.html", map[string]string{"error": "Invalid password"})
		return
	}

	sess, _ := sessionStore.Get(r, "session")
	sess.Values["user_id"] = user.ID
	sess.Save(r, w)

	http.Redirect(w, r, "/", http.StatusFound)
}

func apiRegisterHandler (w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	username := r.FormValue("username")
	email := r.FormValue("email")
	password := r.FormValue("password")
	password2 := r.FormValue("password2")

	if username == "" || email == "" || password == "" {
		tmpl.ExecuteTemplate(w, "register.html", map[string]string{"error": "All fields required"})
		return
	}

	if password != password2 {
		tmpl.ExecuteTemplate(w, "register.html", map[string]string{"error": "Password do not match"})
		return
	}

	var exists int
	db.QueryRow("SELECT COUNT(*) FROM users WHERE username =?", username).Scan(&exists)
	if exists > 0 {
		tmpl.ExecuteTemplate(w, "register.html", map[string]string{"error": "Registration failed, Username already in use"})
		return
	}

	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	_, err := db.Exec("INSERT INTO users (username, email, password) VALUES (?, ?, ?)", username, email, string(hash))
	if err != nil {
		tmpl.ExecuteTemplate(w, "register.html", map[string]string{"error": "Registration failed"})
		return
	}

	http.Redirect(w, r, "/", http.StatusFound)
}

func apiLogoutHandler (w http.ResponseWriter, r *http.Request) {
	sess, _ := sessionStore.Get(r, "session")
	delete(sess.Values, "user_id")
	sess.Save(r, w)
	http.Redirect(w, r, "/", http.StatusFound)
}

//Security helpers
func checkPassword(hash string, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
