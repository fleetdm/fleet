package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20250813205039, Down_20250813205039)
}

func Up_20250813205039(tx *sql.Tx) error {
	if _, err := tx.Exec(`
ALTER TABLE software_titles
	ADD COLUMN is_kernel TINYINT(1) NOT NULL DEFAULT '0'`); err != nil {
		return fmt.Errorf("failed to add software_titles.is_kernel column: %w", err)
	}

	// Backfill existing software titles
	if _, err := tx.Exec(`
UPDATE software_titles
SET is_kernel =
	-- Debian/Ubuntu
	CASE WHEN name REGEXP '^linux-image-[[:digit:]]+\.[[:digit:]]+\.[[:digit:]]+-[[:digit:]]+-[[:alnum:]]+' THEN
		1
	-- Amazon Linux
	WHEN name = 'kernel' THEN
		1
	-- RHEL
	WHEN name = 'kernel-core' THEN
		1
	ELSE
		0
	END
WHERE source IN ('rpm_packages', 'deb_packages')
	`); err != nil {
		return fmt.Errorf("failed to backfill software_titles.is_kernel column: %w", err)
	}

	if _, err := tx.Exec(`
CREATE TABLE kernel_host_counts (
  id int unsigned NOT NULL AUTO_INCREMENT,
  software_title_id int unsigned DEFAULT NULL,
  software_id int unsigned DEFAULT NULL,
  os_version_id int unsigned DEFAULT NULL,
  hosts_count int unsigned NOT NULL,
  team_id int unsigned NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY idx_kernels_unique_mapping (os_version_id,team_id,software_id),
  FOREIGN KEY (software_title_id) REFERENCES software_titles (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`); err != nil {
		return fmt.Errorf("failed to create kernel_host_counts table: %w", err)
	}

	return nil
}

func Down_20250813205039(tx *sql.Tx) error {
	return nil
}
