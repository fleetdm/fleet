package tables

import (
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/reflectx"
)

func init() {
	MigrationClient.AddMigration(Up_20240905200000, Down_20240905200000)
}

const placeholderUninstallScript = "# This script will be automatically updated within the next hour\nexit 1"
const placeholderUninstallScriptWindows = "# This script will be automatically updated within the next hour\nExit 1"

func Up_20240905200000(tx *sql.Tx) error {
	if _, err := tx.Exec(`
ALTER TABLE software_installers 
ADD COLUMN package_ids TEXT COLLATE utf8mb4_unicode_ci NOT NULL,
ADD COLUMN extension VARCHAR(32) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
ADD COLUMN uninstall_script_content_id int unsigned NOT NULL,
ADD COLUMN updated_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
MODIFY COLUMN uploaded_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6)
		`); err != nil {
		return fmt.Errorf("failed to alter software_installers: %w", err)
	}

	txx := sqlx.Tx{Tx: tx, Mapper: reflectx.NewMapperFunc("db", sqlx.NameMapper)}

	// Add dummy uninstall scripts if needed -- these will be updated later by a cron job
	var result []int
	if err := txx.Select(&result, `SELECT 1 FROM software_installers WHERE platform IN ('linux', 'darwin')`); err != nil {
		return fmt.Errorf("failed to check software installers for linux or darwin: %w", err)
	}
	if len(result) > 0 {
		linuxScriptID, err := getOrInsertScript(txx, placeholderUninstallScript)
		if err != nil {
			return err
		}
		// Update software installers with the scripts
		if _, err := tx.Exec(`UPDATE software_installers SET uninstall_script_content_id = ? WHERE platform IN ('linux', 'darwin')`,
			linuxScriptID); err != nil {
			return fmt.Errorf("failed to update software installers: %w", err)
		}
	}

	if err := txx.Select(&result, `SELECT 1 FROM software_installers WHERE platform IN ('windows')`); err != nil {
		return fmt.Errorf("failed to check software installers for windows: %w", err)
	}
	if len(result) > 0 {
		windowsScriptID, err := getOrInsertScript(txx, placeholderUninstallScriptWindows)
		if err != nil {
			return err
		}
		// Update software installers with the scripts
		if _, err := tx.Exec(`UPDATE software_installers SET uninstall_script_content_id = ? WHERE platform IN ('windows')`,
			windowsScriptID); err != nil {
			return fmt.Errorf("failed to update windows software installers: %w", err)
		}
	}

	// Add best-guess installer extensions if needed -- these will be updated later by a cron job to file contents based types
	// Also set existing updated_at timestamps to uploaded_at since installers were previously immutable
	if _, err := tx.Exec(`UPDATE software_installers SET extension = SUBSTRING_INDEX(filename,'.',-1), updated_at = uploaded_at`); err != nil {
		return fmt.Errorf("failed to backfill best-guess installer extensions: %w", err)
	}

	// Add foreign key
	if _, err := tx.Exec(`
ALTER TABLE software_installers
ADD CONSTRAINT fk_uninstall_script_content_id 
	FOREIGN KEY (uninstall_script_content_id)
	REFERENCES script_contents(id)
	ON DELETE RESTRICT ON UPDATE CASCADE`); err != nil {
		return fmt.Errorf("failed to add foreign key to software_installers: %w", err)
	}

	if _, err := tx.Exec(`
ALTER TABLE host_software_installs
ADD COLUMN uninstall_script_output TEXT COLLATE utf8mb4_unicode_ci,
ADD COLUMN uninstall_script_exit_code INT DEFAULT NULL,
ADD COLUMN uninstall TINYINT UNSIGNED NOT NULL DEFAULT 0,
ADD COLUMN status ENUM('pending_install', 'failed_install', 'installed', 'pending_uninstall', 'failed_uninstall')
GENERATED ALWAYS AS (
CASE
	WHEN removed = 1 THEN NULL

	WHEN post_install_script_exit_code IS NOT NULL AND
		post_install_script_exit_code = 0 THEN 'installed'

	WHEN post_install_script_exit_code IS NOT NULL AND
		post_install_script_exit_code != 0 THEN 'failed_install'

	WHEN install_script_exit_code IS NOT NULL AND
		install_script_exit_code = 0 THEN 'installed'

	WHEN install_script_exit_code IS NOT NULL AND
		install_script_exit_code != 0 THEN 'failed_install'

	WHEN pre_install_query_output IS NOT NULL AND
		pre_install_query_output = '' THEN 'failed_install'

	WHEN host_id IS NOT NULL AND uninstall = 0 THEN 'pending_install'

	WHEN uninstall_script_exit_code IS NOT NULL AND
		uninstall_script_exit_code != 0 THEN 'failed_uninstall'

	WHEN uninstall_script_exit_code IS NOT NULL AND
		uninstall_script_exit_code = 0 THEN NULL -- available for install again

	WHEN host_id IS NOT NULL AND uninstall = 1 THEN 'pending_uninstall'

	ELSE NULL -- not installed from Fleet installer or successfully uninstalled
END
) STORED NULL,
MODIFY COLUMN created_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
MODIFY COLUMN updated_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
MODIFY COLUMN host_deleted_at TIMESTAMP(6) NULL DEFAULT NULL
		`); err != nil {
		return fmt.Errorf("failed to alter host_software_installs: %w", err)
	}

	return nil
}

func getOrInsertScript(txx sqlx.Tx, script string) (int64, error) {
	var ids []int64
	// check is script already exists
	csum := md5ChecksumScriptContent(script)
	if err := txx.Select(&ids, `SELECT id FROM script_contents WHERE md5_checksum = UNHEX(?)`, csum); err != nil {
		return 0, fmt.Errorf("failed to find script contents: %w", err)
	}
	var scriptID int64
	if len(ids) > 0 {
		scriptID = ids[0]
	} else {
		// create new script
		var result sql.Result
		var err error
		if result, err = txx.Exec(`INSERT INTO script_contents (md5_checksum, contents) VALUES (UNHEX(?), ?)`, csum,
			script); err != nil {
			return 0, fmt.Errorf("failed to insert script contents: %w", err)
		}
		scriptID, _ = result.LastInsertId()
	}
	return scriptID, nil
}

func Down_20240905200000(_ *sql.Tx) error {
	return nil
}
