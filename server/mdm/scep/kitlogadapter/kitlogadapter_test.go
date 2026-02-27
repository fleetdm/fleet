package kitlogadapter

import (
	"log/slog"
	"testing"

	"github.com/fleetdm/fleet/v4/server/platform/logging/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLog(t *testing.T) {
	t.Parallel()

	t.Run("basic key-value logging", func(t *testing.T) {
		handler := testutils.NewTestHandler()
		adapter := NewLogger(slog.New(handler))

		err := adapter.Log("msg", "hello world", "key", "value")
		require.NoError(t, err)

		record := handler.LastRecord()
		require.NotNil(t, record)
		assert.Equal(t, "hello world", record.Message)

		attrs := testutils.RecordAttrs(record)
		assert.Equal(t, "value", attrs["key"])
	})

	t.Run("with context via With", func(t *testing.T) {
		handler := testutils.NewTestHandler()
		adapter := NewLogger(slog.New(handler))

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
