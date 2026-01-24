package logging

import (
	"bytes"
	"testing"

	kitlog "github.com/go-kit/log"
	"github.com/stretchr/testify/assert"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestLogWithTraceContext(t *testing.T) {
	t.Run("without span", func(t *testing.T) {
		buf := new(bytes.Buffer)
		lc := &LoggingContext{}
		ctx := NewContext(t.Context(), lc)

		lc.Log(ctx, kitlog.NewLogfmtLogger(buf))

		assert.NotContains(t, buf.String(), "trace_id=")
		assert.NotContains(t, buf.String(), "span_id=")
	})

	t.Run("with span", func(t *testing.T) {
		tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(tracetest.NewSpanRecorder()))
		ctx, span := tp.Tracer("test").Start(t.Context(), "test-span")
		defer span.End()

		buf := new(bytes.Buffer)
		lc := &LoggingContext{}
		ctx = NewContext(ctx, lc)

		lc.Log(ctx, kitlog.NewLogfmtLogger(buf))

		assert.Contains(t, buf.String(), "trace_id="+span.SpanContext().TraceID().String())
		assert.Contains(t, buf.String(), "span_id="+span.SpanContext().SpanID().String())
	})
}
