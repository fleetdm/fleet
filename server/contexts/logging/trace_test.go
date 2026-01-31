package logging

import (
	"bytes"
	"testing"

	kitlog "github.com/go-kit/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

func newTestTracer() trace.Tracer {
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(tracetest.NewSpanRecorder()))
	return tp.Tracer("test")
}

func TestLogWithTraceContext(t *testing.T) {
	t.Run("without span", func(t *testing.T) {
		buf := new(bytes.Buffer)
		lc := &LoggingContext{}
		ctx := NewContext(t.Context(), lc)
		lc.Log(ctx, kitlog.NewLogfmtLogger(buf))

		assert.NotContains(t, buf.String(), "trace_id=")
	})

	t.Run("with span", func(t *testing.T) {
		ctx, span := newTestTracer().Start(t.Context(), "test")
		defer span.End()

		buf := new(bytes.Buffer)
		lc := &LoggingContext{}
		ctx = NewContext(ctx, lc)
		lc.Log(ctx, kitlog.NewLogfmtLogger(buf))

		assert.Contains(t, buf.String(), "trace_id="+span.SpanContext().TraceID().String())
		assert.Contains(t, buf.String(), "span_id="+span.SpanContext().SpanID().String())
	})
}

func TestTraceLogger(t *testing.T) {
	t.Run("without span", func(t *testing.T) {
		ctx := t.Context()
		buf := new(bytes.Buffer)
		logger := NewTraceLogger(&ctx, kitlog.NewLogfmtLogger(buf))
		require.NoError(t, logger.Log("msg", "test"))

		assert.NotContains(t, buf.String(), "trace_id=")
		assert.Contains(t, buf.String(), "msg=test")
	})

	t.Run("with span", func(t *testing.T) {
		ctx, span := newTestTracer().Start(t.Context(), "test")
		defer span.End()

		buf := new(bytes.Buffer)
		logger := NewTraceLogger(&ctx, kitlog.NewLogfmtLogger(buf))
		require.NoError(t, logger.Log("msg", "test"))

		assert.Contains(t, buf.String(), "trace_id="+span.SpanContext().TraceID().String())
		assert.Contains(t, buf.String(), "span_id="+span.SpanContext().SpanID().String())
	})

	t.Run("picks up child span", func(t *testing.T) {
		tracer := newTestTracer()
		ctx, parent := tracer.Start(t.Context(), "parent")
		defer parent.End()

		buf := new(bytes.Buffer)
		logger := NewTraceLogger(&ctx, kitlog.NewLogfmtLogger(buf))

		ctx, child := tracer.Start(ctx, "child")
		defer child.End()

		require.NoError(t, logger.Log("msg", "test"))
		assert.Contains(t, buf.String(), "span_id="+child.SpanContext().SpanID().String())
		assert.NotContains(t, buf.String(), "span_id="+parent.SpanContext().SpanID().String())
	})
}
