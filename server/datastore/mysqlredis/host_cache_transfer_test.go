package mysqlredis

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/mysqltest"
	"github.com/fleetdm/fleet/v4/server/datastore/redis"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// TestHostCacheTransferEscrowFlag exercises the in-place team patch against a real
// MySQL so the cached MDM.EncryptionKeyAvailable flag can be compared with what an
// uncached load returns. A transfer deletes escrowed disk encryption keys for
// platforms the destination doesn't escrow for, per host, so a batch that mixes
// platforms must come out of the patch with per-host answers.
// isolatedRedis gives this test its own Redis logical database. It cannot use
// redistest: mysqltest.CreateMySQLDS marks the test parallel, and redistest's
// cleanup deletes every key under the host-cache prefix — which is the same
// prefix the rest of this package's tests are actively using.
func isolatedRedis(t *testing.T, database int) fleet.RedisPool {
	if _, ok := os.LookupEnv("REDIS_TEST"); !ok {
		t.Skip("set REDIS_TEST environment variable to run redis-based tests")
	}
	pool, err := redis.NewPool(redis.PoolConfig{
		Server:      "127.0.0.1:6379",
		Database:    database,
		ConnTimeout: 5 * time.Second,
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		conn := pool.Get()
		_, _ = conn.Do("FLUSHDB")
		conn.Close()
		pool.Close()
	})
	return pool
}

