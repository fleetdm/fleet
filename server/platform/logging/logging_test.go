package logging

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
)

// newTestLogger creates a slog logger with a buffer for capturing output.
func newTestLogger(t *testing.T, opts Options) (*bytes.Buffer, *slog.Logger) {
	t.Helper()
	var buf bytes.Buffer
	opts.Output = &buf
	return &buf, NewSlogLogger(opts)
}

// newTestSpanContext creates a valid span context for testing trace correlation.
func newTestSpanContext(t *testing.T) trace.SpanContext {
	t.Helper()
	traceID, err := trace.TraceIDFromHex("4bf92f3577b34da6a3ce929d0e0e4736")
	require.NoError(t, err)
	spanID, err := trace.SpanIDFromHex("00f067aa0ba902b7")
	require.NoError(t, err)

	return trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled,
	})
}

// newOtelTestLogger creates a logger with OtelHandler wrapping a JSON handler.
func newOtelTestLogger(t *testing.T) (*bytes.Buffer, *slog.Logger) {
	t.Helper()
	var buf bytes.Buffer
	baseHandler := slog.NewJSONHandler(&buf, nil)
	handler := NewOtelHandler(baseHandler)
	return &buf, slog.New(handler)
}

func TestNewSlogLogger(t *testing.T) {
	t.Parallel()

	t.Run("text handler for development", func(t *testing.T) {
		t.Parallel()
		buf, logger := newTestLogger(t, Options{JSON: false, Debug: true})

		logger.Info("test message", "key", "value")
		output := buf.String()

		assert.Contains(t, output, "test message")
		assert.Contains(t, output, "key=value")
		assert.NotContains(t, output, "{", "text format should not use JSON braces")
	})

	t.Run("JSON handler for production", func(t *testing.T) {
		t.Parallel()
		buf, logger := newTestLogger(t, Options{JSON: true})

		logger.Info("test message", "key", "value")
		output := buf.String()

		assert.Contains(t, output, `"msg":"test message"`)
		assert.Contains(t, output, `"key":"value"`)
	})

	t.Run("debug level filtering", func(t *testing.T) {
		t.Parallel()
		buf, logger := newTestLogger(t, Options{Debug: false})

		logger.Debug("debug message")
		logger.Info("info message")
		output := buf.String()

		assert.NotContains(t, output, "debug message")
		assert.Contains(t, output, "info message")
	})

	t.Run("debug level enabled", func(t *testing.T) {
		t.Parallel()
		buf, logger := newTestLogger(t, Options{Debug: true})

		logger.Debug("debug message")

		assert.Contains(t, buf.String(), "debug message")
	})
}

func TestNewSlogLoggerBackwardCompatibility(t *testing.T) {
	t.Parallel()

	t.Run("uses ts key instead of time", func(t *testing.T) {
		t.Parallel()
		buf, logger := newTestLogger(t, Options{JSON: true})

		logger.Info("test")
		output := buf.String()

		assert.Contains(t, output, `"ts":`)
		assert.NotContains(t, output, `"time":`)
	})

	t.Run("uses RFC3339 format without nanoseconds", func(t *testing.T) {
		t.Parallel()
		buf, logger := newTestLogger(t, Options{JSON: true})

		logger.Info("test")
		output := buf.String()

		// RFC3339Nano would have decimal: 2024-01-15T10:00:00.123456789Z
		assert.NotRegexp(t, `"ts":"[^"]+\.[0-9]+Z"`, output)
	})

	t.Run("uses lowercase levels", func(t *testing.T) {
		t.Parallel()
		buf, logger := newTestLogger(t, Options{JSON: true, Debug: true})

		levels := []struct {
			logFunc       func(string, ...any)
			expectedLevel string
		}{
			{logger.Info, "info"},
			{logger.Debug, "debug"},
			{logger.Warn, "warn"},
			{logger.Error, "error"},
		}

		for _, tc := range levels {
			tc.logFunc("test")
		}
		output := buf.String()

		for _, tc := range levels {
			assert.Contains(t, output, `"level":"`+tc.expectedLevel+`"`)
			assert.NotContains(t, output, `"level":"`+strings.ToUpper(tc.expectedLevel)+`"`)
		}
	})
}

func TestOtelHandler(t *testing.T) {
	t.Parallel()

	t.Run("injects trace context when span is active", func(t *testing.T) {
		t.Parallel()
		buf, logger := newOtelTestLogger(t)
		spanCtx := newTestSpanContext(t)
		ctx := trace.ContextWithSpanContext(t.Context(), spanCtx)

		logger.InfoContext(ctx, "traced message")
		output := buf.String()

		assert.Contains(t, output, `"trace_id":"4bf92f3577b34da6a3ce929d0e0e4736"`)
		assert.Contains(t, output, `"span_id":"00f067aa0ba902b7"`)
	})

	t.Run("no trace context without span", func(t *testing.T) {
		t.Parallel()
		buf, logger := newOtelTestLogger(t)

		logger.InfoContext(t.Context(), "untraced message")
		output := buf.String()

		assert.NotContains(t, output, "trace_id")
		assert.NotContains(t, output, "span_id")
	})

	t.Run("preserves tracing with WithAttrs", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		baseHandler := slog.NewJSONHandler(&buf, nil)
		handler := NewOtelHandler(baseHandler).WithAttrs([]slog.Attr{slog.String("component", "test")})
		logger := slog.New(handler)

		spanCtx := newTestSpanContext(t)
		ctx := trace.ContextWithSpanContext(t.Context(), spanCtx)

		logger.InfoContext(ctx, "message")
		output := buf.String()

		assert.Contains(t, output, `"component":"test"`)
		assert.Contains(t, output, `"trace_id"`)
	})

	t.Run("preserves tracing with WithGroup", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		baseHandler := slog.NewJSONHandler(&buf, nil)
		handler := NewOtelHandler(baseHandler).WithGroup("mygroup")
		logger := slog.New(handler)

		spanCtx := newTestSpanContext(t)
		ctx := trace.ContextWithSpanContext(t.Context(), spanCtx)

		logger.InfoContext(ctx, "message", "key", "value")
		output := buf.String()

		assert.Contains(t, output, "mygroup")
		assert.Contains(t, output, `"trace_id"`)
	})
}
