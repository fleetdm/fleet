package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20261110152908, Down_20261110152908)
}

func Up_20261110152908(tx *sql.Tx) error {
	// Change proxy_server_secret from varchar(64) to blob to hold encrypted data.
	if colType := columnDataType(tx, "microsoft_compliance_partner_integrations", "proxy_server_secret"); colType != "" && colType != "blob" {
		if _, err := tx.Exec(`ALTER TABLE microsoft_compliance_partner_integrations MODIFY COLUMN proxy_server_secret blob NOT NULL`); err != nil {
			return fmt.Errorf("alter proxy_server_secret to blob: %w", err)
		}
	}

	// Change bootstrap_token_b64 from text to mediumblob to hold encrypted data.
	if colType := columnDataType(tx, "nano_devices", "bootstrap_token_b64"); colType != "" && colType != "mediumblob" {
		if _, err := tx.Exec(`ALTER TABLE nano_devices MODIFY COLUMN bootstrap_token_b64 mediumblob`); err != nil {
			return fmt.Errorf("alter bootstrap_token_b64 to mediumblob: %w", err)
		}
	}

	// unlock_token is already mediumblob, no change needed.

	return nil
}

// columnDataType returns the DATA_TYPE for the given table.column, or "" if
// the column does not exist.
func columnDataType(tx *sql.Tx, table, column string) string {
	var dt string
	err := tx.QueryRow(`
SELECT DATA_TYPE FROM information_schema.columns
WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ? AND COLUMN_NAME = ?`,
		table, column,
	).Scan(&dt)
	if err != nil {
		return ""
	}
	return dt
}

func Down_20261110152908(tx *sql.Tx) error {
	return nil
}