func TestHostCacheTransferEscrowFlag(t *testing.T) {
	ds := mysqltest.CreateMySQLDS(t)
	pool := isolatedRedis(t, 3)
	d := New(ds, pool, WithHostCache(3*time.Minute))
	ctx := t.Context()

	newTeam := func(name string, mac, win, linux bool) *fleet.Team {
		tm, err := ds.NewTeam(ctx, &fleet.Team{Name: name, Config: fleet.TeamConfig{
			MDM: fleet.TeamMDM{
				MacOSSettings:   fleet.MacOSSettings{EnableEscrowDiskEncryptionKey: optjson.SetBool(mac)},
				WindowsSettings: fleet.WindowsSettings{EnableDiskEncryption: optjson.SetBool(win)},
				LinuxSettings:   fleet.LinuxSettings{EnableEscrowDiskEncryptionKey: optjson.SetBool(linux)},
			},
		}})
		require.NoError(t, err)
		return tm
	}

	// Escrows for macOS only, so one transfer covers "destination escrows for this
	// host's platform" and "it doesn't" at the same time.
	macOnly := newTeam("escrow-macos-only", true, false, false)
	noEscrow := newTeam("escrow-none", false, false, false)

	type testHost struct {
		name     string
		platform string
		withKey  bool
		host     *fleet.Host
		orbitKey string
	}
	hosts := []*testHost{
		{name: "darwin with escrowed key", platform: "darwin", withKey: true},
		{name: "darwin without key", platform: "darwin", withKey: false},
		{name: "windows with escrowed key", platform: "windows", withKey: true},
		{name: "ubuntu with escrowed key", platform: "ubuntu", withKey: true},
	}

	hostIDs := make([]uint, 0, len(hosts))
	for i, th := range hosts {
		uid := uuid.NewString()
		h, err := ds.NewHost(ctx, &fleet.Host{
			OsqueryHostID:   new(uid),
			NodeKey:         new(fmt.Sprintf("nk-%d-%s", i, uid)),
			UUID:            uid,
			Hostname:        fmt.Sprintf("host-%d", i),
			HardwareSerial:  fmt.Sprintf("serial-%d", i),
			Platform:        th.platform,
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now(),
		})
		require.NoError(t, err)

		th.orbitKey = fmt.Sprintf("onk-%d-%s", i, uid)
		_, err = ds.EnrollOrbit(ctx,
			fleet.WithEnrollOrbitHostInfo(fleet.OrbitHostInfo{HardwareUUID: uid, HardwareSerial: h.HardwareSerial}),
			fleet.WithEnrollOrbitNodeKey(th.orbitKey),
		)
		require.NoError(t, err)

		if th.withKey {
			_, err = ds.SetOrUpdateHostDiskEncryptionKey(ctx, h, "encrypted-key-blob", "", new(true))
			require.NoError(t, err)
		}
		th.host = h
		hostIDs = append(hostIDs, h.ID)
	}

	// requireConsistent asserts the cached snapshot agrees with an uncached load on
	// both fields a transfer can change, and that the entry survived (was patched,
	// not dropped by the fallback).
	requireConsistent := func(t *testing.T, th *testHost, wantTeam *uint, wantKeyAvailable bool) {
		t.Helper()

		cached, lookup := d.hostCacheGetByOrbitNodeKey(ctx, th.orbitKey)
		require.Equal(t, hostCacheLookupHit, lookup, "%s: entry must survive the transfer", th.name)
		require.Equal(t, wantKeyAvailable, cached.MDM.EncryptionKeyAvailable, "%s: cached encryption_key_available", th.name)
		require.Equal(t, wantTeam, cached.TeamID, "%s: cached team_id", th.name)

		fresh, err := ds.LoadHostByOrbitNodeKey(ctx, th.orbitKey)
		require.NoError(t, err)
		require.Equal(t, fresh.MDM.EncryptionKeyAvailable, cached.MDM.EncryptionKeyAvailable,
			"%s: cached flag must match an uncached load", th.name)
		require.Equal(t, fresh.TeamID, cached.TeamID, "%s: cached team_id must match an uncached load", th.name)

		key, err := ds.GetHostDiskEncryptionKey(ctx, th.host.ID)
		if wantKeyAvailable {
			require.NoError(t, err, "%s: escrowed key must still exist", th.name)
			require.NotNil(t, key)
		} else {
			require.Error(t, err, "%s: escrowed key must be gone", th.name)
			require.True(t, fleet.IsNotFound(err), "%s: expected NotFound, got %v", th.name, err)
		}
	}

	primeCache := func(t *testing.T) {
		t.Helper()
		for _, th := range hosts {
			_, err := d.LoadHostByOrbitNodeKey(ctx, th.orbitKey)
			require.NoError(t, err)
			_, lookup := d.hostCacheGetByOrbitNodeKey(ctx, th.orbitKey)
			require.Equal(t, hostCacheLookupHit, lookup, "%s: failed to prime", th.name)
		}
	}

	t.Run("destination escrows for some platforms but not others", func(t *testing.T) {
		primeCache(t)
		require.NoError(t, d.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&macOnly.ID, hostIDs)))

		// macOS is escrowed by the destination, so the key survives and the flag
		// stays true; Windows and Linux are not, so their keys were deleted and the
		// patched flag must follow. The host that never had a key stays false.
		requireConsistent(t, hosts[0], &macOnly.ID, true)
		requireConsistent(t, hosts[1], &macOnly.ID, false)
		requireConsistent(t, hosts[2], &macOnly.ID, false)
		requireConsistent(t, hosts[3], &macOnly.ID, false)
	})

	t.Run("destination escrows for no platform", func(t *testing.T) {
		// Re-escrow the macOS key so this transfer has something to drop.
		_, err := ds.SetOrUpdateHostDiskEncryptionKey(ctx, hosts[0].host, "encrypted-key-blob", "", new(true))
		require.NoError(t, err)
		primeCache(t)
		reprimed, lookup := d.hostCacheGetByOrbitNodeKey(ctx, hosts[0].orbitKey)
		require.Equal(t, hostCacheLookupHit, lookup)
		require.True(t, reprimed.MDM.EncryptionKeyAvailable, "precondition: macOS key must be escrowed again")

		require.NoError(t, d.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&noEscrow.ID, hostIDs)))

		for _, th := range hosts {
			requireConsistent(t, th, &noEscrow.ID, false)
		}
	})

	t.Run("transfer to No team", func(t *testing.T) {
		primeCache(t)
		require.NoError(t, d.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(nil, hostIDs)))

		for _, th := range hosts {
			requireConsistent(t, th, nil, false)
		}
	})
}
