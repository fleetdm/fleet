package service

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/fleetdm/fleet/v4/server/mdm/nanodep/tokenpki"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

// appleReconcileMocks is the shared mock setup for the batched Apple profile reconciler: everything the tick does before the
// drain loop (app config, SCEP asset, ensureFleetProfiles) stubbed out so the tests can focus on the loop itself.
func appleReconcileMocks(t *testing.T) *mock.Store {
	t.Helper()

	testCert, _, err := apple_mdm.NewSCEPCACertKey()
	require.NoError(t, err)
	testCertPEM := tokenpki.PEMCertificate(testCert.Raw)

	ds := new(mock.Store)
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{MDM: fleet.MDM{EnabledAndConfigured: true}}, nil
	}
	ds.GetAllMDMConfigAssetsByNameFunc = func(ctx context.Context, assetNames []fleet.MDMAssetName, _ sqlx.QueryerContext) (map[fleet.MDMAssetName]fleet.MDMConfigAsset, error) {
		return map[fleet.MDMAssetName]fleet.MDMConfigAsset{
			fleet.MDMAssetCACert: {Name: fleet.MDMAssetCACert, Value: testCertPEM},
		}, nil
	}
	ds.GetMDMAppleConfigProfileByTeamAndIdentifierFunc = func(ctx context.Context, teamID *uint, identifier string) (*fleet.MDMAppleConfigProfile, error) {
		return nil, newNotFoundError()
	}
	ds.AggregateEnrollSecretPerTeamFunc = func(ctx context.Context) ([]*fleet.EnrollSecret, error) {
		return nil, nil
	}
	ds.BulkUpsertMDMAppleConfigProfilesFunc = func(ctx context.Context, payload []*fleet.MDMAppleConfigProfile) error {
		return nil
	}
	ds.ApplyHostMDMProfileOptInChangesFunc = func(ctx context.Context, changes *fleet.MDMProfileOptInChanges) error {
		return nil
	}
	return ds
}

// pageAppleHosts wires GetAppleProfileReconcileSnapshot to page through allHosts by the host_uuid cursor the way the real
// query does (uuid > afterHostUUID, ascending, LIMIT batchSize), and counts the windows the drain loop pulls. No profiles are
// returned, so every window computes zero work — which is exactly the idle/sparse case the drain loop exists to speed up.
func pageAppleHosts(ds *mock.Store, allHosts []*fleet.AppleHostReconcileInfo, windows *int) {
	ds.GetAppleProfileReconcileSnapshotFunc = func(ctx context.Context, afterHostUUID string, batchSize int) ([]*fleet.AppleHostReconcileInfo, []*fleet.AppleProfileForReconcile, map[uint]map[uint]struct{}, map[string][]*fleet.MDMAppleProfilePayload, map[string]map[string]struct{}, bool, error) {
		*windows++
		var page []*fleet.AppleHostReconcileInfo
		for _, h := range allHosts {
			if h.UUID > afterHostUUID {
				page = append(page, h)
			}
			if len(page) == batchSize {
				break
			}
		}
		return page, nil, nil, nil, nil, len(page) == batchSize, nil
	}
}

func appleHostsNamed(uuids ...string) []*fleet.AppleHostReconcileInfo {
	hosts := make([]*fleet.AppleHostReconcileInfo, 0, len(uuids))
	for i, u := range uuids {
		hosts = append(hosts, &fleet.AppleHostReconcileInfo{HostID: uint(i + 1), UUID: u, Platform: "darwin"}) //nolint:gosec // test data
	}
	return hosts
}

