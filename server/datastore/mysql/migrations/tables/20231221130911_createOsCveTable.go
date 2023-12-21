package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20231221130911, Down_20231221130911)
}

func Up_20231221130911(tx *sql.Tx) error {
	createQuery := `
		CREATE TABLE operating_system_cve (
			id int(10) unsigned NOT NULL AUTO_INCREMENT,
			cve varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
			created_at timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			operating_system_id int(10) unsigned NOT NULL,
			resolved_in_version varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
			PRIMARY KEY (id),
			UNIQUE KEY unq_cve_operating_system (cve,operating_system_id),
			KEY fk_operating_system_cve_operating_system_id (operating_system_id)
		)
	`

	if _, err := tx.Exec(createQuery); err != nil {
		return err
	}

	return nil
}

func Down_20231221130911(tx *sql.Tx) error {
	return nil
}
