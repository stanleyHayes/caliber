// Package httpserver builds the public HTTP surface: a chi router exposing
// health/readiness checks and mounting the grpc-gateway REST handlers.
package httpserver

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// NewRouter builds the chi router: request-id + structured access-log +
// panic-recovery middleware, health and readiness endpoints, and the gateway
// mounted under /v1/. When log is non-nil, every request is logged with its
// correlation id (CAL-007).
func NewRouter(gateway http.Handler, hsts bool, log *slog.Logger) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	if log != nil {
		r.Use(requestLogger(log))
	}
	r.Use(middleware.Recoverer)
	r.Use(secureHeaders(hsts))
	r.Get("/healthz", health("ok"))
	r.Get("/readyz", health("ready"))
	r.Handle("/v1/*", gateway)
	return r
}

// requestLogger emits one structured log line per request, correlated by the chi
// request id, so every request is traceable end-to-end (CAL-007). It logs only
// method, path, status, and timing — never bodies or query strings — keeping the
// access log PII-free (data-protection.md).
func requestLogger(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			start := time.Now()
			next.ServeHTTP(ww, r)
			status := ww.Status()
			if status == 0 {
				status = http.StatusOK
			}
			log.Info("http_request",
				slog.String("request_id", middleware.GetReqID(r.Context())),
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", status),
				slog.Int64("duration_ms", time.Since(start).Milliseconds()),
			)
		})
	}
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