func TestReconcileAppleProfilesBatchedCursorAdvance(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	newMockDS := func(t *testing.T, snapshotHosts []*fleet.AppleHostReconcileInfo, pageFull bool) (*mock.Store, *string) {
		ds := appleReconcileMocks(t)
		ds.GetMDMAppleReconcileCursorFunc = func(ctx context.Context) (string, error) {
			return "", nil
		}
		var savedCursor string
		ds.SetMDMAppleReconcileCursorFunc = func(ctx context.Context, cursor string) error {
			savedCursor = cursor
			return nil
		}
		// Returns the same page once, then nothing, so the drain loop terminates on the second window rather than spinning
		// against a snapshot that ignores the cursor.
		served := false
		ds.GetAppleProfileReconcileSnapshotFunc = func(ctx context.Context, afterHostUUID string, batchSize int) ([]*fleet.AppleHostReconcileInfo, []*fleet.AppleProfileForReconcile, map[uint]map[uint]struct{}, map[string][]*fleet.MDMAppleProfilePayload, map[string]map[string]struct{}, bool, error) {
			if served {
				return nil, nil, nil, nil, nil, false, nil
			}
			served = true
			return snapshotHosts, nil, nil, nil, nil, pageFull, nil
		}
		return ds, &savedCursor
	}

	t.Run("full raw page that deduped below batch size still advances the cursor", func(t *testing.T) {
		// One host survives dedupe out of a raw page that hit the SQL limit
		// (duplicate-UUID rows collapsed). The window must be treated as full;
		// deciding from len(hosts) is the bug that permanently starves every
		// host later in the UUID ordering.
		hosts := appleHostsNamed("uuid-dup")
		ds, _ := newMockDS(t, hosts, true)

		var windows int
		inner := ds.GetAppleProfileReconcileSnapshotFunc
		ds.GetAppleProfileReconcileSnapshotFunc = func(ctx context.Context, afterHostUUID string, batchSize int) ([]*fleet.AppleHostReconcileInfo, []*fleet.AppleProfileForReconcile, map[uint]map[uint]struct{}, map[string][]*fleet.MDMAppleProfilePayload, map[string]map[string]struct{}, bool, error) {
			windows++
			if windows > 1 {
				// The loop only gets here if the full page was treated as full.
				require.Equal(t, "uuid-dup", afterHostUUID)
			}
			return inner(ctx, afterHostUUID, batchSize)
		}

		require.NoError(t, ReconcileAppleProfilesBatched(ctx, ds, nil, nil, logger, 0, false))
		require.Equal(t, 2, windows, "a full page must be followed by another window in the same tick")
	})

	t.Run("short raw page wraps the cursor", func(t *testing.T) {
		hosts := appleHostsNamed("uuid-last")
		ds, _ := newMockDS(t, hosts, false)

		require.NoError(t, ReconcileAppleProfilesBatched(ctx, ds, nil, nil, logger, 0, false))
		// cursor was already "" and the page was short, so it stays "" (no write).
		require.False(t, ds.SetMDMAppleReconcileCursorFuncInvoked)
	})
}

func TestReconcileAppleProfilesBatchedDrainLoop(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	// Restore the package-level knobs after each subtest mutates them.
	savedBatch, savedCap, savedBudget := reconcileAppleProfilesBatchSize, reconcileAppleProfilesDeliveryCap, reconcileAppleProfilesScanBudget
	t.Cleanup(func() {
		reconcileAppleProfilesBatchSize = savedBatch
		reconcileAppleProfilesDeliveryCap = savedCap
		reconcileAppleProfilesScanBudget = savedBudget
	})

	newPagingDS := func(t *testing.T, entryCursor string, allHosts []*fleet.AppleHostReconcileInfo, windows *int) (*mock.Store, *string, *bool) {
		ds := appleReconcileMocks(t)
		ds.GetMDMAppleReconcileCursorFunc = func(ctx context.Context) (string, error) {
			return entryCursor, nil
		}
		var savedCursor string
		saved := false
		ds.SetMDMAppleReconcileCursorFunc = func(ctx context.Context, cursor string) error {
			savedCursor = cursor
			saved = true
			return nil
		}
		pageAppleHosts(ds, allHosts, windows)
		return ds, &savedCursor, &saved
	}

	t.Run("idle fleet larger than one window drains fully in one tick and wraps", func(t *testing.T) {
		reconcileAppleProfilesBatchSize = 2
		reconcileAppleProfilesDeliveryCap = 2000
		reconcileAppleProfilesScanBudget = savedBudget

		var windows int
		// Start mid-fleet so the wrap back to "" is an observable write.
		ds, savedCursor, saved := newPagingDS(t, "h02", appleHostsNamed("h01", "h02", "h03", "h04", "h05"), &windows)

		require.NoError(t, ReconcileAppleProfilesBatched(ctx, ds, nil, nil, logger, 0, false))

		// h03+h04 (full), h05 (short) => 2 windows, all inside a single tick. Before the drain loop this took one tick each.
		require.Equal(t, 2, windows)
		require.True(t, *saved)
		require.Empty(t, *savedCursor, "reaching the end of the host space must wrap the cursor")
	})

	t.Run("scan budget stops the drain and keeps the cursor where it got to", func(t *testing.T) {
		reconcileAppleProfilesBatchSize = 2
		reconcileAppleProfilesDeliveryCap = 2000
		reconcileAppleProfilesScanBudget = 0 // deadline already passed: stop after the first window

		var windows int
		ds, savedCursor, saved := newPagingDS(t, "", appleHostsNamed("h01", "h02", "h03", "h04", "h05"), &windows)

		require.NoError(t, ReconcileAppleProfilesBatched(ctx, ds, nil, nil, logger, 0, false))

		require.Equal(t, 1, windows, "an exhausted budget must stop the drain")
		require.True(t, *saved)
		require.Equal(t, "h02", *savedCursor, "the next tick resumes after the last scanned host")
	})

	t.Run("snapshot error mid-drain leaves the cursor untouched", func(t *testing.T) {
		reconcileAppleProfilesBatchSize = 2
		reconcileAppleProfilesDeliveryCap = 2000
		reconcileAppleProfilesScanBudget = savedBudget

		var windows int
		ds, _, saved := newPagingDS(t, "", appleHostsNamed("h01", "h02", "h03", "h04", "h05"), &windows)
		paging := ds.GetAppleProfileReconcileSnapshotFunc
		ds.GetAppleProfileReconcileSnapshotFunc = func(ctx context.Context, afterHostUUID string, batchSize int) ([]*fleet.AppleHostReconcileInfo, []*fleet.AppleProfileForReconcile, map[uint]map[uint]struct{}, map[string][]*fleet.MDMAppleProfilePayload, map[string]map[string]struct{}, bool, error) {
			if windows == 1 {
				// Fail on the second window, after the first already advanced commitCursor.
				return nil, nil, nil, nil, nil, false, errors.New("boom")
			}
			return paging(ctx, afterHostUUID, batchSize)
		}

		err := ReconcileAppleProfilesBatched(ctx, ds, nil, nil, logger, 0, false)
		require.Error(t, err)
		require.False(t, *saved, "a failed tick must not advance the cursor; the next tick re-scans from the same point")
	})
}

