package mysql

import (
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/platform/mysql/testing_utils"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

// TestSoftwareIngestionAfterRemoteTitleCleanup reproduces software rows being written
// with a NULL title_id when a software title is garbage-collected by the cleanup crons
// running on a different Fleet instance (two datastore handles on one database), then
// re-reported by the host. The row must end up linked to a title.
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
		var rows []struct {
			TitleID *uint `db:"title_id"`
		}
		err := sqlx.SelectContext(ctx, ds.writer(ctx), &rows,
			`SELECT title_id FROM software WHERE bundle_identifier = ?`, uniqueSW.BundleIdentifier)
		require.NoError(t, err)
		require.Len(t, rows, 1)
		return rows[0].TitleID
	}

	// Initial report: creates the software rows and titles.
	_, err := ds.UpdateHostSoftware(ctx, host.ID, []fleet.Software{uniqueSW, otherSW})
	require.NoError(t, err)
	require.NotNil(t, uniqueAppTitleID())

	// The host uninstalls UniqueApp; no other host has it, so its software row and
	// title are now orphaned.
	_, err = ds.UpdateHostSoftware(ctx, host.ID, []fleet.Software{otherSW})
	require.NoError(t, err)

	// The hourly cron cycle runs on the second instance, as in production
	// (vuln_process.go): SyncHostsSoftware deletes the unused software row, then
	// CleanupSoftwareTitles deletes the orphaned title.
	require.NoError(t, ds2.SyncHostsSoftware(ctx, time.Now()))
	require.NoError(t, ds2.CleanupSoftwareTitles(ctx))

	var titleCount int
	err = sqlx.GetContext(ctx, ds.writer(ctx), &titleCount,
		`SELECT COUNT(*) FROM software_titles WHERE bundle_identifier = ?`, uniqueSW.BundleIdentifier)
	require.NoError(t, err)
	require.Zero(t, titleCount, "the orphaned title should have been garbage-collected")

	// The host reinstalls UniqueApp and reports it to the first instance. The row must
	// end up linked to a title.
	_, err = ds.UpdateHostSoftware(ctx, host.ID, []fleet.Software{uniqueSW, otherSW})
	require.NoError(t, err)
	require.NotNil(t, uniqueAppTitleID(), "software row was inserted with a NULL title_id")

	// The row must stay linked on subsequent report cycles.
	_, err = ds.UpdateHostSoftware(ctx, host.ID, []fleet.Software{uniqueSW})
	require.NoError(t, err)
	_, err = ds.UpdateHostSoftware(ctx, host.ID, []fleet.Software{uniqueSW, otherSW})
	require.NoError(t, err)
	require.NotNil(t, uniqueAppTitleID(), "software row lost its title on a later cycle")
}
