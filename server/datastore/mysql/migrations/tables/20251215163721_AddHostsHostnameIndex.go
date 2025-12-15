package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20251215163721, Down_20251215163721)
}

func Up_20251215163721(tx *sql.Tx) error {
	_, err := tx.Exec(`
	ALTER TABLE hosts ADD INDEX idx_hosts_hostname (hostname)
	`)
	return err
}

func Down_20251215163721(tx *sql.Tx) error {
	return nil
}
