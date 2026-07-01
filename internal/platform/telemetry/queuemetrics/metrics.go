// Package queuemetrics exposes queue/job metrics using the global OpenTelemetry
// meter provider. It records only task type and outcome — no payloads or PII.
package queuemetrics

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

const meterName = "github.com/xcreativs/caliber/internal/platform/telemetry/queuemetrics"

// Recorder holds the queue/job instruments.
type Recorder struct {
	enqueued        metric.Int64Counter
	processed       metric.Int64Counter
	durationSeconds metric.Float64Histogram
}

//nolint:gochecknoglobals // package-level metric recorder cache for convenience API.
var (
	recOnce sync.Once
	rec     *Recorder
)

func recorder() *Recorder {
	recOnce.Do(func() {
		var err error
		rec, err = New(otel.Meter(meterName))
		if err != nil {
			// Metrics are best-effort; fall back to a no-op recorder.
			rec = &Recorder{}
		}
	})
	return rec
}

// New creates a Recorder from the supplied meter. Callers normally do not need
// this; use the package-level RecordEnqueue/RecordJob functions instead.
func New(m metric.Meter) (*Recorder, error) {
	r := &Recorder{}
	var err error
	if r.enqueued, err = m.Int64Counter("caliber.queue.enqueued.total",
		metric.WithDescription("Total tasks enqueued")); err != nil {
		return nil, fmt.Errorf("queuemetrics: enqueued counter: %w", err)
	}
	if r.processed, err = m.Int64Counter("caliber.queue.jobs.processed.total",
		metric.WithDescription("Total jobs processed")); err != nil {
		return nil, fmt.Errorf("queuemetrics: processed counter: %w", err)
	}
	if r.durationSeconds, err = m.Float64Histogram("caliber.queue.job.duration.seconds",
		metric.WithDescription("Job processing duration"),
		metric.WithUnit("s"),
	); err != nil {
		return nil, fmt.Errorf("queuemetrics: duration histogram: %w", err)
	}
	return r, nil
}

// RecordEnqueue records one enqueued task.
func RecordEnqueue(ctx context.Context, taskType string) {
	if r := recorder(); r != nil && r.enqueued != nil {
		r.enqueued.Add(ctx, 1, metric.WithAttributes(attribute.String("task_type", taskType)))
	}
}

// RecordJob records one processed job, including its duration and outcome.
func RecordJob(ctx context.Context, taskType string, d time.Duration, err error) {
	if r := recorder(); r != nil {
		status := "ok"
		if err != nil {
			status = "error"
		}
		attrs := metric.WithAttributes(
			attribute.String("task_type", taskType),
			attribute.String("status", status),
		)
		if r.processed != nil {
			r.processed.Add(ctx, 1, attrs)
		}
		if r.durationSeconds != nil {
			r.durationSeconds.Record(ctx, d.Seconds(), attrs)
		}
	}
}
