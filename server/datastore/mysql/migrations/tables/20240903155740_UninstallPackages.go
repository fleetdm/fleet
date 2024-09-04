package tables

import (
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/reflectx"
)

func init() {
	MigrationClient.AddMigration(Up_20240903155740, Down_20240903155740)
}

func Up_20240903155740(tx *sql.Tx) error {
	if _, err := tx.Exec(`
ALTER TABLE software_installers 
ADD COLUMN package_ids TEXT COLLATE utf8mb4_unicode_ci NOT NULL,
ADD COLUMN uninstall_script_content_id int unsigned NOT NULL
		`); err != nil {
		return fmt.Errorf("failed to add package_ids to software_installers: %w", err)
	}

	txx := sqlx.Tx{Tx: tx, Mapper: reflectx.NewMapperFunc("db", sqlx.NameMapper)}

	// Add dummy uninstall scripts -- these will be updated by a cron job
	linuxScriptID, err := getOrInsertScript(txx, "exit 1")
	if err != nil {
		return err
	}
	windowsScriptID, err := getOrInsertScript(txx, "Exit 1")
	if err != nil {
		return err
	}

	// Update software installers with the scripts
	if _, err := tx.Exec(`UPDATE software_installers SET uninstall_script_content_id = ? WHERE platform IN ('linux', 'darwin')`,
		linuxScriptID); err != nil {
		return fmt.Errorf("failed to update software installers: %w", err)
	}
	if _, err := tx.Exec(`UPDATE software_installers SET uninstall_script_content_id = ? WHERE platform IN ('windows')`,
		windowsScriptID); err != nil {
		return fmt.Errorf("failed to update software installers: %w", err)
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

	return nil
}

func getOrInsertScript(txx sqlx.Tx, script string) (int64, error) {
	var ids []int64
	// check is such script already exists
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

func Down_20240903155740(_ *sql.Tx) error {
	return nil
}