func TestAppleReconcileDeliveryCapHelpers(t *testing.T) {
	hosts := appleHostsNamed("h01", "h02", "h03")
	install := []*fleet.MDMAppleProfilePayload{{HostUUID: "h03"}, {HostUUID: "h01"}}
	remove := []*fleet.MDMAppleProfilePayload{{HostUUID: "h02"}}

	t.Run("work hosts come back in host order, not payload order", func(t *testing.T) {
		// The cap keeps a contiguous prefix of this slice and resumes the cursor at its last entry, so ascending-uuid order
		// is what makes the cursor arithmetic correct.
		require.Equal(t, []string{"h01", "h02", "h03"}, appleHostsWithWork(hosts, install, remove))
	})

	t.Run("hosts with no work are skipped", func(t *testing.T) {
		require.Equal(t, []string{"h01"}, appleHostsWithWork(hosts, []*fleet.MDMAppleProfilePayload{{HostUUID: "h01"}}, nil))
		require.Empty(t, appleHostsWithWork(hosts, nil, nil))
	})

	t.Run("filtering keeps only the allowed hosts, preserving order", func(t *testing.T) {
		allowed := map[string]struct{}{"h01": {}}
		got := filterApplePayloadsByHost(install, allowed)
		require.Len(t, got, 1)
		require.Equal(t, "h01", got[0].HostUUID)
		require.Empty(t, filterApplePayloadsByHost(install, map[string]struct{}{}))
	})
}

