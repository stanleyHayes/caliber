// Package httpserver builds the public HTTP surface: a chi router exposing
// health/readiness checks and mounting the grpc-gateway REST handlers.
package httpserver

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/trace"

	"github.com/xcreativs/caliber/internal/app"
	"github.com/xcreativs/caliber/internal/domain/identity"
)

// ReadinessChecker reports whether the app can serve dependency-backed traffic.
type ReadinessChecker interface {
	Check(ctx context.Context) error
}

const corsAllowedHeaders = "Authorization, Content-Type, Connect-Protocol-Version, " +
	"Connect-Timeout, Grpc-Timeout, X-Requested-With"

// maxRequestBodyBytes caps an inbound REST request body (CAL-120). It is sized to
// fit the largest legitimate payload — a 10 MiB CV upload base64-encoded into
// JSON (~13.3 MiB) plus envelope — while still rejecting an unbounded body that
// would otherwise be buffered into memory. It stays at/under the gRPC
// MaxRecvMsgSize the gateway relays into, so a body that passes here is not later
// rejected downstream.
const maxRequestBodyBytes = 16 << 20 // 16 MiB

const (
	bearerPrefix      = "bearer "
	asynqmonCSPHeader = "default-src 'self'; script-src 'self' 'unsafe-inline'; " +
		"style-src 'self' 'unsafe-inline'; img-src 'self' data:; " +
		"connect-src 'self'; frame-ancestors 'none'"
)

// NewRouter builds the chi router: request-id + strict CORS + structured
// access-log + panic-recovery middleware, health and readiness endpoints, and
// the gateway mounted under /v1/. allowedOrigins is the CORS allowlist (empty =
// same-origin only). When log is non-nil, every request is logged with its
// correlation id (CAL-007).
//
//nolint:ireturn // Returns the standard chi.Router interface for mounting.
func NewRouter(
	gateway http.Handler, hsts bool, allowedOrigins []string, log *slog.Logger, readiness ...ReadinessChecker,
) chi.Router {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(cors(allowedOrigins))
	if log != nil {
		r.Use(requestLogger(log))
	}
	r.Use(middleware.Recoverer)
	r.Use(limitBody(maxRequestBodyBytes))
	r.Use(secureHeaders(hsts))
	r.Get("/healthz", health("ok"))
	r.Get("/readyz", ready(readiness...))
	// Instrument the gateway so every REST request becomes an OTel span and the
	// trace context propagates to the downstream gRPC services.
	r.Handle("/v1/*", otelhttp.NewHandler(gateway, "gateway"))
	return r
}

// AIQualityStatsProvider returns a snapshot of AI call quality metrics. It is
// implemented by the LLM memory recorder (CAL-137).
type AIQualityStatsProvider interface {
	Stats() app.AIQualityStats
}

// AIQualityMetrics returns an http.HandlerFunc that serves PII-free AI quality
// metrics (call volume, failure/JSON/refusal rates, guardrail trips, latency)
// as JSON. It is the lightweight surfacing for CAL-137 until CAL-131 wires
// Prometheus exposition format.
func AIQualityMetrics(provider AIQualityStatsProvider) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		stats := provider.Stats()
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(aiQualityResponseFrom(stats)); err != nil {
			// Encoding our own PII-free stats struct should never fail; log and
			// return a minimal error body so the request still terminates.
			http.Error(w, `{"error":"metrics encoding failed"}`, http.StatusInternalServerError)
		}
	}
}

