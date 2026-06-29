package jobs_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"sync/atomic"
	"testing"

	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/xcreativs/caliber/internal/adapters/inbound/jobs"
)

func TestJobFrameworkSkipsDuplicateDelivery(t *testing.T) {
	store := jobs.NewMemoryIdempotencyStore()
	var buf bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&buf, nil))
	var calls int32
	mux := jobs.NewMux(log,
		jobs.WithIdempotencyStore(store),
		jobs.WithHealthcheckCallback(func(context.Context, jobs.HealthcheckPayload) error {
			atomic.AddInt32(&calls, 1)
			return nil
		}),
	)
	payload, err := jobs.EncodeHealthcheckPayload(jobs.HealthcheckPayload{Probe: "same"})
	require.NoError(t, err)

	require.NoError(t, mux.ProcessTask(t.Context(), asynq.NewTask(jobs.TypeHealthcheck, payload)))
	require.NoError(t, mux.ProcessTask(t.Context(), asynq.NewTask(jobs.TypeHealthcheck, payload)))

	assert.Equal(t, int32(1), atomic.LoadInt32(&calls), "duplicate delivery must not double-apply")
	assert.Contains(t, buf.String(), "job completed")
	assert.Contains(t, buf.String(), "job skipped duplicate")
}

func TestJobFrameworkReleasesFailedDeliveryForRetry(t *testing.T) {
	store := jobs.NewMemoryIdempotencyStore()
	var calls int32
	mux := jobs.NewMux(slog.New(slog.DiscardHandler),
		jobs.WithIdempotencyStore(store),
		jobs.WithHealthcheckCallback(func(context.Context, jobs.HealthcheckPayload) error {
			if atomic.AddInt32(&calls, 1) == 1 {
				return errors.New("transient failure")
			}
			return nil
		}),
	)
	payload, err := jobs.EncodeHealthcheckPayload(jobs.HealthcheckPayload{Probe: "retry"})
	require.NoError(t, err)

	err = mux.ProcessTask(t.Context(), asynq.NewTask(jobs.TypeHealthcheck, payload))
	require.ErrorContains(t, err, "transient failure")
	require.NoError(t, mux.ProcessTask(t.Context(), asynq.NewTask(jobs.TypeHealthcheck, payload)))

	assert.Equal(t, int32(2), atomic.LoadInt32(&calls), "failed delivery should release the key for retry")
}

func TestJobFrameworkEmitsTraceSpan(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	previous := otel.GetTracerProvider()
	otel.SetTracerProvider(provider)
	t.Cleanup(func() {
		otel.SetTracerProvider(previous)
		require.NoError(t, provider.Shutdown(context.Background()))
	})

	mux := jobs.NewMux(slog.New(slog.DiscardHandler),
		jobs.WithHealthcheckCallback(func(context.Context, jobs.HealthcheckPayload) error {
			return nil
		}),
	)
	payload, err := jobs.EncodeHealthcheckPayload(jobs.HealthcheckPayload{Probe: "trace"})
	require.NoError(t, err)

	require.NoError(t, mux.ProcessTask(t.Context(), asynq.NewTask(jobs.TypeHealthcheck, payload)))

	spans := recorder.Ended()
	require.Len(t, spans, 1)
	assert.Equal(t, "jobs."+jobs.TypeHealthcheck, spans[0].Name())
	assert.Equal(t, jobs.TypeHealthcheck, spanStringAttr(spans[0], "job.type"))
	assert.NotEmpty(t, spanStringAttr(spans[0], "job.idempotency_key"))
}

func spanStringAttr(span sdktrace.ReadOnlySpan, key string) string {
	for _, attr := range span.Attributes() {
		if attr.Key == attribute.Key(key) {
			return attr.Value.AsString()
		}
	}
	return ""
}
