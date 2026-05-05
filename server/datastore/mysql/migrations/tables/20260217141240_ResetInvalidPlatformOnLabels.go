package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20260217141240, Down_20260217141240)
}

func Up_20260217141240(tx *sql.Tx) error {
	_, err := tx.Exec(`UPDATE labels SET platform = '' WHERE platform NOT IN ('', 'centos', 'darwin', 'windows', 'ubuntu')`)
	if err != nil {
		return err
	}
	return nil
}

func Down_20260217141240(tx *sql.Tx) error {
	return nil
}
