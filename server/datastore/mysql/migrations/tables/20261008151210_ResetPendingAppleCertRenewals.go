package tables

import (
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
)

func init() {
	MigrationClient.AddMigration(Up_20261008151210, Down_20261008151210)
}

// Up_20261008151210 cancels every pending Apple MDM certificate renewal, like ResetPendingCertRenewals, so none
// stays outstanding with a static SCEP challenge. The renewal cron re-sends them with per-host challenges, and ACME
// renewals as acme_renewal enrollments.
func Up_20261008151210(tx *sql.Tx) error {
	const batchSize = 1000
	for {
		var cmdUUIDs []string
		rows, err := tx.Query(`SELECT renew_command_uuid FROM nano_cert_auth_associations WHERE renew_command_uuid IS NOT NULL LIMIT ?`, batchSize)
		if err != nil {
			return fmt.Errorf("selecting pending cert renewals: %w", err)
		}
		for rows.Next() {
			var cmdUUID string
			if err := rows.Scan(&cmdUUID); err != nil {
				rows.Close()
				return fmt.Errorf("scanning pending cert renewal: %w", err)
			}
			cmdUUIDs = append(cmdUUIDs, cmdUUID)
		}
		if err := rows.Close(); err != nil {
			return fmt.Errorf("closing pending cert renewals: %w", err)
		}
		if err := rows.Err(); err != nil {
			return fmt.Errorf("iterating pending cert renewals: %w", err)
		}
		if len(cmdUUIDs) == 0 {
			return nil
		}

		stmt, args, err := sqlx.In(`UPDATE nano_enrollment_queue SET active = 0 WHERE command_uuid IN (?)`, cmdUUIDs)
		if err != nil {
			return fmt.Errorf("building query to deactivate cert renewal commands: %w", err)
		}
		if _, err := tx.Exec(stmt, args...); err != nil {
			return fmt.Errorf("deactivating cert renewal commands: %w", err)
		}

		stmt, args, err = sqlx.In(`UPDATE nano_cert_auth_associations SET renew_command_uuid = NULL WHERE renew_command_uuid IN (?)`, cmdUUIDs)
		if err != nil {
			return fmt.Errorf("building query to reset pending cert renewals: %w", err)
		}
		if _, err := tx.Exec(stmt, args...); err != nil {
			return fmt.Errorf("resetting pending cert renewals: %w", err)
		}
	}
}

func Down_20261008151210(tx *sql.Tx) error {
	return nil
}
