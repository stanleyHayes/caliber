// Package httpserver builds the public HTTP surface: a chi router exposing
// health/readiness checks and mounting the grpc-gateway REST handlers.
package httpserver

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// NewRouter builds the chi router: request-id + panic-recovery middleware,
// health and readiness endpoints, and the gateway mounted under /v1/.
func NewRouter(gateway http.Handler) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)
	r.Get("/healthz", health("ok"))
	r.Get("/readyz", health("ready"))
	r.Handle("/v1/*", gateway)
	return r
}

func health(statusText string) http.HandlerFunc {
	body := []byte(`{"status":"` + statusText + `"}`)
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}
}
