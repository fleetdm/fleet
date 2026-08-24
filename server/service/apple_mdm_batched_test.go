package service

import (
	"context"
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

	testCert, _, err := apple_mdm.NewSCEPCACertKey()
	require.NoError(t, err)
	testCertPEM := tokenpki.PEMCertificate(testCert.Raw)

	newMockDS := func(snapshotHosts []*fleet.AppleHostReconcileInfo, pageFull bool) (*mock.Store, *string) {
		ds := new(mock.Store)
		ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
			return &fleet.AppConfig{MDM: fleet.MDM{EnabledAndConfigured: true}}, nil
		}
		ds.GetAllMDMConfigAssetsByNameFunc = func(ctx context.Context, assetNames []fleet.MDMAssetName, _ sqlx.QueryerContext) (map[fleet.MDMAssetName]fleet.MDMConfigAsset, error) {
			return map[fleet.MDMAssetName]fleet.MDMConfigAsset{
				fleet.MDMAssetCACert: {Name: fleet.MDMAssetCACert, Value: testCertPEM},
			}, nil
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
			return snapshotHosts, nil, nil, nil, pageFull, nil
		}
		return ds, &savedCursor
	}

	t.Run("full raw page that deduped below batch size still advances the cursor", func(t *testing.T) {
		// One host survives dedupe out of a raw page that hit the SQL limit
		// (duplicate-UUID rows collapsed). The cursor must advance to that
		// host's UUID; wrapping to "" here is the bug that permanently starves
		// every host later in the UUID ordering.
		hosts := []*fleet.AppleHostReconcileInfo{{HostID: 2, UUID: "uuid-dup", Platform: "darwin"}}
		ds, savedCursor := newMockDS(hosts, true)

		require.NoError(t, ReconcileAppleProfilesBatched(ctx, ds, nil, nil, logger, 0))
		require.True(t, ds.SetMDMAppleReconcileCursorFuncInvoked)
		require.Equal(t, "uuid-dup", *savedCursor)
	})

	t.Run("short raw page wraps the cursor", func(t *testing.T) {
		hosts := []*fleet.AppleHostReconcileInfo{{HostID: 2, UUID: "uuid-last", Platform: "darwin"}}
		ds, _ := newMockDS(hosts, false)

		require.NoError(t, ReconcileAppleProfilesBatched(ctx, ds, nil, nil, logger, 0))
		// cursor was already "" and the page was short, so it stays "" (no write).
		require.False(t, ds.SetMDMAppleReconcileCursorFuncInvoked)
	})
}
