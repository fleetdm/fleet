package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20230517140952, Down_20230517140952)
}

func Up_20230517140952(tx *sql.Tx) error {
	_, err := tx.Exec(`
		CREATE TABLE host_software_installed_paths
		(
			id             BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			host_id        INT    UNSIGNED NOT NULL,
			software_id    BIGINT UNSIGNED NOT NULL,
			installed_path TEXT            NOT NULL,
			PRIMARY KEY (id),
			KEY host_id_software_id_idx (host_id, software_id)
		) DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci
	`)
	if err != nil {
		return err
	}
	return nil
}

func Down_20230517140952(tx *sql.Tx) error {
	return nil
}
