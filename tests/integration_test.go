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

// setupTestServer initializes in-memory DB, templates, sessions, and router
func setupTestServer(t *testing.T) (*mux.Router, *sql.DB) {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}

	// Seed database schema
	if err := h.InitSchema(db); err != nil {
		_ = db.Close()
		t.Fatal(err)
	}

	// Template functions
	funcs := template.FuncMap{
		"now":  time.Now,
		"year": func() int { return time.Now().Year() },
	}

	// Load templates with funcs
	tmpl := template.Must(template.New("").Funcs(funcs).ParseGlob("../templates/*.html"))

	// Session store
	sessionStore := sessions.NewCookieStore([]byte("test-key"))

	// Init handlers
	h.Init(db, tmpl, sessionStore)

	// Router setup
	r := mux.NewRouter()
	r.HandleFunc("/", h.HomePageHandler).Methods(http.MethodGet)
	r.HandleFunc("/search", h.SearchPageHandler).Methods(http.MethodGet)
	r.HandleFunc("/about", h.AboutPageHandler).Methods(http.MethodGet)
	r.HandleFunc("/login", h.LoginPageHandler).Methods(http.MethodGet)
	r.HandleFunc("/register", h.RegisterPageHandler).Methods(http.MethodGet)

	r.HandleFunc("/api/login", h.APILoginHandler).Methods(http.MethodPost)
	r.HandleFunc("/api/register", h.APIRegisterHandler).Methods(http.MethodPost)
	r.HandleFunc("/api/logout", h.APILogoutHandler).Methods(http.MethodPost)
	r.HandleFunc("/api/search", h.APISearchHandler).Methods(http.MethodGet)

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

	// 1. Register a new user
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

	// 2. Login with the new user
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

	// Extract session cookie
	cookies := rr.Result().Cookies()

	// 3. Perform a search (GET => query params in URL, no body)
	req = httptest.NewRequest(http.MethodGet, "/api/search?q=test", nil)
	for _, c := range cookies {
		req.AddCookie(c)
	}

	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 on search, got %d", rr.Code)
	}

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
