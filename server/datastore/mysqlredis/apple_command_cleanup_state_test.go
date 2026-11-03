package mysqlredis

import (
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/datastore/redis/redistest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	redigo "github.com/gomodule/redigo/redis"
	"github.com/stretchr/testify/require"
)

func TestMDMAppleCommandCleanupState(t *testing.T) {
	pool := redistest.SetupRedis(t, "command_cleanup_state", false, false, false)
	var logBuf strings.Builder
	ds := New(&mock.Store{}, pool, WithLogger(slog.New(slog.NewTextHandler(&logBuf, nil))))
	ctx := t.Context()

	// the state key is fixed (not test-prefixed), so clean it up ourselves.
	cleanup := func() {
		conn := pool.Get()
		defer conn.Close()
		_, err := conn.Do("DEL", appleCommandCleanupStateKey)
		require.NoError(t, err)
	}
	cleanup()
	t.Cleanup(cleanup)

	state, err := ds.GetMDMAppleCommandCleanupState(ctx)
	require.NoError(t, err)
	require.Nil(t, state, "unset key reads as no cursors stored")

	// UTC so the round trip through JSON compares equal (no monotonic clock,
	// no location pointer).
	at := time.Date(2026, 9, 21, 10, 30, 0, 0, time.UTC)
	want := &fleet.MDMAppleCommandCleanupState{
		Retention: map[string]fleet.MDMAppleCommandCleanupCursor{
			"short:Acknowledged": {UpdatedAt: at, ID: "enrollment-1", CommandUUID: "REFETCH-DEVICE-1"},
			"standard:Error":     {UpdatedAt: at.Add(-time.Hour), ID: "enrollment-2", CommandUUID: "cmd-2"},
		},
		Orphan: fleet.MDMAppleCommandOrphanCursor{CreatedAt: at.Add(-24 * time.Hour), CommandUUID: "cmd-0"},
	}
	require.NoError(t, ds.SetMDMAppleCommandCleanupState(ctx, want))
	state, err = ds.GetMDMAppleCommandCleanupState(ctx)
	require.NoError(t, err)
	require.Equal(t, want, state)

	// partial states round-trip too: the map's nil-ness is preserved either
	// way, so a caller that emptied the map does not get a nil map back.
	for _, partial := range []*fleet.MDMAppleCommandCleanupState{
		{Orphan: want.Orphan},
		{Retention: map[string]fleet.MDMAppleCommandCleanupCursor{}},
	} {
		require.NoError(t, ds.SetMDMAppleCommandCleanupState(ctx, partial))
		state, err = ds.GetMDMAppleCommandCleanupState(ctx)
		require.NoError(t, err)
		require.Equal(t, partial, state)
	}

	require.NoError(t, ds.SetMDMAppleCommandCleanupState(ctx, nil))
	state, err = ds.GetMDMAppleCommandCleanupState(ctx)
	require.NoError(t, err)
	require.Nil(t, state, "nil set resets the state")

	// a poisoned key self-heals to a fresh start instead of wedging the cron,
	// and the bad key is dropped rather than left to linger.
	conn := pool.Get()
	_, err = conn.Do("SET", appleCommandCleanupStateKey, "{not json")
	require.NoError(t, err)
	conn.Close()
	state, err = ds.GetMDMAppleCommandCleanupState(ctx)
	require.NoError(t, err)
	require.Nil(t, state)

	conn = pool.Get()
	exists, err := redigo.Int(conn.Do("EXISTS", appleCommandCleanupStateKey))
	conn.Close()
	require.NoError(t, err)
	require.Zero(t, exists, "poisoned key must be deleted on read")
	require.Contains(t, logBuf.String(), "level=WARN", "self-heal must be visible in logs")
	require.Contains(t, logBuf.String(), appleCommandCleanupStateKey)
}
