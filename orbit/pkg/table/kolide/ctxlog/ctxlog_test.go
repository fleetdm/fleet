package ctxlog

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opencensus.io/trace"
)

func TestIsTraceUninitializedDetectPlatform(t *testing.T) {
	t.Parallel()

	{
		span := trace.FromContext(context.TODO()).SpanContext()
		require.True(t, isTraceUninitialized(span))
	}

	{
		ctx, _ := trace.StartSpan(context.TODO(), "testing")
		span := trace.FromContext(ctx).SpanContext()
		require.False(t, isTraceUninitialized(span))
	}
}
