package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260603120000, Down_20260603120000)
}

func Up_20260603120000(tx *sql.Tx) error {
	// Three columns on mdm_windows_enrollments support the Windows MDM on-demand sync (issue #43773):
	//   - poll_schedule_relaxed: the intended DMClient poll schedule (true once a host whose fleetd can be woken is
	//     relaxed). The management session reconciles against it so it does not re-send the poll Replace each session.
	//   - has_pending_commands: a denormalized flag, true when the enrollment has queued, unacknowledged commands
	//     other than internal poll-schedule Replaces. The orbit config check-in already reads this row every ~30s
	//     per host, so it reads this column instead of recomputing an EXISTS over the command queue on every poll.
	//   - fleetd_sync_capable: the last-observed X-Fleet-Capabilities CapabilityWindowsMDMSync flag, persisted by the
	//     orbit-config endpoint. The OMA-DM management session has no capability header, so it gates poll relaxation
	//     on this column instead of re-deriving the capability from host_orbit_info. Default 0; populated on the next
	//     config poll, so no backfill is needed.
	if _, err := tx.Exec(`ALTER TABLE mdm_windows_enrollments
		ADD COLUMN poll_schedule_relaxed TINYINT(1) NOT NULL DEFAULT 0,
		ADD COLUMN has_pending_commands TINYINT(1) NOT NULL DEFAULT 0,
		ADD COLUMN fleetd_sync_capable TINYINT(1) NOT NULL DEFAULT 0`); err != nil {
		return fmt.Errorf("add poll_schedule_relaxed, has_pending_commands and fleetd_sync_capable to mdm_windows_enrollments: %w", err)
	}

	// Backfill has_pending_commands so commands queued before this migration are not missed. has_pending_commands
	// defaults to 0, so we only need to flip the enrollments that have an unacknowledged queued command. Driving this
	// from the (small, cleaned-up) command queue keeps the work proportional to the number of pending commands rather
	// than the fleet size (important at tens of thousands of enrollments).
	if _, err := tx.Exec(`UPDATE mdm_windows_enrollments e
		JOIN (
			SELECT DISTINCT q.enrollment_id
			FROM windows_mdm_command_queue q
			WHERE NOT EXISTS (
				SELECT 1 FROM windows_mdm_command_results r
				WHERE r.enrollment_id = q.enrollment_id AND r.command_uuid = q.command_uuid
			)
		) pending ON pending.enrollment_id = e.id
		SET e.has_pending_commands = 1`); err != nil {
		return fmt.Errorf("backfill has_pending_commands: %w", err)
	}
	return nil
}

func Down_20260603120000(tx *sql.Tx) error {
	return nil
}
