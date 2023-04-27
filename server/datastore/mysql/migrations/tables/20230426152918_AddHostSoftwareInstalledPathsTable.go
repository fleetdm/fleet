package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20230426152918, Down_20230426152918)
}

func Up_20230426152918(tx *sql.Tx) error {
	_, err := tx.Exec(`
		CREATE TABLE host_software_installed_paths
		(
			id             BIGINT PRIMARY KEY,
			host_id        INT    UNSIGNED NOT NULL,
			software_id    BIGINT UNSIGNED NOT NULL,
			installed_path TEXT            NOT NULL,
			KEY host_id_software_id_idx (host_id, software_id)
		)
	`)
	if err != nil {
		return err
	}

	return nil
}

func Down_20230426152918(tx *sql.Tx) error {
	return nil
}