// aiQualityResponse is the JSON shape returned by /metrics. Latencies are in
// milliseconds for human readability.
type aiQualityResponse struct {
	TotalCalls               int                                   `json:"total_calls"`
	FailedCalls              int                                   `json:"failed_calls"`
	FailureRate              float64                               `json:"failure_rate"`
	JSONFailures             int                                   `json:"json_failures"`
	JSONFailureRate          float64                               `json:"json_failure_rate"`
	Refusals                 int                                   `json:"refusals"`
	RefusalRate              float64                               `json:"refusal_rate"`
	GuardrailTrips           int                                   `json:"guardrail_trips"`
	GuardrailTripsByCategory map[string]int                        `json:"guardrail_trips_by_category"`
	P50LatencyMs             int64                                 `json:"p50_latency_ms"`
	P95LatencyMs             int64                                 `json:"p95_latency_ms"`
	InputChars               int                                   `json:"input_chars"`
	OutputChars              int                                   `json:"output_chars"`
	ByOperation              map[string]aiQualityOperationResponse `json:"by_operation"`
}

type aiQualityOperationResponse struct {
	Calls                    int            `json:"calls"`
	Failed                   int            `json:"failed"`
	FailureRate              float64        `json:"failure_rate"`
	JSONFailures             int            `json:"json_failures"`
	JSONFailureRate          float64        `json:"json_failure_rate"`
	Refusals                 int            `json:"refusals"`
	RefusalRate              float64        `json:"refusal_rate"`
	GuardrailTrips           int            `json:"guardrail_trips"`
	GuardrailTripsByCategory map[string]int `json:"guardrail_trips_by_category"`
	P95LatencyMs             int64          `json:"p95_latency_ms"`
}

func aiQualityResponseFrom(s app.AIQualityStats) aiQualityResponse {
	resp := aiQualityResponse{
		TotalCalls:               s.TotalCalls,
		FailedCalls:              s.FailedCalls,
		FailureRate:              s.FailureRate,
		JSONFailures:             s.JSONFailures,
		JSONFailureRate:          s.JSONFailureRate,
		Refusals:                 s.Refusals,
		RefusalRate:              s.RefusalRate,
		GuardrailTrips:           s.GuardrailTrips,
		GuardrailTripsByCategory: s.GuardrailTripsByCategory,
		P50LatencyMs:             s.P50Latency.Milliseconds(),
		P95LatencyMs:             s.P95Latency.Milliseconds(),
		InputChars:               s.InputChars,
		OutputChars:              s.OutputChars,
		ByOperation:              make(map[string]aiQualityOperationResponse, len(s.ByOperation)),
	}
	if resp.GuardrailTripsByCategory == nil {
		resp.GuardrailTripsByCategory = make(map[string]int)
	}
	for op, os := range s.ByOperation {
		resp.ByOperation[op] = aiQualityOperationResponse{
			Calls:                    os.Calls,
			Failed:                   os.Failed,
			FailureRate:              os.FailureRate,
			JSONFailures:             os.JSONFailures,
			JSONFailureRate:          os.JSONFailureRate,
			Refusals:                 os.Refusals,
			RefusalRate:              os.RefusalRate,
			GuardrailTrips:           os.GuardrailTrips,
			GuardrailTripsByCategory: os.GuardrailTripsByCategory,
			P95LatencyMs:             os.P95Latency.Milliseconds(),
		}
	}
	return resp
}

// MountAsynqmon attaches the Asynqmon monitoring UI under the given path,
// protected by bearer-token RBAC so only employer and recruiter principals can
// inspect queue state (CAL-028). The path is normalized to start with a slash
// and requests to the bare path are redirected to the trailing-slash form so
// the UI's relative asset links resolve correctly.
func MountAsynqmon(r chi.Router, path string, handler http.Handler, verifier app.TokenService) {
	path = normalizeMountPath(path)
	guard := Authorize(verifier, identity.RoleEmployer, identity.RoleRecruiter)

	// Redirect /asynqmon -> /asynqmon/ so the SPA's relative URLs work.
	r.Get(path, func(w http.ResponseWriter, req *http.Request) {
		http.Redirect(w, req, path+"/", http.StatusFound)
	})
	r.With(guard).Handle(path+"/*", withAsynqmonCSP(handler))
}

