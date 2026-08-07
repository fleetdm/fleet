package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260606051849, Down_20260606051849)
}

func Up_20260606051849(tx *sql.Tx) error {
	// acked_at is the soft-dequeue marker for the Windows MDM command queue. Queue rows persist after acknowledgment until the
	// periodic GC, so every predicate that needs "pending = queued and unacknowledged" had to anti-join windows_mdm_command_results
	// per row. With the wake model recomputing the denormalized has_pending_commands flag during ack processing, that anti-join walk
	// over acked-but-not-yet-GC'd rows became the top writer statement under bulk profile waves. The ack transaction now stamps
	// acked_at on exactly the rows it records results for, turning every pending predicate into an index probe on (enrollment_id,
	// acked_at) and the GC into an index range delete. Two indexes: the composite serves the hot-path pending probes (WHERE
	// enrollment_id = ? AND acked_at IS NULL, index-only), and the single-column index serves the GC's range delete (WHERE acked_at <
	// ?), which cannot use the second column of the composite.
	if _, err := tx.Exec(`ALTER TABLE windows_mdm_command_queue
		ADD COLUMN acked_at DATETIME(6) NULL DEFAULT NULL,
		ADD INDEX idx_win_mdm_cmd_queue_enrollment_acked (enrollment_id, acked_at),
		ADD INDEX idx_win_mdm_cmd_queue_acked (acked_at)`); err != nil {
		return fmt.Errorf("add acked_at to windows_mdm_command_queue: %w", err)
	}

	// Backfill: rows acknowledged before this migration (result row exists) must not become visible as pending under the
	// new acked_at IS NULL predicate, or every previously delivered command would be re-sent on the next session. Use the
	// result row's created_at so the GC's 1-hour age floor keeps its meaning. The join is PK-to-PK; the work is bounded
	// by the acked-but-not-yet-GC'd backlog (at most the GC interval's worth of traffic).
	if _, err := tx.Exec(`UPDATE windows_mdm_command_queue q
		JOIN windows_mdm_command_results r
			ON r.enrollment_id = q.enrollment_id AND r.command_uuid = q.command_uuid
		SET q.acked_at = r.created_at
		WHERE q.acked_at IS NULL`); err != nil {
		return fmt.Errorf("backfill acked_at from windows_mdm_command_results: %w", err)
	}
	return nil
}

func Down_20260606051849(tx *sql.Tx) error {
	return nil
}
