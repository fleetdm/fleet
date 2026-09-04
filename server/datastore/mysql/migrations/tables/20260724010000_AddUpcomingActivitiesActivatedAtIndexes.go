package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260724010000, Down_20260724010000)
}

func Up_20260724010000(tx *sql.Tx) error {
	// Both the fleet-initiated release cron and the unblock job find hosts
	// whose queue has no activated activity via a self anti-join on
	// upcoming_activities. Neither side of that join was indexed on
	// activated_at, forcing full scans that degrade with queue depth (which is
	// deepest exactly when the queue is backing up).
	//
	// (host_id, activated_at) serves the anti-join probe ("does this host have
	// an activated activity?"). (activated_at, fleet_initiated, created_at,
	// host_id) covers the release cron's scan for the oldest waiting
	// fleet-initiated activities without touching rows.
	if !indexExistsTx(tx, "upcoming_activities", "idx_upcoming_activities_host_id_activated_at") {
		if _, err := tx.Exec(`
			ALTER TABLE upcoming_activities
			ADD INDEX idx_upcoming_activities_host_id_activated_at (host_id, activated_at)`); err != nil {
			return fmt.Errorf("add host_id, activated_at index to upcoming_activities: %w", err)
		}
	}
	if !indexExistsTx(tx, "upcoming_activities", "idx_upcoming_activities_activated_at_fleet_initiated") {
		if _, err := tx.Exec(`
			ALTER TABLE upcoming_activities
			ADD INDEX idx_upcoming_activities_activated_at_fleet_initiated (activated_at, fleet_initiated, created_at, host_id)`); err != nil {
			return fmt.Errorf("add activated_at, fleet_initiated index to upcoming_activities: %w", err)
		}
	}
	return nil
}

func Down_20260724010000(tx *sql.Tx) error {
	return nil
}
