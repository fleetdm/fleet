package logging

import (
	"bytes"
	"testing"

	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestAdapter creates a kitlog adapter with a buffer for capturing output.
func newTestAdapter(t *testing.T) (*bytes.Buffer, kitlog.Logger) {
	t.Helper()
	var buf bytes.Buffer
	slogLogger := NewSlogLogger(Options{
		JSON:   true,
		Debug:  true,
		Output: &buf,
	})
	return &buf, NewKitlogAdapter(slogLogger)
}

func TestKitlogAdapter(t *testing.T) {
	t.Parallel()

	t.Run("basic logging", func(t *testing.T) {
		t.Parallel()
		buf, adapter := newTestAdapter(t)

		err := adapter.Log("msg", "hello world", "key", "value")
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, `"msg":"hello world"`)
		assert.Contains(t, output, `"key":"value"`)
	})

	t.Run("with context via With", func(t *testing.T) {
		t.Parallel()
		buf, adapter := newTestAdapter(t)

		kitlogAdapter, ok := adapter.(*KitlogAdapter)
		require.True(t, ok, "adapter should be *KitlogAdapter")

		contextLogger := kitlogAdapter.With("component", "test-component")
		err := contextLogger.Log("msg", "message with context")
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, `"msg":"message with context"`)
		assert.Contains(t, output, `"component":"test-component"`)
	})

	t.Run("empty log produces no output", func(t *testing.T) {
		t.Parallel()
		buf, adapter := newTestAdapter(t)

		err := adapter.Log()
		require.NoError(t, err)

		assert.Empty(t, buf.String())
	})

	t.Run("skips kitlog timestamp", func(t *testing.T) {
		t.Parallel()
		buf, adapter := newTestAdapter(t)

		// kitlog typically adds "ts" key via kitlog.With()
		// Our adapter skips this since slog adds its own timestamp
		err := adapter.Log("ts", "2024-01-01T00:00:00Z", "msg", "message")
		require.NoError(t, err)

		output := buf.String()
		// slog's timestamp should be present, but not the kitlog value
		assert.Contains(t, output, `"ts":`)
		assert.NotContains(t, output, `"2024-01-01T00:00:00Z"`)
	})
}

func TestKitlogAdapterLevels(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		levelFunc     func(kitlog.Logger) kitlog.Logger
		expectedLevel string
	}{
		{
			name:          "info",
			levelFunc:     level.Info,
			expectedLevel: "info",
		},
		{
			name:          "debug",
			levelFunc:     level.Debug,
			expectedLevel: "debug",
		},
		{
			name:          "warn",
			levelFunc:     level.Warn,
			expectedLevel: "warn",
		},
		{
			name:          "error",
			levelFunc:     level.Error,
			expectedLevel: "error",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			buf, adapter := newTestAdapter(t)

			leveledLogger := tc.levelFunc(adapter)
			err := leveledLogger.Log("msg", tc.name+" message")
			require.NoError(t, err)

			output := buf.String()
			assert.Contains(t, output, `"msg":"`+tc.name+` message"`)
			assert.Contains(t, output, `"level":"`+tc.expectedLevel+`"`)
		})
	}
}
