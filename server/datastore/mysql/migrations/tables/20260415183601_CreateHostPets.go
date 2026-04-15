package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260415183601, Down_20260415183601)
}

func Up_20260415183601(tx *sql.Tx) error {
	if _, err := tx.Exec(`
		CREATE TABLE host_pets (
			id                 INT UNSIGNED NOT NULL AUTO_INCREMENT,
			host_id            INT UNSIGNED NOT NULL,
			name               VARCHAR(64) COLLATE utf8mb4_unicode_ci NOT NULL,
			species            VARCHAR(32) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'cat',
			health             TINYINT UNSIGNED NOT NULL DEFAULT 80,
			happiness          TINYINT UNSIGNED NOT NULL DEFAULT 80,
			hunger             TINYINT UNSIGNED NOT NULL DEFAULT 20,
			cleanliness        TINYINT UNSIGNED NOT NULL DEFAULT 80,
			last_interacted_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
			created_at         TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
			updated_at         TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
			PRIMARY KEY (id),
			UNIQUE KEY idx_host_pets_host_id (host_id),
			CONSTRAINT fk_host_pets_host_id FOREIGN KEY (host_id) REFERENCES hosts (id) ON DELETE CASCADE
		)
	`); err != nil {
		return fmt.Errorf("creating host_pets table: %w", err)
	}
	return nil
}

func Down_20260415183601(tx *sql.Tx) error {
	return nil
}
