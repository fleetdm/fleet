package tables

import (
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
)

func init() {
	MigrationClient.AddMigration(Up_20250320200000, Down_20250320200000)
}

func Up_20250320200000(tx *sql.Tx) error {
	// Clean up Fleet Library App associated scripts before we drop the columns on the table
	_, err := tx.Exec(`DELETE FROM
  script_contents
WHERE
  NOT EXISTS (
    SELECT 1 FROM host_script_results WHERE script_content_id = script_contents.id)
  AND NOT EXISTS (
    SELECT 1 FROM scripts WHERE script_content_id = script_contents.id)
  AND NOT EXISTS (
    SELECT 1 FROM software_installers si
    WHERE script_contents.id IN (si.install_script_content_id, si.post_install_script_content_id, si.uninstall_script_content_id)
  )
  AND NOT EXISTS (
    SELECT 1 FROM fleet_library_apps fla
			WHERE script_contents.id IN (fla.install_script_content_id, fla.uninstall_script_content_id)
  )
  AND NOT EXISTS (
    SELECT 1 FROM setup_experience_scripts WHERE script_content_id = script_contents.id
	)
  AND NOT EXISTS (
    SELECT 1 FROM script_upcoming_activities WHERE script_content_id = script_contents.id
	)`)
	if err != nil {
		return fmt.Errorf("failed to clean up unused scripts: %w", err)
	}

	_, err = tx.Exec(`
ALTER TABLE software_installers
	CHANGE COLUMN fleet_library_app_id fleet_maintained_app_id INT unsigned DEFAULT NULL
`)
	if err != nil {
		return fmt.Errorf("failed to rename fleet_library_app_id column: %w", err)
	}

	_, err = tx.Exec(`RENAME TABLE fleet_library_apps TO fleet_maintained_apps`)
	if err != nil {
		return fmt.Errorf("failed to rename fleet_library_apps: %w", err)
	}

	_, err = tx.Exec(`UPDATE software_installers si
		JOIN fleet_maintained_apps fma ON fma.id = si.fleet_maintained_app_id
		SET fleet_maintained_app_id = NULL
	    WHERE fma.install_script_content_id != si.install_script_content_id
	    	OR fma.uninstall_script_content_id != si.uninstall_script_content_id`)
	if err != nil {
		return fmt.Errorf("failed to unlink diverged Fleet-maintained apps: %w", err)
	}

	_, err = tx.Exec(`
ALTER TABLE fleet_maintained_apps
    DROP CONSTRAINT fk_fleet_library_apps_install_script_content,
    DROP CONSTRAINT fk_fleet_library_apps_uninstall_script_content
`)
	if err != nil {
		return fmt.Errorf("failed to drop obsolete indexed from fleet_maintained_apps: %w", err)
	}

	_, err = tx.Exec(`
ALTER TABLE fleet_maintained_apps
	DROP COLUMN version,
	DROP COLUMN installer_url,
	DROP COLUMN sha256,
	DROP COLUMN install_script_content_id,
	DROP COLUMN uninstall_script_content_id,
	CHANGE COLUMN bundle_identifier unique_identifier VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
 	CHANGE COLUMN token slug VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL
`)
	if err != nil {
		return fmt.Errorf("failed to alter fleet_maintained_apps: %w", err)
	}

	_, err = tx.Exec(`UPDATE fleet_maintained_apps SET slug = concat(slug, '/', platform)`)
	if err != nil {
		return fmt.Errorf("failed to rename FMA slugs: %w", err)
	}

	txx := sqlx.Tx{Tx: tx}

	var slugs []string
	err = txx.Select(&slugs, `SELECT slug FROM fleet_maintained_apps WHERE slug in ('zoom/darwin', 'zoom-for-it-admins/darwin')`)
	if err != nil {
		return fmt.Errorf("checking Zoom apps: %w", err)
	}

	// clear old Zoom FMA before swapping in new one
	if len(slugs) > 1 || (len(slugs) == 1 && slugs[0] == "zoom/darwin") {
		_, err = tx.Exec(`DELETE FROM fleet_maintained_apps WHERE slug = 'zoom/darwin'`)
		if err != nil {
			return fmt.Errorf("failed to remove duplicate Zoom FMA: %w", err)
		}
	}

	_, err = tx.Exec(`UPDATE fleet_maintained_apps SET slug = 'zoom/darwin', name = 'Zoom' WHERE slug = 'zoom-for-it-admins/darwin'`)
	if err != nil {
		return fmt.Errorf("failed to rename Zoom FMA: %w", err)
	}

	return nil
}

func Down_20250320200000(tx *sql.Tx) error {
	return nil
}
