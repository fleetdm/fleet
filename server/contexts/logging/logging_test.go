package logging

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"testing"

	"github.com/fleetdm/fleet/v4/server/platform/logging/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoggingErrs(t *testing.T) {
	setupTest := func() (*testutils.TestHandler, *slog.Logger, *LoggingContext, context.Context) {
		handler := testutils.NewTestHandler()
		logger := slog.New(handler)
		lc := &LoggingContext{}
		ctx := NewContext(context.Background(), lc)
		return handler, logger, lc, ctx
	}

	t.Run("one error", func(t *testing.T) {
		handler, logger, lc, ctx := setupTest()

		WithErr(ctx, fmt.Errorf("BLAH: %w", errors.New("AAAA")))
		lc.Log(ctx, logger)
		records := handler.Records()
		require.Len(t, records, 1)
		attrs := testutils.RecordAttrs(&records[0])
		assert.Equal(t, "BLAH: AAAA", attrs["err"])
	})
	t.Run("two errors", func(t *testing.T) {
		handler, logger, lc, ctx := setupTest()

		WithErr(ctx, fmt.Errorf("BLAH: %w", errors.New("AAAA")))
		WithErr(ctx, fmt.Errorf("FOO: %w", errors.New("BBBB")))
		lc.Log(ctx, logger)
		records := handler.Records()
		require.Len(t, records, 1)
		attrs := testutils.RecordAttrs(&records[0])
		assert.Equal(t, "BLAH: AAAA || FOO: BBBB", attrs["err"])
	})
}
