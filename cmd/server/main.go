package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	httpSwagger "github.com/swaggo/http-swagger"
	_ "modernc.org/sqlite"

	_ "devops-valgfag/docs"
	h "devops-valgfag/handlers"
	dbseed "devops-valgfag/internal/db"
)

func main() {
	port := getenv("PORT", "8080")
	dbPath := getenv("DATABASE_PATH", "data/seed/whoknows.db")

	// ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		log.Fatal(err)
	}

	// open sqlite database
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// ALWAYS seed the database on first run
	if err := dbseed.Seed(db); err != nil {
		log.Fatal(err)
	}
	log.Println("Database seeded successfully")

	// templates
	funcs := template.FuncMap{
		"now":  time.Now,
		"year": func() int { return time.Now().Year() },
	}
	tmpl := template.Must(template.New("").Funcs(funcs).ParseGlob("./templates/*.html"))

	// session
	sessionKey := getenv("SESSION_KEY", "development key")
	sessionStore := sessions.NewCookieStore([]byte(sessionKey))

	h.Init(db, tmpl, sessionStore)

	// router
	r := mux.NewRouter()

	fs := http.FileServer(http.Dir("static"))
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", fs))

	r.HandleFunc("/", h.SearchPageHandler).Methods("GET")
	r.HandleFunc("/about", h.AboutPageHandler).Methods("GET")
	r.HandleFunc("/login", h.LoginPageHandler).Methods("GET")
	r.HandleFunc("/register", h.RegisterPageHandler).Methods("GET")

	r.HandleFunc("/api/login", h.APILoginHandler).Methods("POST")
	r.HandleFunc("/api/register", h.APIRegisterHandler).Methods("POST")
	r.HandleFunc("/api/logout", h.APILogoutHandler).Methods("POST", "GET")
	r.HandleFunc("/api/search", h.APISearchHandler).Methods("POST")

	// health check
	r.HandleFunc("/healthz", h.Healthz).Methods(http.MethodGet)

	// swagger
	r.PathPrefix("/swagger/").Handler(httpSwagger.WrapHandler)

	fmt.Printf("Server running on :%s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
