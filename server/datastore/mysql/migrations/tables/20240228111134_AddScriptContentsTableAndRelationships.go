package tables

import (
	"crypto/md5" //nolint:gosec
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/reflectx"
)

func init() {
	MigrationClient.AddMigration(Up_20240228111134, Down_20240228111134)
}

func Up_20240228111134(tx *sql.Tx) error {
	txx := sqlx.Tx{Tx: tx, Mapper: reflectx.NewMapperFunc("db", sqlx.NameMapper)}

	// at the time of this migration, we have two tables that deal with scripts:
	//   - host_script_results: stores both the contents of the script and
	//     eventually its result. Both anonymous and saved scripts get their
	//     contents stored here (for saved scripts, this is so that if the saved
	//     script is deleted we can still display the contents in the activities).
	//   - scripts: stores saved scripts with a name and content, which can then
	//     be executed on hosts (which would copy the script content to
	//     host_script_results as well as adding a FK to the saved script).
	//
	// This migration moves the script contents to a separate table, deduplicated
	// so that the same contents (whether it is via a saved script or an
	// anonymous one) are only stored once and both host_script_results and
	// scripts reference that entry.

	// Using md5 checksum stored in binary (so,
	// "UNHEX(md5-string-representation)" when storing) for efficient storage.
	// The choice of md5 despite it being broken is because:
	// 	 - we don't use it for anything critical, just deduplication of scripts
	//   - it's available in mysql; sha2 is also a possibility, but there's this
	//   note in mysql's documentation
	//   (https://dev.mysql.com/doc/refman/5.7/en/encryption-functions.html#function_sha2):
	//     > This function works only if MySQL has been configured with SSL support.
	//   and we need to support a wide variety of MySQL installations in
	//   the wild. (sha1 is also available without this constraint but also broken)
	//   - it's same as what we use elsewhere in the DB, e.g. mdm apple profiles
	createScriptContentsStmt := `
CREATE TABLE script_contents (
	id            INT(10) UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
	md5_checksum  BINARY(16) NOT NULL,
	contents      TEXT COLLATE utf8mb4_unicode_ci NOT NULL,
	created_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

	UNIQUE KEY idx_script_contents_md5_checksum (md5_checksum)
)`
	if _, err := txx.Exec(createScriptContentsStmt); err != nil {
		return fmt.Errorf("create table script_contents: %w", err)
	}

	insertScriptContentsStmt := `
	INSERT INTO
		script_contents (md5_checksum, contents)
	VALUES
		(UNHEX(?), ?)
	ON DUPLICATE KEY UPDATE
		created_at = created_at
`

	type scriptContent struct {
		ID             uint   `db:"id"`
		ScriptContents string `db:"script_contents"`
	}

	// load all contents found in host_script_results to next insert in
	// script_contents
	readHostScriptsStmt := `
	SELECT
		id,
		script_contents
	FROM
		host_script_results
`
	var hostScripts []scriptContent
	if err := txx.Select(&hostScripts, readHostScriptsStmt); err != nil {
		return fmt.Errorf("load host_script_results contents: %w", err)
	}

	// load all contents found in scripts to next insert in script_contents
	readScriptsStmt := `
	SELECT
		id,
		script_contents
	FROM
		scripts
`
	var savedScripts []scriptContent
	if err := txx.Select(&savedScripts, readScriptsStmt); err != nil {
		return fmt.Errorf("load saved scripts contents: %w", err)
	}

	// for every script content, md5-hash it and keep track of the hash
	// associated with the ids to update the host_script_results and scripts
	// tables with the new script_content_id later on.
	hostScriptsLookup := make(map[string][]uint, len(hostScripts))
	savedScriptsLookup := make(map[string][]uint, len(savedScripts))
	for _, s := range hostScripts {
		hexChecksum := md5ChecksumScriptContent(s.ScriptContents)
		hostScriptsLookup[hexChecksum] = append(hostScriptsLookup[hexChecksum], s.ID)
		if _, err := txx.Exec(insertScriptContentsStmt, hexChecksum, s.ScriptContents); err != nil {
			return fmt.Errorf("create script_contents from host_script_results: %w", err)
		}
	}
	for _, s := range savedScripts {
		hexChecksum := md5ChecksumScriptContent(s.ScriptContents)
		savedScriptsLookup[hexChecksum] = append(savedScriptsLookup[hexChecksum], s.ID)
		if _, err := txx.Exec(insertScriptContentsStmt, hexChecksum, s.ScriptContents); err != nil {
			return fmt.Errorf("create script_contents from saved scripts: %w", err)
		}
	}

	alterAddHostScriptsStmt := `
ALTER TABLE host_script_results
	ADD COLUMN script_content_id INT(10) UNSIGNED NULL,
	ADD FOREIGN KEY (script_content_id) REFERENCES script_contents (id) ON DELETE CASCADE`
	if _, err := tx.Exec(alterAddHostScriptsStmt); err != nil {
		return fmt.Errorf("alter add column to table host_script_results: %w", err)
	}

	alterAddScriptsStmt := `
ALTER TABLE scripts
	ADD COLUMN script_content_id INT(10) UNSIGNED NULL,
	ADD FOREIGN KEY (script_content_id) REFERENCES script_contents (id) ON DELETE CASCADE`
	if _, err := tx.Exec(alterAddScriptsStmt); err != nil {
		return fmt.Errorf("alter add column to table scripts: %w", err)
	}

	updateHostScriptsStmt := `
UPDATE
	host_script_results
SET
	script_content_id = (SELECT id FROM script_contents WHERE md5_checksum = UNHEX(?)),
	updated_at = updated_at
WHERE
	id = ?`

	// for saved scripts, the `updated_at` timestamp is used as the "uploaded_at"
	// information in the UI, so we ensure that it doesn't change with this
	// migration.
	updateSavedScriptsStmt := `
UPDATE
	scripts
SET
	script_content_id = (SELECT id FROM script_contents WHERE md5_checksum = UNHEX(?)),
	updated_at = updated_at
WHERE
	id = ?`

	// insert the associated script_content_id into host_script_results and
	// scripts
	for hexChecksum, ids := range hostScriptsLookup {
		for _, id := range ids {
			if _, err := txx.Exec(updateHostScriptsStmt, hexChecksum, id); err != nil {
				return fmt.Errorf("update host_script_results with script_content_id: %w", err)
			}
		}
	}
	for hexChecksum, ids := range savedScriptsLookup {
		for _, id := range ids {
			if _, err := txx.Exec(updateSavedScriptsStmt, hexChecksum, id); err != nil {
				return fmt.Errorf("update saved scripts with script_content_id: %w", err)
			}
		}
	}

	// TODO(mna): we cannot drop the "script_contents" column immediately from
	// the host_script_results and scripts tables because that would break the
	// current code, we need to wait for the feature to be fully implemented.
	// There's no harm in leaving it in there unused for now, as stored scripts
	// were previously smallish.

	return nil
}

func md5ChecksumScriptContent(s string) string {
	rawChecksum := md5.Sum([]byte(s)) //nolint:gosec
	return strings.ToUpper(hex.EncodeToString(rawChecksum[:]))
}

func Down_20240228111134(tx *sql.Tx) error {
	return nil
}
