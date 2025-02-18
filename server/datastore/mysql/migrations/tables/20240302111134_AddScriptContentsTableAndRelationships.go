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
	MigrationClient.AddMigration(Up_20240302111134, Down_20240302111134)
}

func Up_20240302111134(tx *sql.Tx) error {
	txx := &sqlx.Tx{Tx: tx, Mapper: reflectx.NewMapperFunc("db", sqlx.NameMapper)}

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
	// Note: MEDIUMTEXT can handle up to ~16 million bytes, so it should be plenty for our use case.
	createScriptContentsStmt := `
CREATE TABLE script_contents (
	id            INT(10) UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
	md5_checksum  BINARY(16) NOT NULL,
	contents      MEDIUMTEXT COLLATE utf8mb4_unicode_ci NOT NULL,
	created_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

	UNIQUE KEY idx_script_contents_md5_checksum (md5_checksum)
)`
	if _, err := txx.Exec(createScriptContentsStmt); err != nil {
		return fmt.Errorf("create table script_contents: %w", err)
	}

	// map tracking the script_contents id for a given md5 checksum
	scriptContentsIDLookup := make(map[string]uint)
	// map tracking the host_script_results and scripts ids for a given md5
	// checksum
	hostScriptsLookup := make(map[string][]uint)
	savedScriptsLookup := make(map[string][]uint)

	// load all contents found in host_script_results to insert in
	// script_contents
	readHostScriptsStmt := `
	SELECT
		id,
		script_contents
	FROM
		host_script_results
`
	if err := createScriptContentsEntries(txx, "host_script_results", readHostScriptsStmt, scriptContentsIDLookup, hostScriptsLookup); err != nil {
		return err
	}

	// load all contents found in scripts to insert in script_contents
	readScriptsStmt := `
	SELECT
		id,
		script_contents
	FROM
		scripts
`
	if err := createScriptContentsEntries(txx, "saved scripts", readScriptsStmt, scriptContentsIDLookup, savedScriptsLookup); err != nil {
		return err
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
	script_content_id = ?,
	updated_at = updated_at
WHERE
	id IN (?)`

	// for saved scripts, the `updated_at` timestamp is used as the "uploaded_at"
	// information in the UI, so we ensure that it doesn't change with this
	// migration.
	updateSavedScriptsStmt := `
UPDATE
	scripts
SET
	script_content_id = ?,
	updated_at = updated_at
WHERE
	id IN (?)`

	// insert the associated script_content_id into host_script_results and
	// scripts
	for hexChecksum, ids := range hostScriptsLookup {
		scID := scriptContentsIDLookup[hexChecksum]
		if err := updateScriptContentIDInBatches(txx, "host_script_results", updateHostScriptsStmt, scID, ids); err != nil {
			return err
		}
	}
	for hexChecksum, ids := range savedScriptsLookup {
		scID := scriptContentsIDLookup[hexChecksum]
		if err := updateScriptContentIDInBatches(txx, "saved scripts", updateSavedScriptsStmt, scID, ids); err != nil {
			return err
		}
	}

	// NOTE: we cannot drop the "script_contents" column immediately from the
	// host_script_results and scripts tables because that would break the
	// current code, we need to wait for the feature to be fully implemented.
	// There's no harm in leaving it in there unused for now, as stored scripts
	// were previously smallish.

	return nil
}

var testBatchSize int

func updateScriptContentIDInBatches(txx *sqlx.Tx, stmtTable, stmt string, scriptContentID uint, allIDs []uint) error {
	const maxBatchSize = 10000

	batchSize := maxBatchSize
	if testBatchSize > 0 {
		// to allow override for tests
		batchSize = testBatchSize
	}

	var startIx int
	for startIx < len(allIDs) {
		batchIDs := allIDs[startIx:]
		if len(batchIDs) > batchSize {
			batchIDs = batchIDs[startIx : startIx+batchSize]
		}
		startIx += len(batchIDs)

		stmt, args, err := sqlx.In(stmt, scriptContentID, batchIDs)
		if err != nil {
			return fmt.Errorf("prepare statement to update %s with script_content_id: %w", stmtTable, err)
		}
		if _, err := txx.Exec(stmt, args...); err != nil {
			return fmt.Errorf("update %s with script_content_id: %w", stmtTable, err)
		}
	}
	return nil
}

func createScriptContentsEntries(txx *sqlx.Tx, stmtTable, stmt string, scriptContentsIDLookup map[string]uint, contentHashToStmtTableIDs map[string][]uint) error {
	type scriptContent struct {
		ID             uint   `db:"id"`
		ScriptContents string `db:"script_contents"`
	}

	// using id = LAST_INSERT_ID(id) to get the id of the row that was  updated
	// in case of a duplicate key.
	insertScriptContentsStmt := `
	INSERT INTO
		script_contents (md5_checksum, contents)
	VALUES
		(UNHEX(?), ?)
	ON DUPLICATE KEY UPDATE
		id = LAST_INSERT_ID(id)
`

	// we cannot use a txx.Query and iterate over the rows, because we would need
	// another connection to be able to insert into script_contents while
	// iterating. So instead, we load the scripts in reasonably-sized batches.
	const batchSize = 1000 // at most ~10MB of script contents
	var lastID uint
	stmt += ` WHERE id > ? ORDER BY id LIMIT ?`

	for {
		var scriptContents []scriptContent
		if err := txx.Select(&scriptContents, stmt, lastID, batchSize); err != nil {
			return fmt.Errorf("load %s contents: %w", stmtTable, err)
		}

		if len(scriptContents) == 0 {
			return nil
		}

		for _, s := range scriptContents {
			lastID = s.ID

			hexChecksum := md5ChecksumScriptContent(s.ScriptContents)
			contentHashToStmtTableIDs[hexChecksum] = append(contentHashToStmtTableIDs[hexChecksum], s.ID)
			if id := scriptContentsIDLookup[hexChecksum]; id == 0 {
				// insert the script content into the script_contents table, we don't
				// have its id yet
				res, err := txx.Exec(insertScriptContentsStmt, hexChecksum, s.ScriptContents)
				if err != nil {
					return fmt.Errorf("create script_contents from %s: %w", stmtTable, err)
				}
				id, _ := res.LastInsertId()
				scriptContentsIDLookup[hexChecksum] = uint(id) //nolint:gosec // dismiss G115
			}
		}
	}
}

func md5ChecksumScriptContent(s string) string {
	rawChecksum := md5.Sum([]byte(s)) //nolint:gosec
	return strings.ToUpper(hex.EncodeToString(rawChecksum[:]))
}

func Down_20240302111134(tx *sql.Tx) error {
	return nil
}
