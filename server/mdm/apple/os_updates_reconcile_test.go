package apple_mdm

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/dev_mode"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/stretchr/testify/require"
)

// TestComputeOSUpdatesTarget covers the in-memory decision logic that maps each
// host to its target OS version / deadline / resend flag. It exercises the
// branches independently of the datastore and GDMF: platform gate, removal when
// a host's team no longer has "latest", version selection, deadline derivation,
// and the unchanged-target short circuit.
func TestComputeOSUpdatesTarget(t *testing.T) {
	logger := slog.New(slog.DiscardHandler)
	ctx := t.Context()

	const (
		macDevice = "Mac14,2"
		iosDevice = "iPhone15,2"
	)

	mustParseDate := func(s string) time.Time {
		d, err := time.Parse("2006-01-02", s)
		if err != nil {
			t.Fatalf("failed to parse date %q: %v", s, err)
		}
		return d
	}

	// Two macOS assets for the same device, higher version listed first so the
	// test proves the reconciler picks the max version, not the last one seen.
	updateAssets := map[string][]fleet.AppleSoftwareUpdateAsset{
		"macos": {
			{ProductVersion: "15.1", PostingDate: mustParseDate("2024-10-28"), SupportedDevices: []string{macDevice}},
			{ProductVersion: "14.6.1", PostingDate: mustParseDate("2024-08-07"), SupportedDevices: []string{macDevice}},
		},
		"ios": {
			{ProductVersion: "18.1", PostingDate: mustParseDate("2024-10-28"), SupportedDevices: []string{iosDevice}},
		},
	}

	// deadline = posting date + deadlineDays. Posting dates parse as midnight UTC.
	macDeadline := time.Date(2024, 10, 30, 0, 0, 0, 0, time.UTC) // 2024-10-28 + 2d
	iosDeadline := time.Date(2024, 10, 30, 0, 0, 0, 0, time.UTC) // 2024-10-28 + 2d

	// teamsWithLatest is keyed by the three supported platforms; nil inner maps
	// mean "no team on that platform has latest configured".
	latest := func(darwin, ios, ipados map[uint]int) map[string]map[uint]int {
		return map[string]map[uint]int{"darwin": darwin, "ios": ios, "ipados": ipados}
	}
	compute := func(host *fleet.AppleSoftwareUpdateHost, teams map[string]map[uint]int) []*fleet.ComputedAppleSoftwareUpdateHost {
		return computeOSUpdatesTarget(ctx, logger, []*fleet.AppleSoftwareUpdateHost{host}, updateAssets, teams)
	}

	t.Run("macOS host in a team with latest gets highest version and posting-date deadline", func(t *testing.T) {
		host := &fleet.AppleSoftwareUpdateHost{HostUUID: "h1", Platform: "darwin", TeamID: 0, SoftwareUpdateDeviceID: macDevice}
		got := compute(host, latest(map[uint]int{0: 2}, nil, nil))
		require.Len(t, got, 1)
		require.True(t, got[0].Resend)
		require.Equal(t, "15.1", got[0].TargetOSVersion)
		require.NotNil(t, got[0].TargetDeadline)
		require.True(t, macDeadline.Equal(*got[0].TargetDeadline), "want %s got %s", macDeadline, got[0].TargetDeadline)
		require.NotNil(t, got[0].ResolvedAt)
	})

	t.Run("iPadOS host resolves against the shared ios asset set", func(t *testing.T) {
		host := &fleet.AppleSoftwareUpdateHost{HostUUID: "h2", Platform: "ipados", TeamID: 3, SoftwareUpdateDeviceID: iosDevice}
		got := compute(host, latest(nil, nil, map[uint]int{3: 2}))
		require.Len(t, got, 1)
		require.True(t, got[0].Resend)
		require.Equal(t, "18.1", got[0].TargetOSVersion)
		require.True(t, iosDeadline.Equal(*got[0].TargetDeadline))
	})

	t.Run("first_seen_at after posting date drives the deadline", func(t *testing.T) {
		firstSeen := time.Date(2024, 11, 5, 12, 0, 0, 0, time.UTC)
		assets := map[string][]fleet.AppleSoftwareUpdateAsset{
			"macos": {{ProductVersion: "15.1", PostingDate: mustParseDate("2024-10-28"), FirstSeenAt: firstSeen, SupportedDevices: []string{macDevice}}},
		}
		host := &fleet.AppleSoftwareUpdateHost{HostUUID: "h3", Platform: "darwin", TeamID: 0, SoftwareUpdateDeviceID: macDevice}
		got := computeOSUpdatesTarget(ctx, logger, []*fleet.AppleSoftwareUpdateHost{host}, assets, latest(map[uint]int{0: 2}, nil, nil))
		require.Len(t, got, 1)
		require.True(t, firstSeen.Add(2*24*time.Hour).Equal(*got[0].TargetDeadline))
	})

	t.Run("host whose team no longer has latest is cleared without resend", func(t *testing.T) {
		deadline := macDeadline
		host := &fleet.AppleSoftwareUpdateHost{
			HostUUID: "h4", Platform: "darwin", TeamID: 5, SoftwareUpdateDeviceID: macDevice,
			TargetOSVersion: "15.1", TargetDeadline: &deadline, ResolvedAt: &deadline,
		}
		// team 5 is not present in the darwin latest set.
		got := compute(host, latest(map[uint]int{0: 2}, nil, nil))
		require.Len(t, got, 1)
		require.False(t, got[0].Resend)
		require.Empty(t, got[0].TargetOSVersion)
		require.Nil(t, got[0].TargetDeadline)
		require.Nil(t, got[0].ResolvedAt)
	})

	t.Run("unchanged target and deadline are skipped", func(t *testing.T) {
		deadline := macDeadline
		host := &fleet.AppleSoftwareUpdateHost{
			HostUUID: "h5", Platform: "darwin", TeamID: 0, SoftwareUpdateDeviceID: macDevice,
			TargetOSVersion: "15.1", TargetDeadline: &deadline,
		}
		got := compute(host, latest(map[uint]int{0: 2}, nil, nil))
		require.Empty(t, got)
	})

	t.Run("unsupported platform is skipped", func(t *testing.T) {
		host := &fleet.AppleSoftwareUpdateHost{HostUUID: "h6", Platform: "tvos", TeamID: 0, SoftwareUpdateDeviceID: macDevice}
		got := compute(host, latest(map[uint]int{0: 2}, nil, nil))
		require.Empty(t, got)
	})

	t.Run("no asset matches the host device id is skipped", func(t *testing.T) {
		host := &fleet.AppleSoftwareUpdateHost{HostUUID: "h7", Platform: "darwin", TeamID: 0, SoftwareUpdateDeviceID: "Mac99,9"}
		got := compute(host, latest(map[uint]int{0: 2}, nil, nil))
		require.Empty(t, got)
	})

	t.Run("no assets for the platform is skipped", func(t *testing.T) {
		host := &fleet.AppleSoftwareUpdateHost{HostUUID: "h8", Platform: "darwin", TeamID: 0, SoftwareUpdateDeviceID: macDevice}
		got := computeOSUpdatesTarget(ctx, logger, []*fleet.AppleSoftwareUpdateHost{host}, map[string][]fleet.AppleSoftwareUpdateAsset{}, latest(map[uint]int{0: 2}, nil, nil))
		require.Empty(t, got)
	})

	t.Run("duplicate host UUIDs in a batch are collapsed to the first row", func(t *testing.T) {
		deadline := macDeadline
		inTeamWithLatest := &fleet.AppleSoftwareUpdateHost{
			HostUUID: "dup", Platform: "darwin", TeamID: 0, SoftwareUpdateDeviceID: macDevice,
		}
		// same UUID, but this row's team has no "latest" set, so on its own it would clear the
		// target computed for the row above
		inTeamWithoutLatest := &fleet.AppleSoftwareUpdateHost{
			HostUUID: "dup", Platform: "darwin", TeamID: 5, SoftwareUpdateDeviceID: macDevice,
			TargetOSVersion: "15.1", TargetDeadline: &deadline, ResolvedAt: &deadline,
		}

		got := computeOSUpdatesTarget(ctx, logger,
			[]*fleet.AppleSoftwareUpdateHost{inTeamWithLatest, inTeamWithoutLatest},
			updateAssets, latest(map[uint]int{0: 2}, nil, nil))
		require.Len(t, got, 1)
		require.Equal(t, "dup", got[0].HostUUID)
		require.True(t, got[0].Resend)
		require.Equal(t, "15.1", got[0].TargetOSVersion)
		require.True(t, macDeadline.Equal(*got[0].TargetDeadline))

		// order decides the winner, and either way only one row per UUID comes out
		got = computeOSUpdatesTarget(ctx, logger,
			[]*fleet.AppleSoftwareUpdateHost{inTeamWithoutLatest, inTeamWithLatest},
			updateAssets, latest(map[uint]int{0: 2}, nil, nil))
		require.Len(t, got, 1)
		require.Equal(t, "dup", got[0].HostUUID)
		require.False(t, got[0].Resend)
		require.Empty(t, got[0].TargetOSVersion)
	})

	t.Run("duplicate host UUIDs are collapsed even when the first row produces nothing", func(t *testing.T) {
		unsupported := &fleet.AppleSoftwareUpdateHost{HostUUID: "dup2", Platform: "tvos", TeamID: 0, SoftwareUpdateDeviceID: macDevice}
		supported := &fleet.AppleSoftwareUpdateHost{HostUUID: "dup2", Platform: "darwin", TeamID: 0, SoftwareUpdateDeviceID: macDevice}

		got := computeOSUpdatesTarget(ctx, logger,
			[]*fleet.AppleSoftwareUpdateHost{unsupported, supported},
			updateAssets, latest(map[uint]int{0: 2}, nil, nil))
		require.Empty(t, got)
	})
}

