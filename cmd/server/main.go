// @title WhoKnows API
// @version 0.1.0
// @description API for the WhoKnows web app: session auth, search content, weather forecast, and health/readiness probes.
// @BasePath /

// @securityDefinitions.apikey sessionAuth
// @in header
// @name Cookie
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

// User represents an application user with login credentials.
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

// dsnMeta is safe-to-log info about the DB connection (no password).
type dsnMeta struct {
	Source string // DB_HOST / DATABASE_URL / default
	Host   string
	DB     string
	User   string
}

func main() {

	// -------------------------
	// Runtime config
	// -------------------------

	// PORT: which TCP port the HTTP server listens on (default 8080).
	port := getenv("PORT", "8080")

	// APP_ENV: used to toggle "prod" behavior (e.g. safer logging).
	appEnv := getenv("APP_ENV", "dev")

	// DSN = "Data Source Name" = connection string used by sql.Open().
	// meta = non-sensitive info we can safely log for debugging.
	dsn, meta := resolvePostgresDSN()

	// In prod we log LESS to avoid leaking details (even if it's "only" username).
	if appEnv != "prod" {
		log.Printf("Using PostgreSQL DSN (source=%s host=%s db=%s user=%s)", meta.Source, meta.Host, meta.DB, meta.User)
	} else {
		log.Printf("Using PostgreSQL DSN (source=%s host=%s db=%s)", meta.Source, meta.Host, meta.DB)
	}

	// SESSION_KEY is used by gorilla/sessions to sign (and possibly encrypt) cookies.
	// If it is weak, sessions can be forged. That's why we enforce 32+ bytes in prod.
	sessionKey := getenv("SESSION_KEY", "")
	if sessionKey == "" {
		log.Fatal("SESSION_KEY is required")
	}
	// Note: len(sessionKey) counts bytes, not characters.
	if appEnv == "prod" && len(sessionKey) < 32 {
		log.Fatal("SESSION_KEY must be at least 32 bytes (not characters) in prod")
	}

	// Feature toggles
	useFTS := getenv("SEARCH_FTS", "0") == "1"
	externalSearchEnabled := getenv("EXTERNAL_SEARCH", "1") == "1"

	// -------------------------
	// Database
	// -------------------------

	// Open PostgreSQL using the pgx driver
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		log.Fatal(err)
	}
	// Ensure DB is closed on main() exit
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

	// -------------------------
	// HTTP (templates, sessions, router)
	// -------------------------

	// Templates
	funcs := template.FuncMap{
		"now":  time.Now,
		"year": func() int { return time.Now().Year() },
	}
	tmpl := template.Must(template.New("").Funcs(funcs).ParseGlob("./templates/*.html"))

	// Session store backed by secure cookies.
	// The sessionKey is used to sign cookies so clients cannot tamper with them.
	sessionStore := sessions.NewCookieStore([]byte(sessionKey))

	// "Wire handlers" = give handlers access to shared dependencies:
	// - db connection
	// - parsed HTML templates
	// - session store
	h.Init(db, tmpl, sessionStore)
	h.EnableFTSSearch(useFTS)
	h.EnableExternalSearch(externalSearchEnabled)

	// Router
	r := mux.NewRouter()

	// Metrics middleware
	r.Use(metrics.RequestMetricsMiddleware())

	// Routes
	// - Static assets
	// - Pages
	// - API
	// - Health/metrics
	// - Swagger
	fs := http.FileServer(http.Dir("static"))
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", fs))

	r.HandleFunc("/", h.HomePageHandler).Methods(http.MethodGet, http.MethodHead)
	r.HandleFunc("/about", h.AboutPageHandler).Methods(http.MethodGet, http.MethodHead)
	r.HandleFunc("/login", h.LoginPageHandler).Methods(http.MethodGet, http.MethodHead)
	r.HandleFunc("/register", h.RegisterPageHandler).Methods(http.MethodGet, http.MethodHead)
	r.HandleFunc("/weather", h.WeatherPageHandler).Methods(http.MethodGet, http.MethodHead)
	r.HandleFunc("/search", h.SearchPageHandler).Methods(http.MethodGet, http.MethodHead)

	r.HandleFunc("/api/login", h.APILoginHandler).Methods(http.MethodPost)
	r.HandleFunc("/api/register", h.APIRegisterHandler).Methods(http.MethodPost)
	r.HandleFunc("/api/logout", h.APILogoutHandler).Methods(http.MethodPost)

	r.HandleFunc("/api/search", h.APISearchHandler).Methods(http.MethodGet)

	r.HandleFunc("/api/weather", h.APIWeatherHandler).Methods(http.MethodGet)

	r.HandleFunc("/healthz", h.Healthz).Methods(http.MethodGet, http.MethodHead)
	r.HandleFunc("/readyz", h.Readyz).Methods(http.MethodGet, http.MethodHead)

	r.Handle("/metrics", promhttp.Handler())

	swaggerHandler := httpSwagger.WrapHandler
	// Support both /swagger and /swagger/index.html (avoids 404 without trailing slash).
	r.HandleFunc("/swagger", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/swagger/index.html", http.StatusPermanentRedirect)
	}).Methods(http.MethodGet, http.MethodHead)
	r.PathPrefix("/swagger/").Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			r.Method = http.MethodGet
		}
		swaggerHandler.ServeHTTP(w, r)
	})).Methods(http.MethodGet, http.MethodHead)

	// -------------------------
	// Server
	// -------------------------

	// http.Server lets us configure timeouts (recommended in production).
	// Handler: r means "use the mux router to handle every incoming request".
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

// resolvePostgresDSN determines how the application should connect to PostgreSQL.
// Precedence:
// 1) DB_HOST + POSTGRES_* env vars (Docker / VM / local compose)
// 2) DATABASE_URL (CI / cloud / managed databases)
// 3) Fallback to docker-compose default service name
//
// It returns:
// - a full DSN string used to open the DB connection
// - safe-to-log metadata describing the chosen configuration
func resolvePostgresDSN() (string, dsnMeta) {

	// Case 1: Running in Docker / VM where DB host is provided explicitly
	if host := os.Getenv("DB_HOST"); host != "" {
		return buildPostgresDSN(host, "DB_HOST")
	}

	// Case 2: Running in CI/cloud where a full DATABASE_URL is injected (e.g. GitHub Actions env).
	if dsn := os.Getenv("DATABASE_URL"); dsn != "" {
		meta, err := extractDSNMeta(dsn)
		if err != nil {
			log.Fatal("invalid DATABASE_URL:", err)
		}
		meta.Source = "DATABASE_URL"
		return dsn, meta
	}

	// Case 3: Local docker-compose fallback (service name resolution)
	return buildPostgresDSN("postgres_db", "default")
}

// buildPostgresDSN constructs a PostgreSQL connection string from individual environment variables.
// This is used when the environment does NOT provide a full DATABASE_URL.
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
