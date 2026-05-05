package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20251121124239, Down_20251121124239)
}

func Up_20251121124239(tx *sql.Tx) error {
	_, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS host_certificate_templates (
			id INT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
			host_uuid varchar(36) NOT NULL,
			certificate_template_id INT UNSIGNED NOT NULL,
			fleet_challenge char(32) NOT NULL,
			status varchar(20) NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
		) DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci;
	`)
	return err
}

func Down_20251121124239(tx *sql.Tx) error {
	return nil
}
