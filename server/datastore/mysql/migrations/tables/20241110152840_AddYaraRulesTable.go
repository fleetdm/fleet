package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20241110152840, Down_20241110152840)
}

func Up_20241110152840(tx *sql.Tx) error {
	_, err := tx.Exec(`
CREATE TABLE yara_rules (
	id INT(10) UNSIGNED NOT NULL AUTO_INCREMENT,
	name VARCHAR(255) NOT NULL,
	contents MEDIUMTEXT NOT NULL,
	PRIMARY KEY (id),
	UNIQUE KEY idx_yara_rules_name (name) 
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci;`)
	if err != nil {
		return fmt.Errorf("failed to create yara_rules table: %w", err)
	}
	return nil
}

func Down_20241110152840(tx *sql.Tx) error {
	return nil
}
