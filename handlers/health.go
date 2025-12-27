package handlers

import (
	"context"
	"log"
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

	// Avoid writing a body for HEAD requests.
	if r.Method == http.MethodHead {
		w.WriteHeader(http.StatusOK)
		return
	}

	w.WriteHeader(http.StatusOK)

	// If the client disconnects while writing, Write may error.
	_, _ = w.Write([]byte("ok"))
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

	// Avoid writing a body for HEAD requests.
	if r.Method == http.MethodHead {
		if db == nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		if err := db.PingContext(ctx); err != nil {
			log.Printf("readyz: db ping failed: %v", err)
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}

		w.WriteHeader(http.StatusOK)
		return
	}

	if db == nil {
		http.Error(w, "database not configured", http.StatusServiceUnavailable)
		return
	}

	// Independent short timeout context so readiness isn't tied to client context,
	// and so it won't hang indefinitely if DB stalls.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		log.Printf("readyz: db ping failed: %v", err)
		http.Error(w, "database not ready", http.StatusServiceUnavailable)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ready"))
}