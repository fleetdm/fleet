package logging

import (
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fleetdm/fleet/v4/server/platform/logging/testutils"
)

func TestKitlogAdapter(t *testing.T) {
	t.Parallel()
	handler := testutils.NewTestHandler()
	adapter := NewLogger(slog.New(handler))

	t.Run("basic logging via Log method", func(t *testing.T) {
		t.Parallel()

		err := adapter.Log("msg", "hello world", "key", "value")
		require.NoError(t, err)

		record := handler.LastRecord()
		require.NotNil(t, record)
		assert.Equal(t, "hello world", record.Message)

		attrs := testutils.RecordAttrs(record)
		assert.Equal(t, "value", attrs["key"])
	})

	t.Run("with context via With", func(t *testing.T) {
		t.Parallel()

		contextLogger := adapter.With("component", "test-component")
		err := contextLogger.Log("msg", "message with context")
		require.NoError(t, err)

		record := handler.LastRecord()
		require.NotNil(t, record)
		assert.Equal(t, "message with context", record.Message)

		attrs := testutils.RecordAttrs(record)
		assert.Equal(t, "test-component", attrs["component"])
	})
}

func TestKitlogSlogWrappers(t *testing.T) {
	t.Parallel()
	handler := testutils.NewTestHandler()
	adapter := NewLogger(slog.New(handler))

	tests := []struct {
		name          string
		logFunc       func(ctx context.Context, msg string, keyvals ...any)
		expectedLevel slog.Level
	}{
		{
			name:          "error",
			logFunc:       adapter.ErrorContext,
			expectedLevel: slog.LevelError,
		},
		{
			name:          "warn",
			logFunc:       adapter.WarnContext,
			expectedLevel: slog.LevelWarn,
		},
		{
			name:          "info",
			logFunc:       adapter.InfoContext,
			expectedLevel: slog.LevelInfo,
		},
		{
			name:          "debug",
			logFunc:       adapter.DebugContext,
			expectedLevel: slog.LevelDebug,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.logFunc(t.Context(), tc.name+" message", "key", "value")

			record := handler.LastRecord()
			require.NotNil(t, record)
			assert.Equal(t, tc.name+" message", record.Message)
			assert.Equal(t, tc.expectedLevel, record.Level)

			attrs := testutils.RecordAttrs(record)
			assert.Equal(t, "value", attrs["key"])
		})
	}
}
