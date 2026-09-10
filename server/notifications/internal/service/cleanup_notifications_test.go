package service

import (
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCleanupNotifications(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.DiscardHandler)
	queue := &mockScriptQueue{}

	t.Run("expired notifications are deleted a batch at a time until a short batch ends the run", func(t *testing.T) {
		ds := &mockDatastore{deletable: endUserNotificationCleanupBatchSize*2 + 10}

		require.NoError(t, NewService(ds, queue, logger).CleanupNotifications(ctx))

		assert.Zero(t, ds.deletable)
		assert.Len(t, ds.deleteBatches, 3)
	})

	t.Run("a run with no expired notifications stops after one batch", func(t *testing.T) {
		ds := &mockDatastore{}

		require.NoError(t, NewService(ds, queue, logger).CleanupNotifications(ctx))

		assert.Len(t, ds.deleteBatches, 1)
	})

	t.Run("a run stops at the cap and leaves the remaining expired notifications for the next run", func(t *testing.T) {
		ds := &mockDatastore{deletable: endUserNotificationCleanupMaxRows * 2}

		require.NoError(t, NewService(ds, queue, logger).CleanupNotifications(ctx))

		assert.EqualValues(t, endUserNotificationCleanupMaxRows, ds.deletable)
		assert.Len(t, ds.deleteBatches, endUserNotificationCleanupMaxRows/endUserNotificationCleanupBatchSize)
	})
}
