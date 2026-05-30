package main

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/version"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	otelsdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
)

// initOTELProviders constructs the OpenTelemetry trace, metric, and (when
// log export is enabled) log providers. Returns nil providers when OTEL is
// disabled in the configuration. Fatal errors during exporter setup are
// reported through initFatal so the server can fail fast at startup.
//
// As a side effect, the constructed tracer and meter providers are registered
// as the OTEL globals via otel.SetTracerProvider and otel.SetMeterProvider —
// matching the original inline behavior in runServeCmd.
func initOTELProviders(cfg config.FleetConfig, initFatal func(err error, msg string)) (
	*otelsdklog.LoggerProvider,
	*sdktrace.TracerProvider,
	*sdkmetric.MeterProvider,
) {
	if !cfg.OTELEnabled() {
		return nil, nil, nil
	}

	// Create shared resource with service identification attributes.
	// OTEL_SERVICE_NAME and OTEL_RESOURCE_ATTRIBUTES env vars can override
	// the defaults below.
	res, err := resource.New(context.Background(),
		resource.WithSchemaURL(semconv.SchemaURL),
		resource.WithAttributes(
			semconv.ServiceName("fleet"),
			semconv.ServiceVersion(version.Version().Version),
		),
		resource.WithFromEnv(),
		resource.WithTelemetrySDK(),
	)
	if err != nil {
		initFatal(err, "Failed to create OTEL resource")
		// Returning here makes the function safe even if a caller's
		// initFatal does not terminate (e.g., tests using a recorder).
		return nil, nil, nil
	}

	// Initialize OTEL traces.
	otlpTraceExporter, err := otlptrace.New(context.Background(), otlptracegrpc.NewClient(
		otlptracegrpc.WithCompressor("gzip"),
	))
	if err != nil {
		initFatal(err, "Failed to initialize OTEL trace exporter")
		return nil, nil, nil
	}
	// Configure batch span processor with smaller batch size to avoid exceeding
	// message size limits (4MB default limit).
	batchSpanProcessor := sdktrace.NewBatchSpanProcessor(otlpTraceExporter,
		sdktrace.WithMaxExportBatchSize(256), // Reduce from default 512 to 256
	)
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(batchSpanProcessor),
	)
	otel.SetTracerProvider(tracerProvider)

	// Initialize OTEL metrics.
	metricExporter, err := otlpmetricgrpc.New(context.Background(),
		otlpmetricgrpc.WithCompressor("gzip"),
	)
	if err != nil {
		initFatal(err, "Failed to initialize OTEL metrics exporter")
		return nil, nil, nil
	}

	// Create views to rename otelsql metrics to match what OpenTelemetry Signoz expects.
	// Reference: https://opentelemetry.io/docs/specs/semconv/db/database-metrics/
	dbMetricViews := []sdkmetric.View{
		sdkmetric.NewView(
			sdkmetric.Instrument{Name: "db.sql.connection.open"},
			sdkmetric.Stream{Name: "db.client.connection.count"},
		),
		sdkmetric.NewView(
			sdkmetric.Instrument{Name: "db.sql.connection.max_open"},
			sdkmetric.Stream{Name: "db.client.connection.max"},
		),
		sdkmetric.NewView(
			sdkmetric.Instrument{Name: "db.sql.connection.wait"},
			sdkmetric.Stream{Name: "db.client.connection.wait_count"},
		),
		sdkmetric.NewView(
			sdkmetric.Instrument{Name: "db.sql.connection.wait_duration"},
			sdkmetric.Stream{Name: "db.client.connection.wait_time"},
		),
		sdkmetric.NewView(
			sdkmetric.Instrument{Name: "db.sql.connection.closed_max_idle"},
			sdkmetric.Stream{Name: "db.client.connection.closed.max_idle"},
		),
		sdkmetric.NewView(
			sdkmetric.Instrument{Name: "db.sql.connection.closed_max_idle_time"},
			sdkmetric.Stream{Name: "db.client.connection.closed.max_idle_time"},
		),
	}

	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter)),
		sdkmetric.WithView(dbMetricViews...),
	)
	otel.SetMeterProvider(meterProvider)

	// Initialize OTEL logs.
	var loggerProvider *otelsdklog.LoggerProvider
	if cfg.Logging.OtelLogsEnabled {
		logExporter, err := otlploggrpc.New(context.Background(),
			otlploggrpc.WithCompressor("gzip"),
		)
		if err != nil {
			initFatal(err, "Failed to initialize OTEL log exporter")
			return nil, nil, nil
		}
		loggerProvider = otelsdklog.NewLoggerProvider(
			otelsdklog.WithResource(res),
			otelsdklog.WithProcessor(otelsdklog.NewBatchProcessor(logExporter)),
		)
	}

	return loggerProvider, tracerProvider, meterProvider
}
