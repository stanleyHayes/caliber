package errortracking_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xcreativs/caliber/internal/platform/config"
	"github.com/xcreativs/caliber/internal/platform/telemetry"
	"github.com/xcreativs/caliber/internal/platform/telemetry/errortracking"
)

func TestRecordProducesMetricAndLog(t *testing.T) {
	cfg := config.Config{
		Env:          "test",
		OTelExporter: "noop",
		ServiceName:  "caliber-test",
	}
	p, err := telemetry.New(cfg)
	require.NoError(t, err)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = p.Shutdown(ctx)
	}()

	ctx := context.Background()
	errortracking.Record(ctx, errors.New("boom"), "interview.complete", "llm")

	rec := httptest.NewRecorder()
	p.PrometheusHandler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	body := rec.Body.String()

	assert.Contains(t, body, "caliber_errors_total")
	assert.Contains(t, body, `operation="interview.complete"`)
	assert.Contains(t, body, `class="llm"`)
}

func TestRecordNilErrorIsNoop(_ *testing.T) {
	errortracking.Record(context.Background(), nil, "op", "class")
}
