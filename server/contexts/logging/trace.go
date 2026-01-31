package logging

import (
	"context"

	kitlog "github.com/go-kit/log"
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

// TraceLogger wraps a go-kit logger to automatically inject OTEL trace context
// (trace_id and span_id) into every log call. It reads from the context pointer
// on each Log() call, so it picks up child spans when the context is updated.
type TraceLogger struct {
	ctx    *context.Context
	logger kitlog.Logger
}

// NewTraceLogger creates a logger that automatically injects trace_id and span_id
// from the OTEL trace context. Pass a pointer to your context variable so the
// logger sees updates when child spans are created.
func NewTraceLogger(ctx *context.Context, logger kitlog.Logger) *TraceLogger {
	return &TraceLogger{ctx: ctx, logger: logger}
}

// Log implements kitlog.Logger. It appends trace_id and span_id (if available)
// to the keyvals before delegating to the underlying logger.
func (t *TraceLogger) Log(keyvals ...any) error {
	if tc := getOTELTraceContext(*t.ctx); tc != nil {
		keyvals = append(keyvals, "trace_id", tc.TraceID, "span_id", tc.SpanID)
	}
	return t.logger.Log(keyvals...)
}
