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

func TestReconcileAppleProfilesBatchedCursorAdvance(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	newMockDS := func(snapshotHosts []*fleet.AppleHostReconcileInfo, pageFull bool) (*mock.Store, *string) {
		return newAppleReconcileBatchedMockDS(t, snapshotHosts, nil, nil, pageFull)
	}

	t.Run("full raw page that deduped below batch size still advances the cursor", func(t *testing.T) {
		// One host survives dedupe out of a raw page that hit the SQL limit
		// (duplicate-UUID rows collapsed). The cursor must advance to that
		// host's UUID; wrapping to "" here is the bug that permanently starves
		// every host later in the UUID ordering.
		hosts := []*fleet.AppleHostReconcileInfo{{HostID: 2, UUID: "uuid-dup", Platform: "darwin"}}
		ds, savedCursor := newMockDS(hosts, true)

		require.NoError(t, ReconcileAppleProfilesBatched(ctx, ds, nil, nil, logger, 0, false))
		require.True(t, ds.SetMDMAppleReconcileCursorFuncInvoked)
		require.Equal(t, "uuid-dup", *savedCursor)
	})

	t.Run("short raw page wraps the cursor", func(t *testing.T) {
		hosts := []*fleet.AppleHostReconcileInfo{{HostID: 2, UUID: "uuid-last", Platform: "darwin"}}
		ds, _ := newMockDS(hosts, false)

		require.NoError(t, ReconcileAppleProfilesBatched(ctx, ds, nil, nil, logger, 0, false))
		// cursor was already "" and the page was short, so it stays "" (no write).
		require.False(t, ds.SetMDMAppleReconcileCursorFuncInvoked)
	})
}