// Authorize returns a chi middleware that verifies a bearer access token and
// enforces that the principal holds one of the allowed roles. It reuses the
// same role model and 401/403 semantics as the gRPC auth guards.
func Authorize(verifier app.TokenService, allowed ...identity.Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw, ok := bearerFromHeader(r)
			if !ok {
				http.Error(w, `{"error":"auth: authentication required"}`, http.StatusUnauthorized)
				return
			}
			principal, err := verifier.VerifyAccess(raw)
			if err != nil {
				http.Error(w, `{"error":"auth: invalid or expired access token"}`, http.StatusUnauthorized)
				return
			}
			for _, role := range allowed {
				if principal.Role == role.String() {
					next.ServeHTTP(w, r)
					return
				}
			}
			http.Error(w, `{"error":"auth: insufficient permissions"}`, http.StatusForbidden)
		})
	}
}

// withAsynqmonCSP relaxes the API's default deny-by-default CSP so the
// Asynqmon SPA can serve its own JS/CSS/images, while keeping the rest of the
// OWASP security headers intact.
func withAsynqmonCSP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", asynqmonCSPHeader)
		next.ServeHTTP(w, r)
	})
}

func normalizeMountPath(path string) string {
	path = strings.TrimSpace(path)
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return strings.TrimSuffix(path, "/")
}

func bearerFromHeader(r *http.Request) (string, bool) {
	auth := r.Header.Get("Authorization")
	if len(auth) <= len(bearerPrefix) {
		return "", false
	}
	if !strings.EqualFold(auth[:len(bearerPrefix)], bearerPrefix) {
		return "", false
	}
	return strings.TrimSpace(auth[len(bearerPrefix):]), true
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
			args := []any{
				slog.String("request_id", middleware.GetReqID(r.Context())),
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", status),
				slog.Int64("duration_ms", time.Since(start).Milliseconds()),
			}
			if span := trace.SpanFromContext(r.Context()); span.SpanContext().IsValid() {
				args = append(args, slog.String("trace_id", span.SpanContext().TraceID().String()))
			}
			log.Info("http_request", args...)
		})
	}
}

// limitBody caps every request body at maxBytes via http.MaxBytesReader, so a
// handler that reads the body (the gateway decoding JSON) sees an error past the
// ceiling instead of buffering an unbounded payload into memory. Bodyless
// requests (health checks, GETs) are unaffected — the reader only trips on read.
func limitBody(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Body != nil {
				r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			}
			next.ServeHTTP(w, r)
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

// cors applies a strict, allowlist-based CORS policy (CAL-114). The SPA is served
// from a different origin than the API, so cross-origin XHR must be explicitly
// permitted — but only for exact, configured origins. A request whose Origin is
// not on the allowlist receives no CORS headers (the browser blocks it); the
// origin is reflected (never "*") and varied on so caches never leak one origin's
// response to another. Preflights (OPTIONS) are answered 204 here.
func cors(allowedOrigins []string) func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		allowed[origin] = struct{}{}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" {
				if _, ok := allowed[origin]; ok {
					h := w.Header()
					h.Set("Access-Control-Allow-Origin", origin)
					h.Add("Vary", "Origin")
					h.Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
					h.Set("Access-Control-Allow-Headers", corsAllowedHeaders)
					h.Set("Access-Control-Max-Age", "600")
				}
			}
			// Answer the preflight here regardless of allow decision: an allowed
			// origin gets the headers above + 204; a disallowed one gets a bare 204
			// with no CORS headers, so the browser blocks the real request.
			if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
				w.WriteHeader(http.StatusNoContent)
				return
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

func ready(checks ...ReadinessChecker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		for _, check := range checks {
			if check == nil {
				continue
			}
			ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
			err := check.Check(ctx)
			cancel()
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusServiceUnavailable)
				_, _ = w.Write([]byte(`{"status":"not_ready"}`))
				return
			}
		}
		health("ready")(w, r)
	}
}
