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
	"strconv"
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
	// HTTP port
	port := getenv("PORT", "8080")

	// Env
	appEnv := getenv("APP_ENV", "dev")

	// Resolve DSN with precedence:
	// 1) DB_HOST + POSTGRES_* (Docker/VM)
	// 2) DATABASE_URL
	// 3) default postgres_db
	dsn, meta := resolvePostgresDSN()

	// Log DSN metadata safely
	if appEnv != "prod" {
		log.Printf("Using PostgreSQL DSN (source=%s host=%s db=%s user=%s)", meta.Source, meta.Host, meta.DB, meta.User)
	} else {
		log.Printf("Using PostgreSQL DSN (source=%s host=%s db=%s)", meta.Source, meta.Host, meta.DB)
	}

	// Session key + feature toggles
	sessionKey := getenv("SESSION_KEY", "")
	if sessionKey == "" {
		log.Fatal("SESSION_KEY is required")
	}
	// Note: len(sessionKey) counts bytes, not characters.
	if appEnv == "prod" && len(sessionKey) < 32 {
		log.Fatal("SESSION_KEY must be at least 32 bytes (not characters) in prod")
	}

	useFTS := getenv("SEARCH_FTS", "0") == "1"
	externalSearchEnabled := getenv("EXTERNAL_SEARCH", "1") == "1"

	// Open PostgreSQL using the pgx driver
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if cerr := db.Close(); cerr != nil {
			log.Printf("error closing DB: %v", cerr)
		}
	}()

	// Optional connection pool tuning (safe defaults)
	db.SetConnMaxLifetime(parseDurationEnv("DB_CONN_MAX_LIFETIME", 30*time.Minute))
	db.SetMaxOpenConns(parseIntEnv("DB_MAX_OPEN_CONNS", 10))
	db.SetMaxIdleConns(parseIntEnv("DB_MAX_IDLE_CONNS", 10))

	// Test DB connection
	if err := db.Ping(); err != nil {
		log.Fatal("Failed to connect to PostgreSQL:", err)
	}

	// Run database migrations
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
	h.EnableFTSSearch(useFTS)
	h.EnableExternalSearch(externalSearchEnabled)

	// Router
	r := mux.NewRouter()

	// Metrics middleware
	r.Use(metrics.RequestMetricsMiddleware())

	// Static files
	fs := http.FileServer(http.Dir("static"))
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", fs))

	// Page endpoints
	r.HandleFunc("/", h.HomePageHandler).Methods(http.MethodGet)
	r.HandleFunc("/about", h.AboutPageHandler).Methods(http.MethodGet)
	r.HandleFunc("/login", h.LoginPageHandler).Methods(http.MethodGet)
	r.HandleFunc("/register", h.RegisterPageHandler).Methods(http.MethodGet)
	r.HandleFunc("/weather", h.WeatherPageHandler).Methods(http.MethodGet)
	r.HandleFunc("/search", h.SearchPageHandler).Methods(http.MethodGet)

	// API endpoints
	r.HandleFunc("/api/login", h.APILoginHandler).Methods(http.MethodPost)
	r.HandleFunc("/api/register", h.APIRegisterHandler).Methods(http.MethodPost)
	r.HandleFunc("/api/logout", h.APILogoutHandler).Methods(http.MethodPost, http.MethodGet)
	// Search is GET-only in swagger + handler
	r.HandleFunc("/api/search", h.APISearchHandler).Methods(http.MethodGet)
	// Weather JSON API
	r.HandleFunc("/api/weather", h.APIWeatherHandler).Methods(http.MethodGet)

	// Health / readiness
	r.HandleFunc("/healthz", h.Healthz).Methods(http.MethodGet)
	r.HandleFunc("/readyz", h.Readyz).Methods(http.MethodGet)

	// Metrics endpoint
	r.Handle("/metrics", promhttp.Handler())

	// Swagger
	r.PathPrefix("/swagger/").Handler(httpSwagger.WrapHandler)

	// Start server (with timeouts)
	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	fmt.Printf("Server running on :%s\n", port)
	log.Fatal(srv.ListenAndServe())
}

func resolvePostgresDSN() (string, dsnMeta) {
	if host := os.Getenv("DB_HOST"); host != "" {
		return buildPostgresDSN(host, "DB_HOST")
	}

	if dsn := os.Getenv("DATABASE_URL"); dsn != "" {
		meta, err := extractDSNMeta(dsn)
		if err != nil {
			log.Fatal("invalid DATABASE_URL:", err)
		}
		meta.Source = "DATABASE_URL"
		return dsn, meta
	}

	// default for docker-compose
	return buildPostgresDSN("postgres_db", "default")
}

func buildPostgresDSN(host, source string) (string, dsnMeta) {
	port := getenv("POSTGRES_PORT", "5432")
	user := getenv("POSTGRES_USER", "devops")
	pass := getenv("POSTGRES_PASSWORD", "devops")
	dbName := getenv("POSTGRES_DB", "whoknows")
	sslmode := getenv("POSTGRES_SSLMODE", "disable") // keeps your current behavior by default

	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		url.QueryEscape(user),
		url.QueryEscape(pass),
		host, // do NOT escape host
		port,
		url.QueryEscape(dbName),
		url.QueryEscape(sslmode),
	)

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

func parseIntEnv(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 0 {
		return fallback
	}
	return n
}

func parseDurationEnv(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil || d <= 0 {
		return fallback
	}
	return d
}
