package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20231130132828, Down_20231130132828)
}

func Up_20231130132828(tx *sql.Tx) error {
	_, err := tx.Exec(`
CREATE TABLE software_titles (
	id INT(10) UNSIGNED NOT NULL AUTO_INCREMENT,
	name VARCHAR(255) NOT NULL,
	source VARCHAR(64) NOT NULL,
	PRIMARY KEY (id),
	UNIQUE KEY idx_software_titles_name_source (name, source) 
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci;`)
	if err != nil {
		return fmt.Errorf("failed to create software_titles table: %w", err)
	}
	return nil
}

func Down_20231130132828(tx *sql.Tx) error {
	return nil
}
