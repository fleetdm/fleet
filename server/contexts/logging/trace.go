package logging

import (
	"context"

	"go.opentelemetry.io/otel/trace"
)

// traceContext holds trace_id and span_id extracted from OTEL trace context.
type traceContext struct {
	TraceID string
	SpanID  string
}

// getOTELTraceContext extracts trace_id and span_id from the OTEL trace context
// in the given context. Returns nil if no valid trace context is present.
func getOTELTraceContext(ctx context.Context) *traceContext {
	span := trace.SpanFromContext(ctx)
	if span == nil {
		return nil
	}

	spanCtx := span.SpanContext()
	if !spanCtx.IsValid() {
		return nil
	}

	return &traceContext{
		TraceID: spanCtx.TraceID().String(),
		SpanID:  spanCtx.SpanID().String(),
	}
}
