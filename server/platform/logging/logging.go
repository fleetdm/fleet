// Package logging provides structured logging configuration using slog.
// It supports JSON output for production and text output for development,
// with optional OpenTelemetry trace correlation.
package logging

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	otellog "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/trace"
)

// Options configures the slog logger.
type Options struct {
	JSON  bool
	Debug bool
	// Output is the destination for log output. Defaults to os.Stderr.
	Output io.Writer
	// TracingEnabled enables OpenTelemetry trace correlation.
	// When enabled, trace_id and span_id are automatically injected into logs.
	TracingEnabled bool
	// OtelLogsEnabled enables exporting logs to an OpenTelemetry collector.
	// When enabled, logs are sent to both the primary handler (stderr) and OTEL.
	OtelLogsEnabled bool
	// LoggerProvider is the OpenTelemetry LoggerProvider for log export.
	// Required when OtelLogsEnabled is true.
	LoggerProvider otellog.LoggerProvider
}

// NewSlogLogger creates a new slog.Logger with the given options.
// If tracing is enabled, logs are correlated with OpenTelemetry traces.
//
// The handler is configured to maintain backward compatibility with go-kit/log:
//   - Timestamp key is "ts" (not "time")
//   - Timestamp format is RFC3339 (not RFC3339Nano)
//   - Level values are lowercase (e.g., "info" not "INFO")
func NewSlogLogger(opts Options) *slog.Logger {
	output := opts.Output
	if output == nil {
		output = os.Stderr
	}

	level := slog.LevelInfo
	if opts.Debug {
		level = slog.LevelDebug
	}

	handlerOpts := &slog.HandlerOptions{
		Level:       level,
		ReplaceAttr: replaceAttr,
	}

	var handler slog.Handler
	if opts.JSON {
		handler = slog.NewJSONHandler(output, handlerOpts)
	} else {
		handler = slog.NewTextHandler(output, handlerOpts)
	}

	// If tracing is enabled, wrap with handler that injects trace context
	if opts.TracingEnabled {
		handler = NewOtelTracingHandler(handler)
	}

	// If OTEL logs export is enabled, add otelslog handler for sending logs to collector
	if opts.OtelLogsEnabled && opts.LoggerProvider != nil {
		otelHandler := otelslog.NewHandler("fleet", otelslog.WithLoggerProvider(opts.LoggerProvider))
		handler = NewMultiHandler(handler, otelHandler)
	}

	return slog.New(handler)
}

// replaceAttr customizes slog output to maintain backward compatibility
// with go-kit/log format.
func replaceAttr(groups []string, a slog.Attr) slog.Attr {
	// Only modify top-level attributes (not in groups)
	if len(groups) > 0 {
		return a
	}

	switch a.Key {
	case slog.TimeKey:
		// Rename "time" to "ts" and use RFC3339 format
		if t, ok := a.Value.Any().(time.Time); ok {
			return slog.String("ts", t.UTC().Format(time.RFC3339))
		}
	case slog.LevelKey:
		// Convert level to lowercase (INFO -> info, DEBUG -> debug, etc.)
		if lvl, ok := a.Value.Any().(slog.Level); ok {
			return slog.String(slog.LevelKey, strings.ToLower(lvl.String()))
		}
	case slog.MessageKey:
		// Suppress empty messages (go-kit/log didn't print msg when absent)
		if a.Value.String() == "" {
			return slog.Attr{}
		}
	}
	return a
}

// OtelTracingHandler wraps a slog.Handler to inject OpenTelemetry trace context
// (trace_id and span_id) into log records when a span is active in the context.
type OtelTracingHandler struct {
	base slog.Handler
}

// NewOtelTracingHandler creates a new handler that wraps the base handler
// and injects trace context into log records.
func NewOtelTracingHandler(base slog.Handler) *OtelTracingHandler {
	return &OtelTracingHandler{base: base}
}

// Enabled reports whether the handler handles records at the given level.
func (h *OtelTracingHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.base.Enabled(ctx, level)
}

// Handle processes the record, adding trace context if available.
func (h *OtelTracingHandler) Handle(ctx context.Context, r slog.Record) error {
	// Extract span context from the context
	spanCtx := trace.SpanContextFromContext(ctx)
	if spanCtx.IsValid() {
		// Add trace_id and span_id as attributes
		r.AddAttrs(
			slog.String("trace_id", spanCtx.TraceID().String()),
			slog.String("span_id", spanCtx.SpanID().String()),
		)
	}
	return h.base.Handle(ctx, r)
}

// WithAttrs returns a new handler with the given attributes added.
func (h *OtelTracingHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &OtelTracingHandler{base: h.base.WithAttrs(attrs)}
}

// WithGroup returns a new handler with the given group name.
func (h *OtelTracingHandler) WithGroup(name string) slog.Handler {
	return &OtelTracingHandler{base: h.base.WithGroup(name)}
}

// Ensure OtelTracingHandler implements slog.Handler at compile time.
var _ slog.Handler = (*OtelTracingHandler)(nil)
