package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20221014084130, Down_20221014084130)
}

func Up_20221014084130(tx *sql.Tx) error {
	_, err := tx.Exec(`
    CREATE TABLE host_orbit_info (
        host_id				INT(10) UNSIGNED NOT NULL,
        version				VARCHAR(50) NOT NULL,

        PRIMARY KEY (host_id),
        KEY idx_host_orbit_info_version (version)
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;`)
	return err
}

func Down_20221014084130(tx *sql.Tx) error {
	return nil
}
