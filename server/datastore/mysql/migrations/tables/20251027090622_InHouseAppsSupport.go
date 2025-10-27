package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20251027090622, Down_20251027090622)
}

func Up_20251027090622(tx *sql.Tx) error {
	createTableStmt := `
CREATE TABLE in_house_apps (
  id int unsigned NOT NULL AUTO_INCREMENT,
  title_id int unsigned DEFAULT NULL,
  team_id int unsigned DEFAULT NULL,
  global_or_team_id int unsigned NOT NULL DEFAULT '0',
  name VARCHAR(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  version VARCHAR(255) NOT NULL DEFAULT '',
  storage_id VARCHAR(64) COLLATE utf8mb4_unicode_ci NOT NULL,
  created_at timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  platform varchar(10) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  bundle_identifier VARCHAR(255) NOT NULL DEFAULT '',
  PRIMARY KEY (id),
  UNIQUE KEY (global_or_team_id,name,platform),
  CONSTRAINT fk_in_house_apps_title FOREIGN KEY (title_id) REFERENCES software_titles (id) ON DELETE SET NULL ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
	`
	if _, err := tx.Exec(createTableStmt); err != nil {
		return fmt.Errorf("create in_house_apps table: %w", err)
	}

	createLabelMappingTableStmt := `
CREATE TABLE in_house_app_labels (
  id int unsigned NOT NULL AUTO_INCREMENT,
  in_house_app_id int unsigned NOT NULL,
  label_id int unsigned NOT NULL,
  exclude tinyint(1) NOT NULL DEFAULT '0',
  created_at timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  updated_at timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  PRIMARY KEY (id),
  UNIQUE KEY id_in_house_app_labels_in_house_app_id_label_id (in_house_app_id,label_id),
  KEY label_id (label_id),
  CONSTRAINT in_house_app_labels_ibfk_1 FOREIGN KEY (in_house_app_id) REFERENCES in_house_apps (id) ON DELETE CASCADE,
  CONSTRAINT in_house_app_labels_ibfk_2 FOREIGN KEY (label_id) REFERENCES labels (id) ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
	`

	if _, err := tx.Exec(createLabelMappingTableStmt); err != nil {
		return fmt.Errorf("create in_house_app_labels table: %w", err)
	}

	return nil
}

func Down_20251027090622(tx *sql.Tx) error {
	return nil
}
