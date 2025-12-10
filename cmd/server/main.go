// @title WhoKnows API
// @version 0.1.0
// @description API for the WhoKnows web app: session auth, search content, weather forecast, and health/readiness probes.
// @BasePath /
package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	_ "devops-valgfag/docs"
	h "devops-valgfag/handlers"
	metrics "devops-valgfag/internal/metrics"
	migrate "devops-valgfag/internal/migrate"

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

type dsnMeta struct {
	Source string
	Host   string
	DB     string
	User   string
}

func main() {
	// Set port
	port := getenv("PORT", "8080")

	// Resolve the database DSN (preferring Docker Compose settings)
	dsn, meta := resolvePostgresDSN()

	log.Printf("Using PostgreSQL DSN (source=%s host=%s db=%s user=%s)", meta.Source, meta.Host, meta.DB, meta.User)

	// Session key + FTS flag
	sessionKey := getenv("SESSION_KEY", "")
	if sessionKey == "" {
		log.Fatal("SESSION_KEY is required")
	}

	appEnv := getenv("APP_ENV", "dev")
	if appEnv == "prod" && len(sessionKey) < 32 {
		log.Fatal("SESSION_KEY must be at least 32 bytes in prod")
	}

	useFTS := getenv("SEARCH_FTS", "0")
	externalSearchEnabled := getenv("EXTERNAL_SEARCH", "1") == "1"

	// Open PostgreSQL
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Test DB connection
	if err := db.Ping(); err != nil {
		log.Fatal("Failed to connect to PostgreSQL:", err)
	}

	// Run PostgreSQL migrations here
	log.Println("Running database migrations...")
	if err := migrate.RunMigrations(db); err != nil {
		log.Fatalf("migration error: %v", err)
	}

	log.Println("Connected to PostgreSQL and migrations applied successfully!")

	// Templates
	funcs := template.FuncMap{
		"now":  time.Now,
		"year": func() int { return time.Now().Year() },
	}
	tmpl := template.Must(template.New("").Funcs(funcs).ParseGlob("./templates/*.html"))

	// Session cookies
	sessionStore := sessions.NewCookieStore([]byte(sessionKey))

	// Wire handlers
	h.Init(db, tmpl, sessionStore)

	// Toggle FTS
	if useFTS == "1" {
		h.EnableFTSSearch(true)
	} else {
		h.EnableFTSSearch(false)
	}
	h.EnableExternalSearch(externalSearchEnabled)

	// Router
	r := mux.NewRouter()
	// Metrics middleware
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
	r.HandleFunc("/api/search", h.APISearchHandler).Methods("GET", "POST")
	r.HandleFunc("/api/weather", h.APIWeatherHandler).Methods("GET")

	// Health check
	r.HandleFunc("/healthz", h.Healthz).Methods(http.MethodGet)

	// Readiness check
	r.HandleFunc("/readyz", h.Readyz).Methods(http.MethodGet)

	// Metrics endpoint
	r.Handle("/metrics", promhttp.Handler())

	// Swagger
	r.PathPrefix("/swagger/").Handler(httpSwagger.WrapHandler)

	// Start server
	fmt.Printf("Server running on :%s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}

// resolvePostgresDSN builds the connection string with sensible defaults.
// If DB_HOST is set (e.g., in Docker Compose), it takes precedence and we assemble
// the DSN from individual PostgreSQL env vars. Otherwise we honor DATABASE_URL,
// and finally fall back to a docker-friendly default host.
func resolvePostgresDSN() (string, dsnMeta) {
	if host := os.Getenv("DB_HOST"); host != "" {
		return buildPostgresDSN(host, "DB_HOST")
	}

	if dsn := os.Getenv("DATABASE_URL"); dsn != "" {
		meta, err := extractDSNMeta(dsn)
		if err != nil {
			log.Fatal("invalid DATABASE_URL")
		}
		meta.Source = "DATABASE_URL"
		return dsn, meta
	}

	return buildPostgresDSN("postgres_db", "default")
}

func buildPostgresDSN(host, source string) (string, dsnMeta) {
	port := getenv("POSTGRES_PORT", "5432")
	user := getenv("POSTGRES_USER", "devops")
	pass := getenv("POSTGRES_PASSWORD", "devops")
	dbName := getenv("POSTGRES_DB", "whoknows")

	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", user, pass, host, port, dbName)
	return dsn, dsnMeta{
		Source: source,
		Host:   host,
		DB:     dbName,
		User:   user,
	}
}

func extractDSNMeta(raw string) (dsnMeta, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return dsnMeta{}, err
	}

	dbName := strings.TrimPrefix(parsed.Path, "/")
	user := ""
	if parsed.User != nil {
		user = parsed.User.Username()
	}

	return dsnMeta{
		Host: parsed.Hostname(),
		DB:   dbName,
		User: user,
	}, nil
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
