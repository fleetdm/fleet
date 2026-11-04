package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20261103181933, Down_20261103181933)
}

// Up_20261103181933 deletes the Entra device IDs macOS hosts reported before Fleet required
// an Entra-signed token to prove them, so only verified device IDs get compliance updates.
// Hosts report a verified one on their next orbit config fetch.
//
// It also indexes the device ID so Fleet can reject one already assigned to another host. The
// index isn't unique because existing rows may already share a device ID.
func Up_20261103181933(tx *sql.Tx) error {
	// The statuses table has at most one row per host; the join is on hosts' primary key.
	if _, err := tx.Exec(`
		DELETE s FROM microsoft_compliance_partner_host_statuses s
		JOIN hosts h ON h.id = s.host_id
		WHERE h.platform = 'darwin'`,
	); err != nil {
		return fmt.Errorf("delete macOS host conditional access statuses: %w", err)
	}

	if indexExistsTx(tx, "microsoft_compliance_partner_host_statuses", "idx_microsoft_compliance_partner_host_statuses_device_id") {
		return nil
	}
	if _, err := tx.Exec(`ALTER TABLE microsoft_compliance_partner_host_statuses
		ADD INDEX idx_microsoft_compliance_partner_host_statuses_device_id (device_id),
		ALGORITHM=INPLACE, LOCK=NONE`); err != nil {
		return fmt.Errorf("add device_id index to microsoft_compliance_partner_host_statuses: %w", err)
	}
	return nil
}

func Down_20261103181933(tx *sql.Tx) error {
	return nil
}
