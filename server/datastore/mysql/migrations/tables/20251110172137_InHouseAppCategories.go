package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20251110172137, Down_20251110172137)
}

func Up_20251110172137(tx *sql.Tx) error {

	createLabelMappingTableStmt := `
CREATE TABLE in_house_app_software_categories (
  id int unsigned NOT NULL AUTO_INCREMENT,
  software_category_id int unsigned NOT NULL,
  in_house_app_id int unsigned NOT NULL,
  created_at datetime(6) DEFAULT CURRENT_TIMESTAMP(6),
  PRIMARY KEY (id),
  UNIQUE KEY idx_unique_in_house_app_id_software_category_id (in_house_app_id,software_category_id),
  KEY software_category_id (software_category_id),
  CONSTRAINT in_house_app_software_categories_ibfk_1 FOREIGN KEY (in_house_app_id) REFERENCES in_house_apps (id) ON DELETE CASCADE,
  CONSTRAINT in_house_app_software_categories_ibfk_2 FOREIGN KEY (software_category_id) REFERENCES software_categories (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
	`

	if _, err := tx.Exec(createLabelMappingTableStmt); err != nil {
		return fmt.Errorf("create in_house_app_software_categories table: %w", err)
	}
	return nil
}

func Down_20251110172137(tx *sql.Tx) error {
	return nil
}
