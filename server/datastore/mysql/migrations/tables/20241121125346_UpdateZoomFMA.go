package tables

import (
	"database/sql"
	"fmt"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

func init() {
	MigrationClient.AddMigration(Up_20241121125346, Down_20241121125346)
}

func Up_20241121125346(tx *sql.Tx) error {
	// Remove the existing Zoom fleet-maintained app
	_, err := tx.Exec(`DELETE FROM fleet_library_apps WHERE token = 'zoom'`)
	if err != nil {
		return fmt.Errorf("failed to remove existing zoom app from fleet_library_apps table: %w", err)
	}

	// Clear out scheduled runs for the maintained_apps cron. This will force the cron to run on
	// next server start and pull in the updated Zoom FMA
	_, err = tx.Exec(`DELETE FROM cron_stats WHERE name = ? AND stats_type = ?`, fleet.CronMaintainedApps, fleet.CronStatsTypeScheduled)
	if err != nil {
		return fmt.Errorf("failed to clear past scheduled runs of maintained_apps from cron_stats table: %w", err)
	}

	return nil
}

func Down_20241121125346(tx *sql.Tx) error {
	return nil
}
