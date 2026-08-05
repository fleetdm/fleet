package mysql

import (
	"context"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestAppleSoftwareUpdateAssets(t *testing.T) {
	ds := CreateMySQLDS(t)
	t.Run("ReplaceUpsertPrune", func(t *testing.T) {
		defer TruncateTables(t, ds)
		testReplaceAppleSoftwareUpdateAssetsUpsertPrune(t, ds)
	})
}

func testReplaceAppleSoftwareUpdateAssetsUpsertPrune(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	class := fleet.AppleSoftwareUpdateAssetClassMacOS
	posting := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)

	require.NoError(t, ds.ReplaceAppleSoftwareUpdateAssets(ctx, class, []fleet.AppleSoftwareUpdateAsset{
		{ProductVersion: "26.4.0", Build: "a", PostingDate: &posting, SupportedDevices: []byte(`["J1"]`)},
		{ProductVersion: "15.7.4", Build: "b", PostingDate: &posting, SupportedDevices: []byte(`["J2"]`)},
	}))

	var firstSeen time.Time
	require.NoError(t, sqlx.GetContext(ctx, ds.reader(ctx), &firstSeen, `
SELECT first_seen_at FROM apple_software_update_assets
WHERE class = ? AND product_version = ? AND build = ?`, class, "26.4.0", "a"))

	time.Sleep(10 * time.Millisecond)
	later := posting.Add(24 * time.Hour)
	require.NoError(t, ds.ReplaceAppleSoftwareUpdateAssets(ctx, class, []fleet.AppleSoftwareUpdateAsset{
		{ProductVersion: "26.4.0", Build: "a", PostingDate: &later, SupportedDevices: []byte(`["J1","J3"]`)},
		{ProductVersion: "26.4.1", Build: "c", PostingDate: &later, SupportedDevices: []byte(`["J1"]`)},
	}))

	var keptFirstSeen time.Time
	require.NoError(t, sqlx.GetContext(ctx, ds.reader(ctx), &keptFirstSeen, `
SELECT first_seen_at FROM apple_software_update_assets
WHERE class = ? AND product_version = ? AND build = ?`, class, "26.4.0", "a"))
	require.Equal(t, firstSeen, keptFirstSeen, "first_seen_at must survive upsert")

	var count int
	require.NoError(t, sqlx.GetContext(ctx, ds.reader(ctx), &count, `
SELECT COUNT(*) FROM apple_software_update_assets WHERE class = ?`, class))
	require.Equal(t, 2, count, "absent rows must be pruned")

	var devices []byte
	require.NoError(t, sqlx.GetContext(ctx, ds.reader(ctx), &devices, `
SELECT supported_devices FROM apple_software_update_assets
WHERE class = ? AND product_version = ? AND build = ?`, class, "26.4.0", "a"))
	require.JSONEq(t, `["J1","J3"]`, string(devices))

	err := ds.ReplaceAppleSoftwareUpdateAssets(ctx, class, nil)
	require.Error(t, err)
}
