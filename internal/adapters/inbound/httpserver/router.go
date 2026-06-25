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
func NewRouter(gateway http.Handler, hsts bool) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)
	r.Use(secureHeaders(hsts))
	r.Get("/healthz", health("ok"))
	r.Get("/readyz", health("ready"))
	r.Handle("/v1/*", gateway)
	return r
}

// secureHeaders sets defensive response headers (OWASP secure-headers baseline).
// The surface is a JSON API, so the CSP locks everything down; the SPA is served
// separately. HSTS is only emitted when serving over HTTPS (prod).
func secureHeaders(hsts bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := w.Header()
			h.Set("X-Content-Type-Options", "nosniff")
			h.Set("X-Frame-Options", "DENY")
			h.Set("Referrer-Policy", "no-referrer")
			h.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
			h.Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
			if hsts {
				h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			}
			next.ServeHTTP(w, r)
		})
	}
}

func health(statusText string) http.HandlerFunc {
	body := []byte(`{"status":"` + statusText + `"}`)
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}
}
