package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20251111105418, Down_20251111105418)
}

func Up_20251111105418(tx *sql.Tx) error {
	stmt := `
CREATE TABLE in_house_app_software_categories (
	id                   INT UNSIGNED NOT NULL AUTO_INCREMENT,
	software_category_id INT UNSIGNED NOT NULL,
	in_house_app_id      INT UNSIGNED NOT NULL,
	created_at           DATETIME(6) DEFAULT CURRENT_TIMESTAMP(6),

	PRIMARY KEY (id),
	UNIQUE KEY (in_house_app_id, software_category_id),
	CONSTRAINT fk_in_house_apps_id FOREIGN KEY (in_house_app_id) REFERENCES in_house_apps (id) ON DELETE CASCADE,
	CONSTRAINT fk_software_category_id FOREIGN KEY (software_category_id) REFERENCES software_categories (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
	`
	if _, err := tx.Exec(stmt); err != nil {
		return fmt.Errorf("create in_house_app_software_categories table: %w", err)
	}
	return nil
}

func Down_20251111105418(tx *sql.Tx) error {
	return nil
}
