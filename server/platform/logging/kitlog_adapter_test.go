package logging

import (
	"log/slog"
	"testing"

	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fleetdm/fleet/v4/server/platform/logging/testutils"
)

// newTestAdapter creates a kitlog adapter with a TestHandler for capturing records.
func newTestAdapter(t *testing.T) (*testutils.TestHandler, kitlog.Logger) {
	t.Helper()
	handler := testutils.NewTestHandler()
	slogLogger := slog.New(handler)
	return handler, NewKitlogAdapter(slogLogger)
}

func TestKitlogAdapter(t *testing.T) {
	t.Parallel()

	t.Run("basic logging", func(t *testing.T) {
		t.Parallel()
		handler, adapter := newTestAdapter(t)

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
		handler, adapter := newTestAdapter(t)

		kitlogAdapter, ok := adapter.(*KitlogAdapter)
		require.True(t, ok, "adapter should be *KitlogAdapter")

		contextLogger := kitlogAdapter.With("component", "test-component")
		err := contextLogger.Log("msg", "message with context")
		require.NoError(t, err)

		record := handler.LastRecord()
		require.NotNil(t, record)
		assert.Equal(t, "message with context", record.Message)

		attrs := testutils.RecordAttrs(record)
		assert.Equal(t, "test-component", attrs["component"])
	})

}

func TestKitlogAdapterLevels(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		levelFunc     func(kitlog.Logger) kitlog.Logger
		expectedLevel slog.Level
	}{
		{
			name:          "info",
			levelFunc:     level.Info,
			expectedLevel: slog.LevelInfo,
		},
		{
			name:          "debug",
			levelFunc:     level.Debug,
			expectedLevel: slog.LevelDebug,
		},
		{
			name:          "warn",
			levelFunc:     level.Warn,
			expectedLevel: slog.LevelWarn,
		},
		{
			name:          "error",
			levelFunc:     level.Error,
			expectedLevel: slog.LevelError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			handler, adapter := newTestAdapter(t)

			leveledLogger := tc.levelFunc(adapter)
			err := leveledLogger.Log("msg", tc.name+" message")
			require.NoError(t, err)

			record := handler.LastRecord()
			require.NotNil(t, record)
			assert.Equal(t, tc.name+" message", record.Message)
			assert.Equal(t, tc.expectedLevel, record.Level)
		})
	}
}
