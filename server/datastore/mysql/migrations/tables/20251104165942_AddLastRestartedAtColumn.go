package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20251104165942, Down_20251104165942)
}

func Up_20251104165942(tx *sql.Tx) error {
	stmt := `
		ALTER TABLE hosts
		ADD COLUMN last_restarted_at datetime(6) DEFAULT '0001-01-01 00:00:00.000000'
	`
	if _, err := tx.Exec(stmt); err != nil {
		return fmt.Errorf("add last_restarted_at to hosts: %w", err)
	}

	updateStmt := `
		UPDATE hosts
		SET last_restarted_at =
			CASE
				WHEN (uptime = 0 OR detail_updated_at IS NULL) THEN '0001-01-01 00:00:00.000000'
				ELSE DATE_SUB(detail_updated_at, INTERVAL uptime/1000 MICROSECOND)
			END
	`
	if _, err := tx.Exec(updateStmt); err != nil {
		return fmt.Errorf("update last_restarted_at in hosts: %w", err)
	}
	return nil
}

func Down_20251104165942(tx *sql.Tx) error {
	return nil
}
