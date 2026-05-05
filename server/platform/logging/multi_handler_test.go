package logging

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fleetdm/fleet/v4/server/platform/logging/testutils"
)

func TestMultiHandler(t *testing.T) {
	t.Parallel()

	handler1 := testutils.NewTestHandler()
	handler2 := testutils.NewTestHandler()
	logger := slog.New(NewMultiHandler(handler1, handler2))

	logger.InfoContext(t.Context(), "test message", "key", "value")

	record1 := handler1.LastRecord()
	record2 := handler2.LastRecord()
	require.NotNil(t, record1)
	require.NotNil(t, record2)
	assert.Equal(t, "test message", record1.Message)
	assert.Equal(t, "test message", record2.Message)
}
