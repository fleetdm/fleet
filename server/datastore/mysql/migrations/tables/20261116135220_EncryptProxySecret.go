package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20261116140000, Down_20261116140000)
}

func Up_20261116140000(tx *sql.Tx) error {
	if colType := columnDataType(tx, "microsoft_compliance_partner_integrations", "proxy_server_secret"); colType != "" && colType != "blob" {
		if _, err := tx.Exec(`ALTER TABLE microsoft_compliance_partner_integrations MODIFY COLUMN proxy_server_secret blob NOT NULL`); err != nil {
			return fmt.Errorf("alter proxy_server_secret to blob: %w", err)
		}
	}
	return nil
}

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

func Down_20261116140000(tx *sql.Tx) error {
	return nil
}
