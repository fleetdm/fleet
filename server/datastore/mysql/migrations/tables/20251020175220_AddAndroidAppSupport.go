package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20251020175220, Down_20251020175220)
}

func Up_20251020175220(tx *sql.Tx) error {
	// 	_, err := tx.Exec(`
	// -- This table is the Android equivalent of the "vpp_apps" table.
	// CREATE TABLE android_apps (
	// 	id int(10) unsigned NOT NULL AUTO_INCREMENT PRIMARY KEY,

	// 	-- FK to the "software title" this app matches
	// 	title_id int(10) unsigned DEFAULT NULL,

	// 	application_id VARCHAR(255) NOT NULL DEFAULT '',
	// 	icon_url VARCHAR(255) NOT NULL DEFAULT '',
	// 	name VARCHAR(255) NOT NULL DEFAULT '',
	// 	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	// 	updated_at TIMESTAMP NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
	// 	UNIQUE KEY (application_id),
	// 	CONSTRAINT fk_android_apps_title
	// 	  FOREIGN KEY (title_id)
	// 	  REFERENCES software_titles (id)
	// 	  ON DELETE SET NULL
	// 	  ON UPDATE CASCADE
	// ) DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci`)
	// 	if err != nil {
	// 		return fmt.Errorf("failed to create table android_apps: %w", err)
	// 	}
	//
	// _, err := tx.Exec(`
	// 	ALTER TABLE vpp_apps
	// 	ADD COLUMN application_id VARCHAR(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT ''`)
	// if err != nil {
	// 	return fmt.Errorf("adding platform to host_vpp_software_installs: %w", err)
	// }

	return nil
}

func Down_20251020175220(tx *sql.Tx) error {
	return nil
}
