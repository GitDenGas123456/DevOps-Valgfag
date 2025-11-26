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

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	h "devops-valgfag/handlers"
	_ "modernc.org/sqlite"
)

// setupTestServer initializes in-memory DB, templates, sessions, and router
func setupTestServer(t *testing.T) (*mux.Router, *sql.DB) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}

	// Seed database schema
	if err := h.SeedDB(db); err != nil {
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
	r.HandleFunc("/", h.SearchPageHandler).Methods("GET")
	r.HandleFunc("/about", h.AboutPageHandler).Methods("GET")
	r.HandleFunc("/login", h.LoginPageHandler).Methods("GET")
	r.HandleFunc("/register", h.RegisterPageHandler).Methods("GET")
	r.HandleFunc("/api/login", h.APILoginHandler).Methods("POST")
	r.HandleFunc("/api/register", h.APIRegisterHandler).Methods("POST")
	r.HandleFunc("/api/logout", h.APILogoutHandler).Methods("POST", "GET")
	r.HandleFunc("/api/search", h.APISearchHandler).Methods("POST")
	r.HandleFunc("/healthz", h.Healthz).Methods("GET")

	return r, db
}



func TestIntegration_RegisterLoginSearch(t *testing.T) {
	router, db := setupTestServer(t)
	defer db.Close()

	// 1. Register a new user
	form := url.Values{}
	form.Set("username", "alice")
	form.Set("email", "alice@example.com")
	form.Set("password", "secret")
	form.Set("password2", "secret")
	req := httptest.NewRequest("POST", "/api/register", strings.NewReader(form.Encode()))
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
	req = httptest.NewRequest("POST", "/api/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusFound {
		t.Fatalf("expected redirect after login, got %d", rr.Code)
	}

	// Extract session cookie
	cookies := rr.Result().Cookies()

	// 3. Perform a search
	form = url.Values{}
	form.Set("q", "test")
	req = httptest.NewRequest("POST", "/api/search?q=test", strings.NewReader(form.Encode()))
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
	router, _ := setupTestServer(t)

	req := httptest.NewRequest("GET", "/healthz", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for /healthz, got %d", rr.Code)
	}

	if rr.Body.String() != "ok" {
		t.Fatalf("expected body 'ok', got '%s'", rr.Body.String())
	}
}
