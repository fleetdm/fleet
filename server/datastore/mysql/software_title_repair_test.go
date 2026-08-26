package mysql

import (
	"context"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/platform/mysql/testing_utils"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func insertUnlinkedSoftware(t *testing.T, ctx context.Context, ds *Datastore, hostID uint,
	name, version, source, bundleID string, upgradeCode, applicationID *string, titleID *uint,
) int64 {
	t.Helper()
	res, err := ds.writer(ctx).ExecContext(ctx,
		`INSERT INTO software (name, version, source, bundle_identifier, upgrade_code, application_id, title_id, checksum)
		 VALUES (?, ?, ?, ?, ?, ?, ?, UNHEX(MD5(?)))`,
		name, version, source, bundleID, upgradeCode, applicationID, titleID, name+version+source)
	require.NoError(t, err)
	id, err := res.LastInsertId()
	require.NoError(t, err)
	_, err = ds.writer(ctx).ExecContext(ctx,
		`INSERT INTO host_software (host_id, software_id) VALUES (?, ?)`, hostID, id)
	require.NoError(t, err)
	return id
}

func softwareTitleID(t *testing.T, ctx context.Context, ds *Datastore, query string, arg any) *uint {
	t.Helper()
	var rows []struct {
		TitleID *uint `db:"title_id"`
	}
	require.NoError(t, sqlx.SelectContext(ctx, ds.writer(ctx), &rows, query, arg))
	require.Len(t, rows, 1)
	return rows[0].TitleID
}

func titleIDByBundle(t *testing.T, ctx context.Context, ds *Datastore, bundleID string) uint {
	t.Helper()
	var id uint
	require.NoError(t, sqlx.GetContext(ctx, ds.writer(ctx), &id,
		`SELECT id FROM software_titles WHERE bundle_identifier = ?`, bundleID))
	return id
}

func TestSoftwareIngestionAfterRemoteTitleCleanup(t *testing.T) {
	const dbName = "software_remote_title_gc"

	ds := CreateNamedMySQLDS(t, dbName)
	ds2 := connectMySQL(t, dbName, new(testing_utils.DatastoreTestOptions))
	t.Cleanup(func() { ds2.Close() })

	ctx := t.Context()
	host := test.NewHost(t, ds, "host1", "10.0.0.1", "hostkey1", "hostuuid1", time.Now())

	uniqueSW := fleet.Software{Name: "UniqueApp", Version: "1.0.0", Source: "apps", BundleIdentifier: "com.example.uniqueapp"}
	otherSW := fleet.Software{Name: "OtherApp", Version: "2.0.0", Source: "apps", BundleIdentifier: "com.example.otherapp"}

	uniqueAppTitleID := func() *uint {
		return softwareTitleID(t, ctx, ds,
			`SELECT title_id FROM software WHERE bundle_identifier = ?`, uniqueSW.BundleIdentifier)
	}

	_, err := ds.UpdateHostSoftware(ctx, host.ID, []fleet.Software{uniqueSW, otherSW})
	require.NoError(t, err)
	require.NotNil(t, uniqueAppTitleID())

	_, err = ds.UpdateHostSoftware(ctx, host.ID, []fleet.Software{otherSW})
	require.NoError(t, err)

	require.NoError(t, ds2.SyncHostsSoftware(ctx, time.Now()))
	require.NoError(t, ds2.CleanupSoftwareTitles(ctx))

	var titleCount int
	err = sqlx.GetContext(ctx, ds.writer(ctx), &titleCount,
		`SELECT COUNT(*) FROM software_titles WHERE bundle_identifier = ?`, uniqueSW.BundleIdentifier)
	require.NoError(t, err)
	require.Zero(t, titleCount, "the orphaned title should have been garbage-collected")

	_, err = ds.UpdateHostSoftware(ctx, host.ID, []fleet.Software{uniqueSW, otherSW})
	require.NoError(t, err)
	require.NotNil(t, uniqueAppTitleID(), "software row was inserted with a NULL title_id")

	_, err = ds.UpdateHostSoftware(ctx, host.ID, []fleet.Software{uniqueSW})
	require.NoError(t, err)
	_, err = ds.UpdateHostSoftware(ctx, host.ID, []fleet.Software{uniqueSW, otherSW})
	require.NoError(t, err)
	require.NotNil(t, uniqueAppTitleID(), "software row with NULL title_id was never repaired")
}

