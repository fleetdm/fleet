package tables

import (
	"database/sql"
	"fmt"
	"strings"
)

func init() {
	MigrationClient.AddMigration(Up_20260922041729, Down_20260922041729)
}

func Up_20260922041729(tx *sql.Tx) error {
	const table = "nano_command_results"

	// The retention sweep scans by (status, updated_at); the single-column status
	// index it replaces then only adds write cost on the hot ack path, so drop it.
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

func Down_20260922041729(tx *sql.Tx) error {
	return nil
}