func TestReconcileAppleProfilesBatchedOptIns(t *testing.T) {
	ctx := t.Context()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	hostA := &fleet.AppleHostReconcileInfo{HostID: 1, UUID: "uuid-A", TeamID: new(uint(1)), Platform: "darwin"}
	hostB := &fleet.AppleHostReconcileInfo{HostID: 2, UUID: "uuid-B", TeamID: new(uint(1)), Platform: "darwin"}
	forced := &fleet.AppleProfileForReconcile{
		ProfileUUID: "aForced", ProfileIdentifier: "com.example.forced", TeamID: 1,
		Checksum: []byte("ffff"), Scope: fleet.PayloadScopeSystem,
	}
	selfService := &fleet.AppleProfileForReconcile{
		ProfileUUID: "aSelfService", ProfileIdentifier: "com.example.ss", TeamID: 1,
		Checksum: []byte("ssss"), Scope: fleet.PayloadScopeSystem, SelfService: true,
	}
	verifiedRow := func(host *fleet.AppleHostReconcileInfo, p *fleet.AppleProfileForReconcile) *fleet.MDMAppleProfilePayload {
		return &fleet.MDMAppleProfilePayload{
			ProfileUUID: p.ProfileUUID, ProfileIdentifier: p.ProfileIdentifier, HostUUID: host.UUID,
			Checksum: p.Checksum, Scope: p.Scope,
			OperationType: fleet.MDMOperationTypeInstall, Status: new(fleet.MDMDeliveryVerified),
		}
	}

	t.Run("loads opt-ins for every host in the batch", func(t *testing.T) {
		ds, _ := newAppleReconcileBatchedMockDS(t, []*fleet.AppleHostReconcileInfo{hostA, hostB}, nil, nil, false)
		var gotHostUUIDs []string
		ds.BulkGetHostMDMProfileOptInsFunc = func(ctx context.Context, hostUUIDs []string) (map[string]map[string]struct{}, error) {
			gotHostUUIDs = hostUUIDs
			return nil, nil
		}

		require.NoError(t, ReconcileAppleProfilesBatched(ctx, ds, nil, nil, logger, 0, false))
		require.ElementsMatch(t, []string{hostA.UUID, hostB.UUID}, gotHostUUIDs)
	})

	t.Run("nothing to do -> opt-ins untouched, nothing enqueued", func(t *testing.T) {
		ds, _ := newAppleReconcileBatchedMockDS(t,
			[]*fleet.AppleHostReconcileInfo{hostA}, []*fleet.AppleProfileForReconcile{forced},
			map[string][]*fleet.MDMAppleProfilePayload{hostA.UUID: {verifiedRow(hostA, forced)}}, false,
		)

		require.NoError(t, ReconcileAppleProfilesBatched(ctx, ds, nil, nil, logger, 0, false))
		require.False(t, ds.ApplyHostMDMProfileOptInChangesFuncInvoked)
		require.False(t, ds.BulkUpsertMDMAppleHostProfilesFuncInvoked)
	})

	t.Run("purge-only tick (self-service flipped to force install) applies the purge", func(t *testing.T) {
		ds, _ := newAppleReconcileBatchedMockDS(t,
			[]*fleet.AppleHostReconcileInfo{hostA}, []*fleet.AppleProfileForReconcile{forced},
			map[string][]*fleet.MDMAppleProfilePayload{hostA.UUID: {verifiedRow(hostA, forced)}}, false,
		)
		ds.BulkGetHostMDMProfileOptInsFunc = func(ctx context.Context, hostUUIDs []string) (map[string]map[string]struct{}, error) {
			return map[string]map[string]struct{}{hostA.UUID: {forced.ProfileUUID: {}}}, nil
		}
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
		ds, _ := newAppleReconcileBatchedMockDS(t,
			[]*fleet.AppleHostReconcileInfo{hostA}, []*fleet.AppleProfileForReconcile{selfService}, nil, false,
		)

		require.NoError(t, ReconcileAppleProfilesBatched(ctx, ds, nil, nil, logger, 0, false))
		require.False(t, ds.ApplyHostMDMProfileOptInChangesFuncInvoked)
		require.False(t, ds.BulkUpsertMDMAppleHostProfilesFuncInvoked)
	})

	t.Run("opt-in load error aborts the tick without advancing the cursor", func(t *testing.T) {
		ds, _ := newAppleReconcileBatchedMockDS(t, []*fleet.AppleHostReconcileInfo{hostA}, nil, nil, true)
		ds.BulkGetHostMDMProfileOptInsFunc = func(ctx context.Context, hostUUIDs []string) (map[string]map[string]struct{}, error) {
			return nil, errors.New("boom")
		}

		require.ErrorContains(t, ReconcileAppleProfilesBatched(ctx, ds, nil, nil, logger, 0, false), "boom")
		require.False(t, ds.SetMDMAppleReconcileCursorFuncInvoked)
		require.False(t, ds.ApplyHostMDMProfileOptInChangesFuncInvoked)
	})

	t.Run("opt-in apply error aborts the tick without advancing the cursor", func(t *testing.T) {
		ds, _ := newAppleReconcileBatchedMockDS(t,
			[]*fleet.AppleHostReconcileInfo{hostA}, []*fleet.AppleProfileForReconcile{forced},
			map[string][]*fleet.MDMAppleProfilePayload{hostA.UUID: {verifiedRow(hostA, forced)}}, true,
		)
		ds.BulkGetHostMDMProfileOptInsFunc = func(ctx context.Context, hostUUIDs []string) (map[string]map[string]struct{}, error) {
			return map[string]map[string]struct{}{hostA.UUID: {forced.ProfileUUID: {}}}, nil
		}
		ds.ApplyHostMDMProfileOptInChangesFunc = func(ctx context.Context, changes *fleet.MDMProfileOptInChanges) error {
			return errors.New("boom")
		}

		require.ErrorContains(t, ReconcileAppleProfilesBatched(ctx, ds, nil, nil, logger, 0, false), "boom")
		require.False(t, ds.SetMDMAppleReconcileCursorFuncInvoked)
	})
}

func newAppleReconcileBatchedMockDS(
	t *testing.T,
	snapshotHosts []*fleet.AppleHostReconcileInfo,
	profiles []*fleet.AppleProfileForReconcile,
	current map[string][]*fleet.MDMAppleProfilePayload,
	pageFull bool,
) (*mock.Store, *string) {
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
	ds.GetMDMAppleReconcileCursorFunc = func(ctx context.Context) (string, error) {
		return "", nil
	}
	var savedCursor string
	ds.SetMDMAppleReconcileCursorFunc = func(ctx context.Context, cursor string) error {
		savedCursor = cursor
		return nil
	}
	ds.GetAppleProfileReconcileSnapshotFunc = func(ctx context.Context, afterHostUUID string, batchSize int) ([]*fleet.AppleHostReconcileInfo, []*fleet.AppleProfileForReconcile, map[uint]map[uint]struct{}, map[string][]*fleet.MDMAppleProfilePayload, bool, error) {
		return snapshotHosts, profiles, nil, current, pageFull, nil
	}
	ds.BulkGetHostMDMProfileOptInsFunc = func(ctx context.Context, hostUUIDs []string) (map[string]map[string]struct{}, error) {
		return nil, nil
	}
	ds.ApplyHostMDMProfileOptInChangesFunc = func(ctx context.Context, changes *fleet.MDMProfileOptInChanges) error {
		return nil
	}
	return ds, &savedCursor
}
