package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20251124140138, Down_20251124140138)
}

func Up_20251124140138(tx *sql.Tx) error {
	_, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS certificate_templates (
			id INT UNSIGNED NOT NULL AUTO_INCREMENT,
			team_id INT UNSIGNED NOT NULL,
			certificate_authority_id INT NOT NULL,
			name VARCHAR(255) NOT NULL,
			subject_name text NOT NULL,
			created_at timestamp NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

			PRIMARY KEY (id),
			UNIQUE KEY idx_cert_team_name (team_id, name),
			FOREIGN KEY (team_id) REFERENCES teams (id),
			FOREIGN KEY (certificate_authority_id) REFERENCES certificate_authorities (id)
		) DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci;
	`)
	return err
}

func Down_20251124140138(tx *sql.Tx) error {
	return nil
}
