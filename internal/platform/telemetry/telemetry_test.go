package telemetry_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xcreativs/caliber/internal/app"
	"github.com/xcreativs/caliber/internal/platform/config"
	"github.com/xcreativs/caliber/internal/platform/telemetry"
)

func noopConfig() config.Config {
	return config.Config{
		Env:            "test",
		OTelExporter:   "noop",
		ServiceName:    "caliber-test",
		ServiceVersion: "0.0.0",
	}
}

func TestNewProviderNoopBuildsAndShutsDown(t *testing.T) {
	p, err := telemetry.New(noopConfig())
	require.NoError(t, err)
	require.NotNil(t, p)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	require.NoError(t, p.Shutdown(ctx))
}

func TestNewProviderStdoutBuilds(t *testing.T) {
	cfg := noopConfig()
	cfg.OTelExporter = "stdout"
	p, err := telemetry.New(cfg)
	require.NoError(t, err)
	require.NotNil(t, p)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	require.NoError(t, p.Shutdown(ctx))
}

func TestNewProviderRejectsUnknownExporter(t *testing.T) {
	cfg := noopConfig()
	cfg.OTelExporter = "unknown"
	p, err := telemetry.New(cfg)
	require.Error(t, err)
	require.Nil(t, p)
}

func TestPrometheusHandlerExposesMetrics(t *testing.T) {
	p, err := telemetry.New(noopConfig())
	require.NoError(t, err)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = p.Shutdown(ctx)
	}()

	rec := httptest.NewRecorder()
	p.PrometheusHandler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))

	assert.Equal(t, 200, rec.Code)
	assert.True(t, strings.HasPrefix(rec.Header().Get("Content-Type"), "text/plain"), "expected text/plain prometheus output")
	body := rec.Body.String()
	assert.Contains(t, body, "target_info")
}

func TestProviderTracerAndMeter(t *testing.T) {
	p, err := telemetry.New(noopConfig())
	require.NoError(t, err)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = p.Shutdown(ctx)
	}()

	require.NotNil(t, p.Tracer("test"))
	require.NotNil(t, p.Meter("test"))
}

func TestProviderShutdownTwiceReturnsError(t *testing.T) {
	p, err := telemetry.New(noopConfig())
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	require.NoError(t, p.Shutdown(ctx))

	// A second shutdown is expected to report that the provider is already shut down.
	require.Error(t, p.Shutdown(ctx))
}

func TestAIMetricsRecorderExposesAIQualityMetrics(t *testing.T) {
	p, err := telemetry.New(noopConfig())
	require.NoError(t, err)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = p.Shutdown(ctx)
	}()

	recorder, err := telemetry.NewAIMetricsRecorder(p)
	require.NoError(t, err)

	recorder.Record(app.AICallRecord{
		Operation:     "interview_report",
		Model:         "dev",
		Latency:       150 * time.Millisecond,
		PromptChars:   100,
		ResponseChars: 200,
		Failed:        true,
		JSONFailure:   true,
		Refusal:       true,
		GuardrailTrips: []string{"delimiter_breakout"},
	})

	r := httptest.NewRecorder()
	p.PrometheusHandler().ServeHTTP(r, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	body := r.Body.String()

	assert.Contains(t, body, "caliber_ai_calls_total")
	assert.Contains(t, body, "caliber_ai_calls_failed_total")
	assert.Contains(t, body, "caliber_ai_calls_json_failures_total")
	assert.Contains(t, body, "caliber_ai_calls_refusals_total")
	assert.Contains(t, body, "caliber_ai_calls_guardrail_trips_total")
	assert.Contains(t, body, "caliber_ai_input_chars_total")
	assert.Contains(t, body, "caliber_ai_output_chars_total")
	assert.Contains(t, body, "caliber_ai_call_duration_seconds")
	assert.Contains(t, body, `operation="interview_report"`)
}
