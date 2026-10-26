package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20261026111120, Down_20261026111120)
}

// Up_20261026111120 repairs recovery lock rows that had a SetRecoveryLock in flight when the
// server was upgraded from a version before 20260902091843. Those versions didn't record which
// command a row was waiting on, and that migration added the columns empty, so the device's
// result matches no row and the host stays pending forever. Only those versions leave a pending
// row with none of the command UUIDs set.
func Up_20261026111120(tx *sql.Tx) error {
	rows, err := tx.Query(`
		SELECT t.host_uuid, t.first_set, t.command_uuid, ncr.status
		FROM (
			SELECT
				rkp.host_uuid,
				rkp.operation_type = 'install' AND rkp.pending_encrypted_password IS NULL AS first_set,
				(
					SELECT q.command_uuid
					FROM nano_enrollment_queue q
					JOIN nano_commands c ON c.command_uuid = q.command_uuid AND c.request_type = 'SetRecoveryLock'
					WHERE q.id = rkp.host_uuid
					ORDER BY q.created_at DESC, q.command_uuid DESC
					LIMIT 1
				) AS command_uuid
			FROM host_recovery_key_passwords rkp
			WHERE rkp.deleted = 0
			  AND rkp.status = 'pending'
			  AND rkp.pending_set_command_uuid IS NULL
			  AND rkp.pending_verify_command_uuid IS NULL
			  AND rkp.set_command_uuid IS NULL
			  AND rkp.verify_command_uuid IS NULL
		) t
		LEFT JOIN nano_command_results ncr ON ncr.id = t.host_uuid AND ncr.command_uuid = t.command_uuid`)
	if err != nil {
		return fmt.Errorf("select recovery lock rows in flight at upgrade: %w", err)
	}

	type inFlight struct {
		hostUUID    string
		firstSet    bool
		commandUUID sql.NullString
		status      sql.NullString
	}
	var inFlightRows []inFlight
	for rows.Next() {
		var r inFlight
		if err := rows.Scan(&r.hostUUID, &r.firstSet, &r.commandUUID, &r.status); err != nil {
			rows.Close()
			return fmt.Errorf("scan recovery lock row in flight at upgrade: %w", err)
		}
		inFlightRows = append(inFlightRows, r)
	}
	if err := rows.Close(); err != nil {
		return fmt.Errorf("close recovery lock rows in flight at upgrade: %w", err)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate recovery lock rows in flight at upgrade: %w", err)
	}

	for _, r := range inFlightRows {
		var stmt string
		var args []any
		switch {
		case !r.commandUUID.Valid:
			// Nothing to tell whether the device took the password, so keep every password and
			// let the cron send the set again.
			stmt = `UPDATE host_recovery_key_passwords SET status = NULL WHERE host_uuid = ?`
			args = []any{r.hostUUID}

		case !r.status.Valid || r.status.String == "NotNow":
			// Not answered yet: record the command so its result completes the set -> verify flow.
			// Earlier first sets stored the new password in the active slot, and the queued command
			// still reads it from there, so it's copied to the pending slot the verify reads.
			stmt = `UPDATE host_recovery_key_passwords
				SET pending_set_command_uuid = ?,
					pending_encrypted_password = IF(?, encrypted_password, pending_encrypted_password)
				WHERE host_uuid = ?`
			args = []any{r.commandUUID.String, r.firstSet, r.hostUUID}

		case r.status.String == "Acknowledged":
			// The device already holds the new password and its result was dropped. A verifying row
			// without a pending verify command is picked up by the recovery lock cron, which sends it.
			stmt = `UPDATE host_recovery_key_passwords
				SET status = 'verifying', set_command_uuid = ?,
					pending_encrypted_password = IF(?, encrypted_password, pending_encrypted_password)
				WHERE host_uuid = ?`
			args = []any{r.commandUUID.String, r.firstSet, r.hostUUID}

		default:
			// The device rejected the set, so it doesn't hold the password a first set stored.
			// The cron sends the set again; a rotation keeps the active password it needs.
			stmt = `UPDATE host_recovery_key_passwords
				SET status = NULL, encrypted_password = IF(?, NULL, encrypted_password)
				WHERE host_uuid = ?`
			args = []any{r.firstSet, r.hostUUID}
		}
		if _, err := tx.Exec(stmt, args...); err != nil {
			return fmt.Errorf("repair recovery lock row for host %s: %w", r.hostUUID, err)
		}
	}

	return nil
}

func Down_20261026111120(tx *sql.Tx) error {
	return nil
}
