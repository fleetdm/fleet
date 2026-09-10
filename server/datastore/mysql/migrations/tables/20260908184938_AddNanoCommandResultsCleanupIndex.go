package tables

import (
	"database/sql"
	"fmt"
	"strings"
)

func init() {
	MigrationClient.AddMigration(Up_20260908184938, Down_20260908184938)
}

// Up_20260908184938 adds the index behind the Apple MDM command retention
// sweep, which scans terminal results by status and walks them in
// (updated_at, id, command_uuid) order. InnoDB appends the primary key to
// every secondary index, so (status, updated_at) serves that order without a
// sort.
//
// The single-column status index is dropped: the new index's prefix covers
// every query that used it, and both would otherwise be maintained on every
// ack.
func Up_20260908184938(tx *sql.Tx) error {
	const table = "nano_command_results"

	var clauses []string
	if !indexExistsTx(tx, table, "idx_ncr_status_updated_at") {
		clauses = append(clauses, "ADD INDEX idx_ncr_status_updated_at (status, updated_at)")
	}
	if indexExistsTx(tx, table, "status") {
		clauses = append(clauses, "DROP INDEX `status`")
	}
	if len(clauses) == 0 {
		return nil
	}

	stmt := fmt.Sprintf("ALTER TABLE %s %s, ALGORITHM=INPLACE, LOCK=NONE", table, strings.Join(clauses, ", "))
	if _, err := tx.Exec(stmt); err != nil {
		return fmt.Errorf("failed to replace status index on %s: %w", table, err)
	}
	return nil
}

func Down_20260908184938(tx *sql.Tx) error {
	return nil
}
