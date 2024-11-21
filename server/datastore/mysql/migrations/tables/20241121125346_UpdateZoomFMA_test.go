package tables

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

func TestUp_20241121125346(t *testing.T) {
	db := applyUpToPrev(t)

	// Insert a scheduled and a triggered job run for maintained_apps
	execNoErr(t, db, `INSERT INTO cron_stats (name, instance, stats_type, status) VALUES (?, 'foo', ?, ?)`, fleet.CronMaintainedApps, fleet.CronStatsTypeScheduled, fleet.CronStatsStatusCompleted)
	execNoErr(t, db, `INSERT INTO cron_stats (name, instance, stats_type, status) VALUES (?, 'foo', ?, ?)`, fleet.CronMaintainedApps, fleet.CronStatsTypeTriggered, fleet.CronStatsStatusCompleted)

	// Apply current migration.
	applyNext(t, db)

	//
	// Check data, insert new entries, e.g. to verify migration is safe.
	//
	// ...
}
