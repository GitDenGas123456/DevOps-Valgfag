// @title WhoKnows API
// @version 0.1.0
// @description API specification for the WhoKnows web application
// @BasePath /
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

	_ "devops-valgfag/docs"
	h "devops-valgfag/handlers"
	metrics "devops-valgfag/internal/metrics"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	httpSwagger "github.com/swaggo/http-swagger"

	// PostgreSQL driver
	_ "github.com/jackc/pgx/v5/stdlib"
)

// User represents an application user
// @Description Application user with login credentials
type User struct {
	ID       int    `json:"id" example:"1"`
	Username string `json:"username" example:"alice"`
	Email    string `json:"email" example:"alice@example.com"`
	Password string `json:"password,omitempty"` // bcrypt hash
}

type SearchResult struct {
	ID       int    `json:"id" example:"1"`
	Language string `json:"language" example:"en"`
	Content  string `json:"content" example:"Sample content"`
}

// SearchResponse represents an API search response
type SearchResponse struct {
	SearchResults []SearchResult `json:"search_results"`
}

// AuthResponse represents a generic auth API response
type AuthResponse struct {
	StatusCode int    `json:"statusCode" example:"200"`
	Message    string `json:"message" example:"Login successful"`
}

func main() {
	// Set port
	port := getenv("PORT", "8080")

	// Required PostgreSQL DSN
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL environment variable is required when using PostgreSQL")
	}

	// This keeps your original directory creation logic (harmless, still needed for templates)
	dbPath := getenv("DATABASE_PATH", "data/seed/whoknows.db")
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		log.Fatal(err)
	}

	// Session key + FTS flag
	sessionKey := getenv("SESSION_KEY", "development key")
	useFTS := getenv("SEARCH_FTS", "")

	// Open PostgreSQL instead of SQLite
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Test DB connection
	if err := db.Ping(); err != nil {
		log.Fatal("Failed to connect to PostgreSQL:", err)
	}

	fmt.Println("Connected to PostgreSQL successfully!")
	// Templates
	funcs := template.FuncMap{
		"now":  time.Now,
		"year": func() int { return time.Now().Year() },
	}
	tmpl := template.Must(template.New("").Funcs(funcs).ParseGlob("./templates/*.html"))

	// Session cookies
	sessionStore := sessions.NewCookieStore([]byte(sessionKey))

	h.Init(db, tmpl, sessionStore)

	// Toggle FTS
	if useFTS == "1" {
		h.EnableFTSSearch(true)
	} else {
		h.EnableFTSSearch(false)
	}

	// Router
	r := mux.NewRouter()
	r.Use(metrics.RequestMetricsMiddleware())

	// Static files
	fs := http.FileServer(http.Dir("static"))
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", fs))

	// Page endpoints
	r.HandleFunc("/", h.SearchPageHandler).Methods("GET")
	r.HandleFunc("/about", h.AboutPageHandler).Methods("GET")
	r.HandleFunc("/login", h.LoginPageHandler).Methods("GET")
	r.HandleFunc("/register", h.RegisterPageHandler).Methods("GET")
	r.HandleFunc("/weather", h.WeatherPageHandler).Methods("GET")

	// API endpoints
	r.HandleFunc("/api/login", h.APILoginHandler).Methods("POST")
	r.HandleFunc("/api/register", h.APIRegisterHandler).Methods("POST")
	r.HandleFunc("/api/logout", h.APILogoutHandler).Methods("POST", "GET")
	r.HandleFunc("/api/search", h.APISearchHandler).Methods("POST")

	// Health check
	r.HandleFunc("/healthz", h.Healthz).Methods(http.MethodGet)

	// Metrics endpoint
	r.Handle("/metrics", promhttp.Handler())

	// Swagger
	r.PathPrefix("/swagger/").Handler(httpSwagger.WrapHandler)

	// Start server
	fmt.Printf("Server running on :%s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
