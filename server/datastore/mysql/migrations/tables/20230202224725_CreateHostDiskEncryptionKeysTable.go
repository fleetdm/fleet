package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20230202224725, Down_20230202224725)
}

func Up_20230202224725(tx *sql.Tx) error {
	// `decryptable` can be NULL to signal that we have fetched the key but
	// we don't know yet if we can decrypt it or not, the related index is
	// to aid querying for this scenario by taking adventage of MySQL's IS
	// NULL optimization:
	// https://dev.mysql.com/doc/refman/5.7/en/is-null-optimization.html
	_, err := tx.Exec(`
	  CREATE TABLE IF NOT EXISTS host_disk_encryption_keys (
            host_id             int(10) UNSIGNED NOT NULL,
	    base64_encrypted    text NOT NULL,
	    decryptable         tinyint(1) DEFAULT NULL,
	    created_at          timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
	    updated_at          timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

	    PRIMARY KEY (host_id),
	    KEY idx_host_disk_encryption_keys_decryptable (decryptable)
	 ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;`)
	return err
}

func Down_20230202224725(tx *sql.Tx) error {
	return nil
}
