package async

import (
	"context"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/stretchr/testify/require"
)

func TestRecordHostLastSeen(t *testing.T) {
	ctx := t.Context()
	ds := new(mock.Store)

	var calledWithHostIDs []uint
	ds.MarkHostsSeenFunc = func(ctx context.Context, hostIDs []uint, ts time.Time) error {
		calledWithHostIDs = append(calledWithHostIDs, hostIDs...)
		return nil
	}

	task := NewTask(ds, false)
	require.NoError(t, task.RecordHostLastSeen(ctx, 1))
	require.NoError(t, task.RecordHostLastSeen(ctx, 2))
	require.NoError(t, task.RecordHostLastSeen(ctx, 2))
	require.NoError(t, task.RecordHostLastSeen(ctx, 3))
	require.False(t, ds.MarkHostsSeenFuncInvoked)

	require.NoError(t, task.FlushHostsLastSeen(ctx, time.Now()))
	require.True(t, ds.MarkHostsSeenFuncInvoked)
	require.ElementsMatch(t, []uint{1, 2, 3}, calledWithHostIDs)

	calledWithHostIDs = nil
	require.NoError(t, task.FlushHostsLastSeen(ctx, time.Now()))
	require.Empty(t, calledWithHostIDs)
}