func TestReconcileAppleProfilesBatchedOptIns(t *testing.T) {
	ctx := t.Context()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	hostA := &fleet.AppleHostReconcileInfo{HostID: 1, UUID: "uuid-A", TeamID: new(uint(1)), Platform: "darwin"}
	forced := &fleet.AppleProfileForReconcile{
		ProfileUUID: "aForced", ProfileIdentifier: "com.example.forced", TeamID: 1,
		Checksum: []byte("ffff"), Scope: fleet.PayloadScopeSystem,
	}
	selfService := &fleet.AppleProfileForReconcile{
		ProfileUUID: "aSelfService", ProfileIdentifier: "com.example.ss", TeamID: 1,
		Checksum: []byte("ssss"), Scope: fleet.PayloadScopeSystem, SelfService: true,
	}
	forcedVerifiedOnA := map[string][]*fleet.MDMAppleProfilePayload{hostA.UUID: {{
		ProfileUUID: forced.ProfileUUID, ProfileIdentifier: forced.ProfileIdentifier, HostUUID: hostA.UUID,
		Checksum: forced.Checksum, Scope: forced.Scope,
		OperationType: fleet.MDMOperationTypeInstall, Status: new(fleet.MDMDeliveryVerified),
	}}}

	// A single short window holding hosts, so the drain loop runs exactly once.
	newDS := func(
		t *testing.T, hosts []*fleet.AppleHostReconcileInfo, profiles []*fleet.AppleProfileForReconcile,
		current map[string][]*fleet.MDMAppleProfilePayload, optIns map[string]map[string]struct{},
	) *mock.Store {
		ds := appleReconcileMocks(t)
		ds.GetMDMAppleReconcileCursorFunc = func(ctx context.Context) (string, error) { return "", nil }
		ds.SetMDMAppleReconcileCursorFunc = func(ctx context.Context, cursor string) error { return nil }
		ds.GetAppleProfileReconcileSnapshotFunc = func(ctx context.Context, afterHostUUID string, batchSize int) ([]*fleet.AppleHostReconcileInfo, []*fleet.AppleProfileForReconcile, map[uint]map[uint]struct{}, map[string][]*fleet.MDMAppleProfilePayload, map[string]map[string]struct{}, bool, error) {
			return hosts, profiles, nil, current, optIns, false, nil
		}
		return ds
	}
	optedInForcedOnA := map[string]map[string]struct{}{hostA.UUID: {forced.ProfileUUID: {}}}

	t.Run("nothing to do -> opt-ins untouched, nothing enqueued", func(t *testing.T) {
		ds := newDS(t, []*fleet.AppleHostReconcileInfo{hostA}, []*fleet.AppleProfileForReconcile{forced}, forcedVerifiedOnA, nil)

		require.NoError(t, ReconcileAppleProfilesBatched(ctx, ds, nil, nil, logger, 0, false))
		require.False(t, ds.ApplyHostMDMProfileOptInChangesFuncInvoked)
		require.False(t, ds.BulkUpsertMDMAppleHostProfilesFuncInvoked)
	})

	t.Run("purge-only window (self-service flipped to force install) applies the purge", func(t *testing.T) {
		ds := newDS(t, []*fleet.AppleHostReconcileInfo{hostA}, []*fleet.AppleProfileForReconcile{forced}, forcedVerifiedOnA, optedInForcedOnA)
		var applied *fleet.MDMProfileOptInChanges
		ds.ApplyHostMDMProfileOptInChangesFunc = func(ctx context.Context, changes *fleet.MDMProfileOptInChanges) error {
			applied = changes
			return nil
		}

		require.NoError(t, ReconcileAppleProfilesBatched(ctx, ds, nil, nil, logger, 0, false))
		require.NotNil(t, applied)
		require.Empty(t, applied.Add)
		require.Equal(t, []fleet.HostProfileUUID{{HostUUID: hostA.UUID, ProfileUUID: forced.ProfileUUID}}, applied.Purge)
		require.False(t, ds.BulkUpsertMDMAppleHostProfilesFuncInvoked)
	})

	t.Run("un-opted self-service profile is not installed", func(t *testing.T) {
		ds := newDS(t, []*fleet.AppleHostReconcileInfo{hostA}, []*fleet.AppleProfileForReconcile{selfService}, nil, nil)

		require.NoError(t, ReconcileAppleProfilesBatched(ctx, ds, nil, nil, logger, 0, false))
		require.False(t, ds.ApplyHostMDMProfileOptInChangesFuncInvoked)
		require.False(t, ds.BulkUpsertMDMAppleHostProfilesFuncInvoked)
	})

	t.Run("opt-in apply error aborts the tick", func(t *testing.T) {
		ds := newDS(t, []*fleet.AppleHostReconcileInfo{hostA}, []*fleet.AppleProfileForReconcile{forced}, forcedVerifiedOnA, optedInForcedOnA)
		ds.ApplyHostMDMProfileOptInChangesFunc = func(ctx context.Context, changes *fleet.MDMProfileOptInChanges) error {
			return errors.New("boom")
		}

		require.ErrorContains(t, ReconcileAppleProfilesBatched(ctx, ds, nil, nil, logger, 0, false), "boom")
	})
}
