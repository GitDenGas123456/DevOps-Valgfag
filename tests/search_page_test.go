package tests

import (
	"net/http"
	"net/http/httptest"
	"testing"

	h "devops-valgfag/handlers"
)

// These tests validate page-level HTTP handlers in isolation.
// Handlers are invoked directly (no router, no real server) to verify
// correct status codes, redirects, and that a response body is rendered.
func TestHomePageHandler_NoQuery_OK(t *testing.T) {
	db := setupTestHandlers(t)
	defer closeDB(t, db)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	h.HomePageHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if rec.Body.String() == "" {
		t.Fatalf("expected non-empty body")
	}
}

// When the home page is requested with a search query,
// the handler should redirect to the /search page with the same parameters.
func TestHomePageHandler_WithQuery_Redirects(t *testing.T) {
	db := setupTestHandlers(t)
	defer closeDB(t, db)

	req := httptest.NewRequest(http.MethodGet, "/?q=hello&language=en", nil)
	rec := httptest.NewRecorder()

	h.HomePageHandler(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("expected status 302, got %d", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "/search?q=hello&language=en" {
		t.Fatalf("unexpected redirect Location: %s", loc)
	}
}

// The search page should render successfully even when no query is provided,
// allowing the user to see an empty or initial search state.
func TestSearchPageHandler_NoQuery_OK(t *testing.T) {
	db := setupTestHandlers(t)
	defer closeDB(t, db)

	req := httptest.NewRequest(http.MethodGet, "/search", nil)
	rec := httptest.NewRecorder()

	h.SearchPageHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if rec.Body.String() == "" {
		t.Fatalf("expected non-empty body")
	}
}

// When a query is provided, the search page handler should return HTTP 200
// and render a response containing search results.
func TestSearchPageHandler_WithQuery_OK(t *testing.T) {
	db := setupTestHandlers(t)
	defer closeDB(t, db)

	req := httptest.NewRequest(http.MethodGet, "/search?q=hello&language=en", nil)
	rec := httptest.NewRecorder()

	h.SearchPageHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if rec.Body.String() == "" {
		t.Fatalf("expected non-empty body")
	}
}
