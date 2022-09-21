package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20220914154915, Down_20220914154915)
}

func Up_20220914154915(tx *sql.Tx) error {
	_, err := tx.Exec(`
    CREATE TABLE host_disks (
        host_id                      INT(10) UNSIGNED NOT NULL,
        gigs_disk_space_available    DECIMAL(10,2) NOT NULL DEFAULT 0,
        percent_disk_space_available DECIMAL(10,2) NOT NULL DEFAULT 0,
        created_at                   TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
        updated_at                   TIMESTAMP NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

        PRIMARY KEY (host_id),
        KEY idx_host_disks_gigs_disk_space_available (gigs_disk_space_available)
    )`)
	return err
}

func Down_20220914154915(tx *sql.Tx) error {
	return nil
}
