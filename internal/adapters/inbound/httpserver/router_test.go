package httpserver_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xcreativs/caliber/internal/adapters/inbound/httpserver"
	"github.com/xcreativs/caliber/internal/app"
)

func TestSecureHeadersAndHealth(t *testing.T) {
	r := httpserver.NewRouter(http.NotFoundHandler(), true, nil, nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/healthz", nil))

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "nosniff", rec.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", rec.Header().Get("X-Frame-Options"))
	assert.Equal(t, "default-src 'none'; frame-ancestors 'none'", rec.Header().Get("Content-Security-Policy"))
	assert.Equal(t, "no-referrer", rec.Header().Get("Referrer-Policy"))
	assert.Equal(t, "geolocation=(), microphone=(), camera=()", rec.Header().Get("Permissions-Policy"))
	assert.Equal(t, "max-age=31536000; includeSubDomains", rec.Header().Get("Strict-Transport-Security"))
}

func TestLimitsRequestBody(t *testing.T) {
	// The gateway handler fully reads the body; http.MaxBytesReader makes that read
	// fail once the 16 MiB cap is crossed, so an unbounded body is never buffered.
	echo := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := io.ReadAll(r.Body); err != nil {
			http.Error(w, "too large", http.StatusRequestEntityTooLarge)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	r := httpserver.NewRouter(echo, false, nil, nil)

	// A modest body passes through.
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequestWithContext(
		t.Context(), http.MethodPost, "/v1/anything", bytes.NewReader(make([]byte, 1024))))
	assert.Equal(t, http.StatusOK, rec.Code)

	// One byte past the cap is rejected.
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequestWithContext(
		t.Context(), http.MethodPost, "/v1/anything", bytes.NewReader(make([]byte, (16<<20)+1))))
	assert.Equal(t, http.StatusRequestEntityTooLarge, rec.Code)
}

func TestNoHSTSOutsideProd(t *testing.T) {
	r := httpserver.NewRouter(http.NotFoundHandler(), false, nil, nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/readyz", nil))
	assert.Empty(t, rec.Header().Get("Strict-Transport-Security"))
	assert.Equal(t, "nosniff", rec.Header().Get("X-Content-Type-Options"))
}

func TestCORSAllowsConfiguredOrigin(t *testing.T) {
	const origin = "https://app.caliber.example"
	r := httpserver.NewRouter(http.NotFoundHandler(), false, []string{origin}, nil)

	// A real cross-origin request from an allowlisted origin gets the origin
	// reflected (never "*") and is varied on, so caches never leak across origins.
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/healthz", nil)
	req.Header.Set("Origin", origin)
	r.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, origin, rec.Header().Get("Access-Control-Allow-Origin"))
	assert.Contains(t, rec.Header().Values("Vary"), "Origin")
}

func TestCORSPreflightFromAllowedOrigin(t *testing.T) {
	const origin = "https://app.caliber.example"
	r := httpserver.NewRouter(http.NotFoundHandler(), false, []string{origin}, nil)

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodOptions, "/v1/roles", nil)
	req.Header.Set("Origin", origin)
	req.Header.Set("Access-Control-Request-Method", "POST")
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Code, "preflight is answered here, not by the gateway")
	assert.Equal(t, origin, rec.Header().Get("Access-Control-Allow-Origin"))
	assert.Contains(t, rec.Header().Get("Access-Control-Allow-Methods"), "POST")
	assert.Contains(t, rec.Header().Get("Access-Control-Allow-Headers"), "Authorization")
}

func TestCORSRejectsUnlistedOrigin(t *testing.T) {
	r := httpserver.NewRouter(http.NotFoundHandler(), false, []string{"https://app.caliber.example"}, nil)

	// An origin not on the allowlist gets NO CORS headers — the browser then
	// blocks the response. The preflight still returns a bare 204.
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodOptions, "/v1/roles", nil)
	req.Header.Set("Origin", "https://evil.example")
	req.Header.Set("Access-Control-Request-Method", "POST")
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Empty(t, rec.Header().Get("Access-Control-Allow-Origin"), "an unlisted origin is never reflected")
}

func TestReadyzReportsDependencyFailure(t *testing.T) {
	r := httpserver.NewRouter(http.NotFoundHandler(), false, nil, nil, failingReadiness{})

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/readyz", nil))

	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
	assert.JSONEq(t, `{"status":"not_ready"}`, rec.Body.String())
}

func TestRequestLoggerEmitsCorrelatedStructuredLog(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&buf, nil))
	r := httpserver.NewRouter(http.NotFoundHandler(), false, nil, log)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/healthz", nil))
	require.Equal(t, http.StatusOK, rec.Code)

	var entry map[string]any
	require.NoError(t, json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &entry), "one structured log line is emitted")
	assert.Equal(t, "http_request", entry["msg"])
	assert.Equal(t, http.MethodGet, entry["method"])
	assert.Equal(t, "/healthz", entry["path"])
	status, ok := entry["status"].(float64) // JSON numbers decode to float64
	require.True(t, ok, "status is logged")
	assert.Equal(t, http.StatusOK, int(status))
	assert.NotEmpty(t, entry["request_id"], "every request is correlated by its chi request id (CAL-007)")
	assert.Contains(t, entry, "duration_ms")
}

func TestAIQualityMetricsServesJSONStats(t *testing.T) {
	provider := &staticStatsProvider{}
	handler := httpserver.AIQualityMetrics(provider)

	rec := httptest.NewRecorder()
	handler(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/metrics", nil))

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.InDelta(t, 0.0, body["total_calls"], 1e-9)
	assert.Contains(t, body, "by_operation")
}

type staticStatsProvider struct{}

func (s *staticStatsProvider) Stats() app.AIQualityStats { return app.AIQualityStats{} }

type failingReadiness struct{}

func (failingReadiness) Check(context.Context) error { return errors.New("down") }
