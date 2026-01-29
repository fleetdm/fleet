package logging

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"

	"github.com/fleetdm/fleet/v4/server/platform/logging/testutils"
)

// newTestLogger creates a slog logger with a buffer for capturing output.
// Use this for tests that need to verify the actual serialized output format.
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

func TestNewSlogLogger(t *testing.T) {
	t.Parallel()

	t.Run("text handler for development", func(t *testing.T) {
		buf, logger := newTestLogger(t, Options{JSON: false, Debug: true})

		logger.InfoContext(t.Context(), "test message", "key", "value")
		output := buf.String()

		assert.Contains(t, output, "test message")
		assert.Contains(t, output, "key=value")
		assert.NotContains(t, output, "{", "text format should not use JSON braces")
	})

	t.Run("JSON handler for production", func(t *testing.T) {
		buf, logger := newTestLogger(t, Options{JSON: true})

		logger.InfoContext(t.Context(), "test message", "key", "value")
		output := buf.String()

		assert.Contains(t, output, `"msg":"test message"`)
		assert.Contains(t, output, `"key":"value"`)
	})

	t.Run("debug level filtering", func(t *testing.T) {
		buf, logger := newTestLogger(t, Options{Debug: false})

		logger.DebugContext(t.Context(), "debug message")
		logger.InfoContext(t.Context(), "info message")
		output := buf.String()

		assert.NotContains(t, output, "debug message")
		assert.Contains(t, output, "info message")
	})

	t.Run("debug level enabled", func(t *testing.T) {
		buf, logger := newTestLogger(t, Options{Debug: true})

		logger.DebugContext(t.Context(), "debug message")

		assert.Contains(t, buf.String(), "debug message")
	})
}

func TestNewSlogLoggerBackwardCompatibility(t *testing.T) {
	t.Parallel()

	t.Run("uses ts key instead of time", func(t *testing.T) {
		buf, logger := newTestLogger(t, Options{JSON: true})

		logger.InfoContext(t.Context(), "test")
		output := buf.String()

		assert.Contains(t, output, `"ts":`)
		assert.NotContains(t, output, `"time":`)
	})

	t.Run("uses RFC3339 format without nanoseconds", func(t *testing.T) {
		buf, logger := newTestLogger(t, Options{JSON: true})

		logger.InfoContext(t.Context(), "test")
		output := buf.String()

		// RFC3339Nano would have decimal: 2024-01-15T10:00:00.123456789Z
		assert.NotRegexp(t, `"ts":"[^"]+\.[0-9]+Z"`, output)
	})

	t.Run("uses lowercase levels", func(t *testing.T) {
		buf, logger := newTestLogger(t, Options{JSON: true, Debug: true})

		ctx := t.Context()
		logger.InfoContext(ctx, "test")
		logger.DebugContext(ctx, "test")
		logger.WarnContext(ctx, "test")
		logger.ErrorContext(ctx, "test")
		output := buf.String()

		for _, level := range []string{"info", "debug", "warn", "error"} {
			assert.Contains(t, output, `"level":"`+level+`"`)
			assert.NotContains(t, output, `"level":"`+strings.ToUpper(level)+`"`)
		}
	})
}

func TestOtelHandler(t *testing.T) {
	t.Parallel()

	t.Run("injects trace context when span is active", func(t *testing.T) {
		testHandler := testutils.NewTestHandler()
		handler := NewOtelHandler(testHandler).
			WithAttrs([]slog.Attr{slog.String("component", "test")}).
			WithGroup("testgroup")
		logger := slog.New(handler)

		spanCtx := newTestSpanContext(t)
		ctx := trace.ContextWithSpanContext(t.Context(), spanCtx)

		logger.InfoContext(ctx, "traced message")

		record := testHandler.LastRecord()
		require.NotNil(t, record)
		assert.Equal(t, "traced message", record.Message)

		attrs := testutils.RecordAttrs(record)
		assert.Equal(t, "4bf92f3577b34da6a3ce929d0e0e4736", attrs["trace_id"])
		assert.Equal(t, "00f067aa0ba902b7", attrs["span_id"])
	})

	t.Run("no trace context without span", func(t *testing.T) {
		testHandler := testutils.NewTestHandler()
		handler := NewOtelHandler(testHandler).
			WithAttrs([]slog.Attr{slog.String("component", "test")}).
			WithGroup("testgroup")
		logger := slog.New(handler)

		logger.InfoContext(t.Context(), "untraced message")

		record := testHandler.LastRecord()
		require.NotNil(t, record)

		attrs := testutils.RecordAttrs(record)
		_, hasTraceID := attrs["trace_id"]
		_, hasSpanID := attrs["span_id"]
		assert.False(t, hasTraceID, "should not have trace_id")
		assert.False(t, hasSpanID, "should not have span_id")
	})
}
