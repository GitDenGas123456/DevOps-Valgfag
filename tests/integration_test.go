package tests

import (
	"database/sql"
	"html/template"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	h "devops-valgfag/handlers"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	_ "modernc.org/sqlite"
)

// setupTestServer builds a lightweight "production-like" HTTP server for integration tests.
//
// What we include (and why):
// - In-memory SQLite DB: fast + isolated per test (no shared state, no external services)
// - Real templates: catches missing template funcs/fields and verifies rendered HTML
// - Cookie-based session store: tests auth flow + cookies realistically
// - Gorilla mux router: validates route wiring + HTTP methods (GET/POST) like production
func setupTestServer(t *testing.T) (*mux.Router, *sql.DB) {
	t.Helper()

	// In-memory SQLite (lives only for the duration of the test process)
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}

	// Create the schema the handlers expect (users/pages/etc.)
	// If schema init fails, close DB immediately to avoid leaking resources.
	if err := h.InitSchema(db); err != nil {
		_ = db.Close()
		t.Fatal(err)
	}

	// Template funcs used by templates (must match production, otherwise rendering may fail)
	funcs := template.FuncMap{
		"now":  time.Now,
		"year": func() int { return time.Now().Year() },
	}

	// Parse templates from disk so we test actual HTML output and template wiring.
	tmpl := template.Must(template.New("").Funcs(funcs).ParseGlob("../templates/*.html"))

	// Cookie sessions: used by login/register to set auth cookie.
	sessionStore := sessions.NewCookieStore([]byte("test-key"))

	// Initialize handlers with test dependencies.
	h.Init(db, tmpl, sessionStore)

	// Keep tests deterministic: avoid calling external services (Wikipedia enrichment etc.).
	h.EnableExternalSearch(false)

	// Router mirrors the routes we support in the application.
	r := mux.NewRouter()

	// Pages (HTML)
	r.HandleFunc("/", h.HomePageHandler).Methods(http.MethodGet)
	r.HandleFunc("/search", h.SearchPageHandler).Methods(http.MethodGet)
	r.HandleFunc("/about", h.AboutPageHandler).Methods(http.MethodGet)
	r.HandleFunc("/login", h.LoginPageHandler).Methods(http.MethodGet)
	r.HandleFunc("/register", h.RegisterPageHandler).Methods(http.MethodGet)

	// API (auth + search)
	r.HandleFunc("/api/login", h.APILoginHandler).Methods(http.MethodPost)
	r.HandleFunc("/api/register", h.APIRegisterHandler).Methods(http.MethodPost)
	r.HandleFunc("/api/logout", h.APILogoutHandler).Methods(http.MethodPost)
	r.HandleFunc("/api/search", h.APISearchHandler).Methods(http.MethodGet)

	// Ops endpoints
	r.HandleFunc("/healthz", h.Healthz).Methods(http.MethodGet)
	r.HandleFunc("/readyz", h.Readyz).Methods(http.MethodGet)

	return r, db
}

func TestIntegration_RegisterLoginSearch(t *testing.T) {
	router, db := setupTestServer(t)
	defer func() {
		if err := db.Close(); err != nil {
			t.Errorf("failed to close db: %v", err)
		}
	}()

	// 1) Register a user via form POST to /api/register
	form := url.Values{}
	form.Set("username", "alice")
	form.Set("email", "alice@example.com")
	form.Set("password", "secret")
	form.Set("password2", "secret")

	req := httptest.NewRequest(http.MethodPost, "/api/register", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusFound {
		t.Fatalf("expected redirect after register, got %d", rr.Code)
	}

	// 2) Login the user, expecting a redirect + session cookie in the response.
	form = url.Values{}
	form.Set("username", "alice")
	form.Set("password", "secret")

	req = httptest.NewRequest(http.MethodPost, "/api/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusFound {
		t.Fatalf("expected redirect after login, got %d", rr.Code)
	}

	// Extract cookies from the login response (this is the auth session).
	cookies := rr.Result().Cookies()

	// 3) Call authenticated endpoint /api/search by attaching the session cookie.
	req = httptest.NewRequest(http.MethodGet, "/api/search?q=test", nil)
	for _, c := range cookies {
		req.AddCookie(c)
	}

	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 on search, got %d", rr.Code)
	}

	// Minimal smoke assertion: response should contain the search results marker.
	if !strings.Contains(rr.Body.String(), "search_results") {
		t.Errorf("expected search_results in response, got %s", rr.Body.String())
	}
}

func TestIntegration_Healthz(t *testing.T) {
	router, db := setupTestServer(t)
	defer func() {
		if err := db.Close(); err != nil {
			t.Errorf("failed to close db: %v", err)
		}
	}()

	// /healthz is liveness: should return 200 + "ok" if process is running.
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for /healthz, got %d", rr.Code)
	}
	if rr.Body.String() != "ok" {
		t.Fatalf("expected body 'ok', got '%s'", rr.Body.String())
	}
}

func TestIntegration_Readyz(t *testing.T) {
	router, db := setupTestServer(t)
	defer func() {
		if err := db.Close(); err != nil {
			t.Errorf("failed to close db: %v", err)
		}
	}()

	// /readyz is readiness: should return 200 + "ready" when DB is reachable.
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for /readyz, got %d", rr.Code)
	}
	if strings.TrimSpace(rr.Body.String()) != "ready" {
		t.Fatalf("expected body 'ready', got '%s'", rr.Body.String())
	}
}
