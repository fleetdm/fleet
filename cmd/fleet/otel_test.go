package main

import (
	"context"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/platform/tracing"
	otelsdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"github.com/stretchr/testify/require"
)

// shutdownOTELProviders releases the background goroutines and exporters held
// by the providers initOTELProviders constructs. A short context timeout keeps
// the cleanup fast even when no OTLP collector is listening — the periodic
// exporters would otherwise block trying to flush in-flight batches.
func shutdownOTELProviders(t *testing.T, lp *otelsdklog.LoggerProvider, tp *sdktrace.TracerProvider, mp *sdkmetric.MeterProvider) {
	t.Helper()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()
		if tp != nil {
			_ = tp.Shutdown(ctx)
		}
		if mp != nil {
			_ = mp.Shutdown(ctx)
		}
		if lp != nil {
			_ = lp.Shutdown(ctx)
		}
	})
}

func TestInitOTELProviders_DisabledReturnsNilProviders(t *testing.T) {
	// Default config has OTEL disabled. The init function should be a no-op
	// and return nil providers without calling initFatal.
	cfg := config.FleetConfig{}
	require.False(t, cfg.OTELEnabled(), "precondition: default config must have OTEL disabled")

	called := false
	lp, tp, mp, sampler := initOTELProviders(cfg, tracing.NewRegistry(), func(err error, msg string) { called = true })

	require.False(t, called, "initFatal must not be called when OTEL is disabled")
	require.Nil(t, lp)
	require.Nil(t, tp)
	require.Nil(t, mp)
	require.Nil(t, sampler, "sampler must be nil when OTEL is disabled")
}

func TestInitOTELProviders_EnabledReturnsTracerAndMeterProviders(t *testing.T) {
	// When OTEL is enabled but OtelLogsEnabled is false, the trace and meter
	// providers should be constructed and the logger provider should remain
	// nil. The OTLP exporter constructors don't dial at construction time, so
	// this is safe to run without a collector.
	cfg := config.FleetConfig{
		Logging: config.LoggingConfig{TracingEnabled: true},
	}
	require.True(t, cfg.OTELEnabled(), "precondition: tracing-enabled config must have OTEL enabled")

	called := false
	lp, tp, mp, sampler := initOTELProviders(cfg, tracing.NewRegistry(), func(err error, msg string) { called = true })
	shutdownOTELProviders(t, lp, tp, mp)

	require.False(t, called, "initFatal must not be called for a healthy enabled config")
	require.Nil(t, lp, "logger provider should be nil when OtelLogsEnabled is false")
	require.NotNil(t, tp)
	require.NotNil(t, mp)
	require.NotNil(t, sampler, "sampler must be returned when OTEL is enabled")
}

func TestInitOTELProviders_LogExportEnabledReturnsLoggerProvider(t *testing.T) {
	cfg := config.FleetConfig{
		Logging: config.LoggingConfig{TracingEnabled: true, OtelLogsEnabled: true},
	}

	called := false
	lp, tp, mp, sampler := initOTELProviders(cfg, tracing.NewRegistry(), func(err error, msg string) { called = true })
	shutdownOTELProviders(t, lp, tp, mp)

	require.False(t, called)
	require.NotNil(t, lp, "logger provider should be set when OtelLogsEnabled is true")
	require.NotNil(t, tp)
	require.NotNil(t, mp)
	require.NotNil(t, sampler)
}
