package tests

import (
	"net/http"
	"net/http/httptest"
	"testing"

	h "devops-valgfag/handlers"
)

func TestSearchPageHandler_NoQuery_OK(t *testing.T) {
	db := setupTestHandlers(t)
	defer closeDB(t, db)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
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

	req := httptest.NewRequest(http.MethodGet, "/?q=hello&language=en", nil)
	rec := httptest.NewRecorder()

	h.SearchPageHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if rec.Body.String() == "" {
		t.Fatalf("expected non-empty body")
	}
}
