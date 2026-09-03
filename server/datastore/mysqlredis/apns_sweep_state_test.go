package mysqlredis

import (
	"log/slog"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/server/datastore/redis/redistest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	redigo "github.com/gomodule/redigo/redis"
	"github.com/stretchr/testify/require"
)

func TestMDMAppleAPNsSweepState(t *testing.T) {
	pool := redistest.SetupRedis(t, "apns_sweep_state", false, false, false)
	var logBuf strings.Builder
	ds := New(&mock.Store{}, pool, WithLogger(slog.New(slog.NewTextHandler(&logBuf, nil))))
	ctx := t.Context()

	// the state key is fixed (not test-prefixed), so clean it up ourselves.
	cleanup := func() {
		conn := pool.Get()
		defer conn.Close()
		_, err := conn.Do("DEL", apnsSweepStateKey)
		require.NoError(t, err)
	}
	cleanup()
	t.Cleanup(cleanup)

	state, err := ds.GetMDMAppleAPNsSweepState(ctx)
	require.NoError(t, err)
	require.Nil(t, state, "unset key reads as no pass in progress")

	want := &fleet.MDMAppleAPNsSweepState{Cursor: "enrollment-uuid-42", BatchSize: 137}
	require.NoError(t, ds.SetMDMAppleAPNsSweepState(ctx, want))
	state, err = ds.GetMDMAppleAPNsSweepState(ctx)
	require.NoError(t, err)
	require.Equal(t, want, state)

	require.NoError(t, ds.SetMDMAppleAPNsSweepState(ctx, nil))
	state, err = ds.GetMDMAppleAPNsSweepState(ctx)
	require.NoError(t, err)
	require.Nil(t, state, "nil set resets the state")

	// a poisoned key self-heals to a fresh pass instead of wedging the cron,
	// and the bad key is dropped rather than left to linger.
	conn := pool.Get()
	_, err = conn.Do("SET", apnsSweepStateKey, "{not json")
	require.NoError(t, err)
	conn.Close()
	state, err = ds.GetMDMAppleAPNsSweepState(ctx)
	require.NoError(t, err)
	require.Nil(t, state)

	conn = pool.Get()
	exists, err := redigo.Int(conn.Do("EXISTS", apnsSweepStateKey))
	conn.Close()
	require.NoError(t, err)
	require.Zero(t, exists, "poisoned key must be deleted on read")
	require.Contains(t, logBuf.String(), "level=WARN", "self-heal must be visible in logs")
	require.Contains(t, logBuf.String(), apnsSweepStateKey)
}
