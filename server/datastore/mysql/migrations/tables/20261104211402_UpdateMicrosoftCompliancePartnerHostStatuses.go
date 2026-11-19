package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20261104211402, Down_20261104211402)
}

// Up_20261104211402 prepares the Entra device IDs for one owner per device ID:
//   - It deletes the device IDs macOS hosts reported before Fleet required an Entra-signed token to
//     prove them. Hosts report a verified one on their next orbit config fetch.
//   - It deletes device IDs that more than one host reported, since Fleet can't tell which one owns
//     it. The owner reports it again on its next check-in.
//   - It adds a verified column, set when the device ID comes from an Entra-signed token, and a
//     non-unique index on the device ID to look up the hosts that have one. The one-host rule itself
//     is enforced by CreateHostConditionalAccessStatus, not by the index.
func Up_20261104211402(tx *sql.Tx) error {
	// The statuses table has at most one row per host; the join is on hosts' primary key.
	if _, err := tx.Exec(`
		DELETE s FROM microsoft_compliance_partner_host_statuses s
		JOIN hosts h ON h.id = s.host_id
		WHERE h.platform = 'darwin'`,
	); err != nil {
		return fmt.Errorf("delete macOS host conditional access statuses: %w", err)
	}

	// The derived table is materialized, which MySQL requires to delete from the table it reads.
	if _, err := tx.Exec(`
		DELETE s FROM microsoft_compliance_partner_host_statuses s
		JOIN (
			SELECT device_id FROM microsoft_compliance_partner_host_statuses
			GROUP BY device_id HAVING COUNT(*) > 1
		) d ON d.device_id = s.device_id`,
	); err != nil {
		return fmt.Errorf("delete shared host conditional access device IDs: %w", err)
	}

	if !columnExists(tx, "microsoft_compliance_partner_host_statuses", "verified") {
		if _, err := tx.Exec(`ALTER TABLE microsoft_compliance_partner_host_statuses
			ADD COLUMN verified tinyint(1) NOT NULL DEFAULT 0`); err != nil {
			return fmt.Errorf("add verified column to microsoft_compliance_partner_host_statuses: %w", err)
		}
	}
	if !indexExistsTx(tx, "microsoft_compliance_partner_host_statuses", "idx_microsoft_compliance_partner_host_statuses_device_id") {
		if _, err := tx.Exec(`ALTER TABLE microsoft_compliance_partner_host_statuses
			ADD INDEX idx_microsoft_compliance_partner_host_statuses_device_id (device_id),
			ALGORITHM=INPLACE, LOCK=NONE`); err != nil {
			return fmt.Errorf("add device_id index to microsoft_compliance_partner_host_statuses: %w", err)
		}
	}
	return nil
}

func Down_20261104211402(tx *sql.Tx) error {
	return nil
}
