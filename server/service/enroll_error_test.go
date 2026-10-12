package service

import (
	"context"
	"errors"
	"testing"

	"github.com/fleetdm/fleet/v4/server/contexts/logging"
	"github.com/stretchr/testify/require"
)

func TestEnrollError(t *testing.T) {
	t.Parallel()

	dbErr := errors.New("Error 1406 (22001): Data too long for column 'osquery_host_id'")

	t.Run("detail moves to the log line", func(t *testing.T) {
		t.Parallel()
		ctx := logging.NewContext(t.Context(), &logging.LoggingContext{})
		err := enrollError(ctx, dbErr, "save host in enroll agent")

		require.EqualError(t, err, "save host in enroll agent")
		require.NotContains(t, err.Error(), "osquery_host_id")
		logCtx, _ := logging.FromContext(ctx)
		require.Contains(t, logCtx.Extras, dbErr.Error())
	})

	t.Run("cancellation keeps its identity", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := context.WithCancel(logging.NewContext(t.Context(), &logging.LoggingContext{}))
		cancel()
		err := enrollError(ctx, context.Canceled, "save host in enroll agent")

		require.ErrorIs(t, err, context.Canceled)
		require.NotContains(t, err.Error(), "osquery_host_id")
	})
}
