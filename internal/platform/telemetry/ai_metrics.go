package telemetry

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/xcreativs/caliber/internal/app"
)

const aiRecorderName = "github.com/xcreativs/caliber/internal/platform/telemetry/ai"

// AIMetricsRecorder turns redacted AI call records into Prometheus metrics. It
// implements app.AICallRecorder so it can be composed with the existing memory
// and slog recorders.
type AIMetricsRecorder struct {
	callsTotal         metric.Int64Counter
	failuresTotal      metric.Int64Counter
	jsonFailuresTotal  metric.Int64Counter
	refusalsTotal      metric.Int64Counter
	guardrailTripsTotal metric.Int64Counter
	inputCharsTotal    metric.Int64Counter
	outputCharsTotal   metric.Int64Counter
	durationSeconds    metric.Float64Histogram
}

// NewAIMetricsRecorder builds the recorder backed by the provider's meter.
func NewAIMetricsRecorder(p *Provider) (*AIMetricsRecorder, error) {
	m := p.Meter(aiRecorderName)
	var r AIMetricsRecorder
	var err error

	if r.callsTotal, err = m.Int64Counter("caliber.ai.calls.total", metric.WithDescription("Total AI model calls")); err != nil {
		return nil, fmt.Errorf("telemetry: ai calls counter: %w", err)
	}
	if r.failuresTotal, err = m.Int64Counter("caliber.ai.calls.failed.total", metric.WithDescription("Failed AI model calls")); err != nil {
		return nil, fmt.Errorf("telemetry: ai failures counter: %w", err)
	}
	if r.jsonFailuresTotal, err = m.Int64Counter(
		"caliber.ai.calls.json_failures.total",
		metric.WithDescription("AI structured-output JSON failures"),
	); err != nil {
		return nil, fmt.Errorf("telemetry: ai json failures counter: %w", err)
	}
	if r.refusalsTotal, err = m.Int64Counter(
		"caliber.ai.calls.refusals.total",
		metric.WithDescription("AI refusal responses"),
	); err != nil {
		return nil, fmt.Errorf("telemetry: ai refusals counter: %w", err)
	}
	if r.guardrailTripsTotal, err = m.Int64Counter(
		"caliber.ai.calls.guardrail_trips.total",
		metric.WithDescription("AI guardrail trips"),
	); err != nil {
		return nil, fmt.Errorf("telemetry: ai guardrail trips counter: %w", err)
	}
	if r.inputCharsTotal, err = m.Int64Counter(
		"caliber.ai.input_chars.total",
		metric.WithDescription("Total AI prompt characters"),
	); err != nil {
		return nil, fmt.Errorf("telemetry: ai input chars counter: %w", err)
	}
	if r.outputCharsTotal, err = m.Int64Counter(
		"caliber.ai.output_chars.total",
		metric.WithDescription("Total AI response characters"),
	); err != nil {
		return nil, fmt.Errorf("telemetry: ai output chars counter: %w", err)
	}
	if r.durationSeconds, err = m.Float64Histogram("caliber.ai.call.duration.seconds",
		metric.WithDescription("AI call duration"),
		metric.WithUnit("s"),
	); err != nil {
		return nil, fmt.Errorf("telemetry: ai duration histogram: %w", err)
	}

	return &r, nil
}

// Record updates all AI-quality metrics from a redacted call record.
func (r *AIMetricsRecorder) Record(rec app.AICallRecord) {
	if r == nil {
		return
	}
	attrs := metric.WithAttributes(
		attribute.String("operation", rec.Operation),
		attribute.String("model", rec.Model),
	)

	r.callsTotal.Add(context.Background(), 1, attrs)
	r.inputCharsTotal.Add(context.Background(), int64(rec.PromptChars), attrs)
	r.outputCharsTotal.Add(context.Background(), int64(rec.ResponseChars), attrs)
	r.durationSeconds.Record(context.Background(), rec.Latency.Seconds(), attrs)

	if rec.Failed {
		r.failuresTotal.Add(context.Background(), 1, attrs)
	}
	if rec.JSONFailure {
		r.jsonFailuresTotal.Add(context.Background(), 1, attrs)
	}
	if rec.Refusal {
		r.refusalsTotal.Add(context.Background(), 1, attrs)
	}
	if len(rec.GuardrailTrips) > 0 {
		r.guardrailTripsTotal.Add(context.Background(), int64(len(rec.GuardrailTrips)), attrs)
	}
}
