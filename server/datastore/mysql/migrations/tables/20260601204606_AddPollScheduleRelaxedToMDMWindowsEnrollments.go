package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260601204606, Down_20260601204606)
}

func Up_20260601204606(tx *sql.Tx) error {
	// Two columns on mdm_windows_enrollments support the Windows MDM on-demand sync (issue #43773):
	//   - poll_schedule_relaxed: the intended DMClient poll schedule (true once a host whose fleetd can be woken is
	//     relaxed). The management session reconciles against it so it does not re-send the poll Replace each session.
	//   - has_pending_commands: a denormalized flag, true when the enrollment has queued, unacknowledged commands
	//     other than internal poll-schedule Replaces. The orbit config check-in already reads this row every ~30s
	//     per host, so it reads this column instead of recomputing an EXISTS over the command queue on every poll.
	if _, err := tx.Exec(`ALTER TABLE mdm_windows_enrollments
		ADD COLUMN poll_schedule_relaxed TINYINT(1) NOT NULL DEFAULT 0,
		ADD COLUMN has_pending_commands TINYINT(1) NOT NULL DEFAULT 0`); err != nil {
		return fmt.Errorf("add poll_schedule_relaxed and has_pending_commands to mdm_windows_enrollments: %w", err)
	}

	// Backfill has_pending_commands so commands queued before this migration are not missed. The poll-schedule
	// LocURI is excluded (matching the recompute) so internal poll tuning never sets the flag. Enrollments with no
	// queued commands short-circuit the EXISTS cheaply.
	const pollLocURI = "./Device/Vendor/MSFT/DMClient/Provider/Fleet/Poll/IntervalForFirstSetOfRetries"
	if _, err := tx.Exec(`UPDATE mdm_windows_enrollments e
		SET e.has_pending_commands = EXISTS (
			SELECT 1
			FROM windows_mdm_command_queue q
			JOIN windows_mdm_commands c ON c.command_uuid = q.command_uuid
			WHERE q.enrollment_id = e.id
				AND c.target_loc_uri <> ?
				AND NOT EXISTS (
					SELECT 1 FROM windows_mdm_command_results r
					WHERE r.enrollment_id = q.enrollment_id AND r.command_uuid = q.command_uuid
				)
		)`, pollLocURI); err != nil {
		return fmt.Errorf("backfill has_pending_commands: %w", err)
	}
	return nil
}

func Down_20260601204606(tx *sql.Tx) error {
	return nil
}
