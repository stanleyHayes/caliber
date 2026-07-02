// Package errortracking emits a unified structured error event and counter for
// operational errors. It groups failures by operation and class so on-call can
// triage from logs and dashboards without needing to grep every source file.
package errortracking

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

const meterName = "github.com/xcreativs/caliber/internal/platform/telemetry/errortracking"

// Recorder emits error metrics and structured log lines.
type Recorder struct {
	errorsTotal metric.Int64Counter
}

//nolint:gochecknoglobals // package-level recorder cache for convenience API.
var (
	recOnce sync.Once
	rec     *Recorder
)

func recorder() *Recorder {
	recOnce.Do(func() {
		var err error
		rec, err = New(otel.Meter(meterName))
		if err != nil {
			// Error tracking is best-effort; fall back to no-op.
			rec = &Recorder{}
		}
	})
	return rec
}

// New creates a Recorder from the supplied meter.
func New(m metric.Meter) (*Recorder, error) {
	counter, err := m.Int64Counter("caliber.errors.total",
		metric.WithDescription("Total operational errors by operation and class"))
	if err != nil {
		return nil, fmt.Errorf("errortracking: errors counter: %w", err)
	}
	return &Recorder{errorsTotal: counter}, nil
}

// Record emits one error event. It is a no-op when err is nil.
func Record(ctx context.Context, err error, operation, class string) {
	if err == nil {
		return
	}
	if r := recorder(); r != nil && r.errorsTotal != nil {
		r.errorsTotal.Add(ctx, 1, metric.WithAttributes(
			attribute.String("operation", operation),
			attribute.String("class", class),
		))
	}

	args := []any{
		slog.String("error", err.Error()),
		slog.String("operation", operation),
		slog.String("class", class),
	}
	if span := trace.SpanFromContext(ctx); span.SpanContext().IsValid() {
		args = append(args, slog.String("trace_id", span.SpanContext().TraceID().String()))
	}
	slog.Default().Error("operational_error", args...)
}
