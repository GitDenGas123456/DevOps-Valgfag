package handlers

import (
	"context"
	"net/http"
	"time"
)

// Healthz godoc
// @Summary      Liveness probe
// @Description  Returns ok when the service is running.
// @Tags         Health
// @Produce      plain
// @Success      200  {string}  string  "ok"
// @Failure      500  {string}  string  "internal error"
// @Router       /healthz [get]
func Healthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	// If the client disconnects while writing, Write may error.
	if _, err := w.Write([]byte("ok")); err != nil {
		_ = err
	}
}

// Readyz godoc
// @Summary      Readiness probe
// @Description  Checks database connectivity.
// @Tags         Health
// @Produce      plain
// @Success      200  {string}  string  "ready"
// @Failure      503  {string}  string  "database not ready"
// @Router       /readyz [get]
func Readyz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")

	if db == nil {
		http.Error(w, "database not configured", http.StatusServiceUnavailable)
		return
	}

	// Iindependent short timeout context so readiness isn't tied to client context,
	// and so it won't hang indefinitely if DB stalls.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		http.Error(w, "database not ready", http.StatusServiceUnavailable)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ready"))
}
