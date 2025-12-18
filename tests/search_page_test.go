package tests

import (
	"net/http"
	"net/http/httptest"
	"testing"

	h "devops-valgfag/handlers"
)

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