func testCleanupSoftwareTitlesRepairsUnlinkedSoftware(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	host := test.NewHost(t, ds, "repairhost", "10.0.0.2", "repairkey", "repairuuid", time.Now())

	keeper := fleet.Software{Name: "KeeperApp", Version: "1.0.0", Source: "apps", BundleIdentifier: "com.example.keeper"}
	_, err := ds.UpdateHostSoftware(ctx, host.ID, []fleet.Software{keeper})
	require.NoError(t, err)
	keeperTitleID := titleIDByBundle(t, ctx, ds, keeper.BundleIdentifier)

	upgradeCode := "{2A1B3C4D-0000-0000-0000-000000000001}"
	applicationID := "com.example.androidapp"

	res, err := ds.writer(ctx).ExecContext(ctx,
		`INSERT INTO software_titles (name, source, extension_for, application_id) VALUES (?, ?, '', ?)`,
		"Android App", "android_apps", applicationID)
	require.NoError(t, err)
	androidTitleID, err := res.LastInsertId()
	require.NoError(t, err)

	newBundleSW := insertUnlinkedSoftware(t, ctx, ds, host.ID, "OrphanApp", "1.0.0", "apps", "com.example.orphan", nil, nil, nil)
	existingBundleSW := insertUnlinkedSoftware(t, ctx, ds, host.ID, "KeeperApp Beta", "2.0.0", "apps", keeper.BundleIdentifier, nil, nil, nil)
	programSW := insertUnlinkedSoftware(t, ctx, ds, host.ID, "SomeProgram", "3.0.0", "programs", "", &upgradeCode, nil, nil)
	kernelSW := insertUnlinkedSoftware(t, ctx, ds, host.ID, "linux-image-6.8.0-45-generic", "6.8.0", "deb_packages", "", nil, nil, nil)
	androidSW := insertUnlinkedSoftware(t, ctx, ds, host.ID, "Android App (renamed)", "4.0.0", "android_apps", "", nil, &applicationID, nil)

	require.NoError(t, ds.CleanupSoftwareTitles(ctx))

	titleOf := func(softwareID int64) fleet.SoftwareTitle {
		var titles []fleet.SoftwareTitle
		require.NoError(t, sqlx.SelectContext(ctx, ds.writer(ctx), &titles,
			`SELECT st.id, st.name, st.source, st.bundle_identifier, st.is_kernel, st.upgrade_code
			 FROM software s JOIN software_titles st ON st.id = s.title_id WHERE s.id = ?`, softwareID))
		require.Len(t, titles, 1, "software row %d is not linked to an existing title", softwareID)
		return titles[0]
	}

	newTitle := titleOf(newBundleSW)
	require.Equal(t, "OrphanApp", newTitle.Name)
	require.Equal(t, "apps", newTitle.Source)
	require.NotNil(t, newTitle.BundleIdentifier)
	require.Equal(t, "com.example.orphan", *newTitle.BundleIdentifier)

	require.Equal(t, keeperTitleID, titleOf(existingBundleSW).ID, "repair created a duplicate title instead of reusing the existing one")
	var keeperTitleCount int
	require.NoError(t, sqlx.GetContext(ctx, ds.writer(ctx), &keeperTitleCount,
		`SELECT COUNT(*) FROM software_titles WHERE bundle_identifier = ?`, keeper.BundleIdentifier))
	require.Equal(t, 1, keeperTitleCount)

	programTitle := titleOf(programSW)
	require.Equal(t, "programs", programTitle.Source)
	require.NotNil(t, programTitle.UpgradeCode)
	require.Equal(t, upgradeCode, *programTitle.UpgradeCode)

	require.True(t, titleOf(kernelSW).IsKernel, "repaired Linux kernel title should be marked as a kernel")

	require.EqualValues(t, androidTitleID, titleOf(androidSW).ID)

	require.NoError(t, ds.CleanupSoftwareTitles(ctx))
	require.Equal(t, newTitle.ID, titleOf(newBundleSW).ID)
	require.Equal(t, keeperTitleID, titleOf(existingBundleSW).ID)
}

func testNullDanglingTitleLinks(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	host := test.NewHost(t, ds, "danglinghost", "10.0.0.3", "danglingkey", "danglinguuid", time.Now())

	linked := fleet.Software{Name: "LinkedApp", Version: "1.0.0", Source: "apps", BundleIdentifier: "com.example.linked"}
	_, err := ds.UpdateHostSoftware(ctx, host.ID, []fleet.Software{linked})
	require.NoError(t, err)
	linkedTitleID := titleIDByBundle(t, ctx, ds, linked.BundleIdentifier)

	deletedTitleID := uint(999999)
	ghostID := insertUnlinkedSoftware(t, ctx, ds, host.ID, "GhostApp", "1.0.0", "apps", "com.example.ghost", nil, nil, &deletedTitleID)

	require.NoError(t, ds.nullDanglingTitleLinks(ctx, []uint{deletedTitleID, linkedTitleID}))

	ghostTitleID := func() *uint {
		return softwareTitleID(t, ctx, ds, `SELECT title_id FROM software WHERE id = ?`, ghostID)
	}

	require.Nil(t, ghostTitleID(), "link to the deleted title should have been cleared")

	var linkedCount int
	require.NoError(t, sqlx.GetContext(ctx, ds.writer(ctx), &linkedCount,
		`SELECT COUNT(*) FROM software WHERE title_id = ?`, linkedTitleID))
	require.Equal(t, 1, linkedCount, "link to a live title must not be cleared")

	require.NoError(t, ds.CleanupSoftwareTitles(ctx))
	repairedTitleID := ghostTitleID()
	require.NotNil(t, repairedTitleID)
	var ghostTitleBundle string
	require.NoError(t, sqlx.GetContext(ctx, ds.writer(ctx), &ghostTitleBundle,
		`SELECT bundle_identifier FROM software_titles WHERE id = ?`, *repairedTitleID))
	require.Equal(t, "com.example.ghost", ghostTitleBundle)
}
