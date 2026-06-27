package httpserver_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xcreativs/caliber/internal/adapters/inbound/httpserver"
)

func TestSecureHeadersAndHealth(t *testing.T) {
	r := httpserver.NewRouter(http.NotFoundHandler(), true, nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/healthz", nil))

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "nosniff", rec.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", rec.Header().Get("X-Frame-Options"))
	assert.NotEmpty(t, rec.Header().Get("Content-Security-Policy"))
	assert.NotEmpty(t, rec.Header().Get("Referrer-Policy"))
	assert.Equal(t, "max-age=31536000; includeSubDomains", rec.Header().Get("Strict-Transport-Security"))
}

func TestNoHSTSOutsideProd(t *testing.T) {
	r := httpserver.NewRouter(http.NotFoundHandler(), false, nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/readyz", nil))
	assert.Empty(t, rec.Header().Get("Strict-Transport-Security"))
	assert.Equal(t, "nosniff", rec.Header().Get("X-Content-Type-Options"))
}

func TestRequestLoggerEmitsCorrelatedStructuredLog(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&buf, nil))
	r := httpserver.NewRouter(http.NotFoundHandler(), false, log)

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