// TestHandleAppleMDMOSUpdatesAssetRefresh covers the asset-cache refresh at the top of the OS
// updates cron: when the cache is stale it fetches from GDMF, caches the fetched assets, and
// prunes the cached assets Apple no longer reports. The pruning must never run on a fetch we
// didn't get, otherwise we'd delete assets based on an incomplete view of what Apple publishes.
func TestHandleAppleMDMOSUpdatesAssetRefresh(t *testing.T) {
	logger := slog.New(slog.DiscardHandler)
	ctx := t.Context()

	gdmfFixture, err := os.ReadFile("./gdmf/testdata/gdmf.json")
	require.NoError(t, err)

	// newDS returns a store with the reconcile part of the cron stubbed out to a no-op, so only
	// the asset refresh is under test. lastUpdatedAt nil means the cache is stale.
	newDS := func(lastUpdatedAt *time.Time) *mock.Store {
		ds := new(mock.Store)
		ds.GetLastAppleOSUpdatesUpdateFunc = func(ctx context.Context) (*time.Time, error) {
			return lastUpdatedAt, nil
		}
		ds.UpsertAppleOSUpdatesFunc = func(ctx context.Context, updates map[string][]fleet.OSUpdateAsset) error {
			return nil
		}
		ds.DeleteStaleAppleOSUpdatesFunc = func(ctx context.Context, updates map[string][]fleet.OSUpdateAsset) (int64, error) {
			return 0, nil
		}
		ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
			return &fleet.AppConfig{}, nil
		}
		ds.ListTeamsFunc = func(ctx context.Context, filter fleet.TeamFilter, opt fleet.ListOptions) ([]*fleet.Team, error) {
			return nil, nil
		}
		ds.ListAppleOSUpdateAssetsFunc = func(ctx context.Context) (map[string][]fleet.AppleSoftwareUpdateAsset, error) {
			return nil, nil
		}
		ds.ListAppleOSUpdateHostsForReconcileFunc = func(ctx context.Context, cursor string, batchSize int, teamsWithLatest map[string]map[uint]int) ([]*fleet.AppleSoftwareUpdateHost, error) {
			return nil, nil
		}
		return ds
	}

	serveGDMF := func(t *testing.T, body []byte) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write(body); err != nil {
				t.Errorf("writing response: %v", err)
			}
		}))
		t.Cleanup(srv.Close)
		dev_mode.SetOverride("FLEET_DEV_GDMF_URL", srv.URL, t)
	}

	t.Run("stale cache caches the fetched assets and prunes with the same set", func(t *testing.T) {
		serveGDMF(t, gdmfFixture)

		ds := newDS(nil)
		var upserted, pruned map[string][]fleet.OSUpdateAsset
		ds.UpsertAppleOSUpdatesFunc = func(ctx context.Context, updates map[string][]fleet.OSUpdateAsset) error {
			upserted = updates
			return nil
		}
		ds.DeleteStaleAppleOSUpdatesFunc = func(ctx context.Context, updates map[string][]fleet.OSUpdateAsset) (int64, error) {
			pruned = updates
			return 3, nil
		}

		require.NoError(t, HandleAppleMDMOSUpdates(ctx, ds, logger))
		require.True(t, ds.UpsertAppleOSUpdatesFuncInvoked)
		require.True(t, ds.DeleteStaleAppleOSUpdatesFuncInvoked)
		require.NotEmpty(t, upserted["macos"])
		require.NotEmpty(t, upserted["ios"])
		// pruning against exactly what we cached is what makes the delete safe
		require.Equal(t, upserted, pruned)
	})

	t.Run("fresh cache neither caches nor prunes", func(t *testing.T) {
		// no GDMF override: a fetch would fail the test by trying to reach Apple
		lastUpdatedAt := time.Now().Add(-time.Hour)
		ds := newDS(&lastUpdatedAt)

		require.NoError(t, HandleAppleMDMOSUpdates(ctx, ds, logger))
		require.False(t, ds.UpsertAppleOSUpdatesFuncInvoked)
		require.False(t, ds.DeleteStaleAppleOSUpdatesFuncInvoked)
		require.True(t, ds.ListAppleOSUpdateAssetsFuncInvoked, "the reconcile still runs on the cached assets")
	})

	t.Run("failed fetch neither caches nor prunes", func(t *testing.T) {
		serveGDMF(t, []byte("not valid json"))

		ds := newDS(nil)

		require.NoError(t, HandleAppleMDMOSUpdates(ctx, ds, logger))
		require.False(t, ds.UpsertAppleOSUpdatesFuncInvoked)
		require.False(t, ds.DeleteStaleAppleOSUpdatesFuncInvoked, "pruning on a failed fetch would delete assets Apple still publishes")
		require.True(t, ds.ListAppleOSUpdateAssetsFuncInvoked, "the reconcile still runs on the cached assets")
	})

	t.Run("failed upsert does not prune", func(t *testing.T) {
		serveGDMF(t, gdmfFixture)

		ds := newDS(nil)
		ds.UpsertAppleOSUpdatesFunc = func(ctx context.Context, updates map[string][]fleet.OSUpdateAsset) error {
			return errors.New("upsert failed")
		}

		require.NoError(t, HandleAppleMDMOSUpdates(ctx, ds, logger))
		require.True(t, ds.UpsertAppleOSUpdatesFuncInvoked)
		require.False(t, ds.DeleteStaleAppleOSUpdatesFuncInvoked, "the cache is out of sync with the fetch, so pruning is unsafe")
	})

	t.Run("failed prune does not fail the cron", func(t *testing.T) {
		serveGDMF(t, gdmfFixture)

		ds := newDS(nil)
		ds.DeleteStaleAppleOSUpdatesFunc = func(ctx context.Context, updates map[string][]fleet.OSUpdateAsset) (int64, error) {
			return 0, errors.New("delete failed")
		}

		require.NoError(t, HandleAppleMDMOSUpdates(ctx, ds, logger))
		require.True(t, ds.DeleteStaleAppleOSUpdatesFuncInvoked)
		require.True(t, ds.ListAppleOSUpdateAssetsFuncInvoked, "the reconcile still runs on the cached assets")
	})
}
