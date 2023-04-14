package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20230413193404, Down_20230413193404)
}

func Up_20230413193404(tx *sql.Tx) error {
	_, err := tx.Exec(`
		CREATE TABLE host_software_installed_paths
		(
			host_id        INT UNSIGNED NOT NULL,
			software_id    INT UNSIGNED NOT NULL,
			installed_path TEXT         NOT NULL,
			PRIMARY KEY (host_id, software_id)
		)
	`)
	if err != nil {
		return err
	}

	return nil
}

func Down_20230413193404(tx *sql.Tx) error {
	return nil
}
