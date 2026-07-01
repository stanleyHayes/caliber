package queuemetrics_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xcreativs/caliber/internal/platform/config"
	"github.com/xcreativs/caliber/internal/platform/telemetry"
	"github.com/xcreativs/caliber/internal/platform/telemetry/queuemetrics"
)

func TestRecordEnqueueAndJobProduceMetrics(t *testing.T) {
	// Resetting the global meter provider is not safe in parallel tests, so keep
	// this test serial and set up a real telemetry provider.
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
	queuemetrics.RecordEnqueue(ctx, "candidate_agent_run")
	queuemetrics.RecordJob(ctx, "candidate_agent_run", 150*time.Millisecond, nil)
	queuemetrics.RecordJob(ctx, "interview_scoring", 50*time.Millisecond, assert.AnError)

	rec := httptest.NewRecorder()
	p.PrometheusHandler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	body := rec.Body.String()

	assert.Contains(t, body, "caliber_queue_enqueued_total")
	assert.Contains(t, body, `task_type="candidate_agent_run"`)
	assert.Contains(t, body, "caliber_queue_jobs_processed_total")
	assert.Contains(t, body, `status="ok"`)
	assert.Contains(t, body, `status="error"`)
	assert.Contains(t, body, "caliber_queue_job_duration_seconds")
}

func TestRecordWithoutProviderDoesNotPanic(_ *testing.T) {
	// When no global meter provider is configured, the package should silently
	// drop records rather than panic.
	ctx := context.Background()
	queuemetrics.RecordEnqueue(ctx, "test_task")
	queuemetrics.RecordJob(ctx, "test_task", 10*time.Millisecond, nil)
}
