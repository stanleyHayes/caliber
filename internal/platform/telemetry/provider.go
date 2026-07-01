// Package telemetry constructs the OpenTelemetry tracer and meter providers used
// by the API and worker processes. It lives in platform so domain/app never import
// infrastructure, and it exposes a thin port-like surface for use-cases that want
// business counters.
package telemetry

import (
	"context"
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	otelprom "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"

	"github.com/xcreativs/caliber/internal/platform/config"
)

// Provider owns the OTel tracer and meter providers plus the Prometheus gatherer.
type Provider struct {
	tracerProvider *sdktrace.TracerProvider
	meterProvider  *sdkmetric.MeterProvider
	registry       *prometheus.Registry
	promHandler    http.Handler
}

// New builds a telemetry provider from configuration. It installs the global
// tracer and meter providers and a W3C trace-context propagator. The Prometheus
// metrics endpoint is always available, regardless of the configured trace exporter.
func New(cfg config.Config) (*Provider, error) {
	res := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceNameKey.String(cfg.ServiceName),
		semconv.ServiceVersionKey.String(cfg.ServiceVersion),
		semconv.DeploymentEnvironmentKey.String(cfg.Env),
	)

	tracerProvider, err := newTracerProvider(cfg, res)
	if err != nil {
		return nil, err
	}

	// Each Provider owns its own Prometheus registry so tests and multiple
	// processes do not collide on the default registerer.
	registry := prometheus.NewRegistry()
	promExporter, err := otelprom.New(otelprom.WithRegisterer(registry))
	if err != nil {
		return nil, fmt.Errorf("telemetry: prometheus exporter: %w", err)
	}

	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(promExporter),
		sdkmetric.WithResource(res),
	)

	otel.SetTracerProvider(tracerProvider)
	otel.SetMeterProvider(meterProvider)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	return &Provider{
		tracerProvider: tracerProvider,
		meterProvider:  meterProvider,
		registry:       registry,
		promHandler:    promhttp.HandlerFor(registry, promhttp.HandlerOpts{Registry: registry}),
	}, nil
}

func newTracerProvider(cfg config.Config, res *resource.Resource) (*sdktrace.TracerProvider, error) {
	switch cfg.OTelExporter {
	case "stdout":
		exp, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
		if err != nil {
			return nil, fmt.Errorf("telemetry: stdout trace exporter: %w", err)
		}
		return sdktrace.NewTracerProvider(
			sdktrace.WithBatcher(exp),
			sdktrace.WithResource(res),
		), nil
	case "noop", "":
		return sdktrace.NewTracerProvider(sdktrace.WithResource(res)), nil
	default:
		return nil, fmt.Errorf("telemetry: unsupported exporter %q", cfg.OTelExporter)
	}
}

// Tracer returns a named tracer from the provider's tracer provider.
//nolint:ireturn // returns the standard OTel tracer interface by design.
func (p *Provider) Tracer(name string) trace.Tracer {
	return p.tracerProvider.Tracer(name)
}

// Meter returns a named meter from the provider's meter provider.
//nolint:ireturn // returns the standard OTel meter interface by design.
func (p *Provider) Meter(name string) metric.Meter {
	return p.meterProvider.Meter(name)
}

// PrometheusHandler serves Prometheus exposition format metrics.
func (p *Provider) PrometheusHandler() http.Handler {
	return p.promHandler
}

// Shutdown flushes and stops the providers. It should be called on process exit.
func (p *Provider) Shutdown(ctx context.Context) error {
	var firstErr error
	if err := p.tracerProvider.Shutdown(ctx); err != nil {
		firstErr = fmt.Errorf("telemetry: shutdown tracer: %w", err)
	}
	if err := p.meterProvider.Shutdown(ctx); err != nil && firstErr == nil {
		firstErr = fmt.Errorf("telemetry: shutdown meter: %w", err)
	}
	return firstErr
}
