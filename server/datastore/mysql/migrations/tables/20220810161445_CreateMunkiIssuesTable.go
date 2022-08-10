package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20220810161445, Down_20220810161445)
}

func Up_20220810161445(tx *sql.Tx) error {
	// name is actually the error/warning message - 255 ought to be enough, based
	// on the example error messages the longest is only ~80. If we need a larger
	// column, we can always add a column for the hash of the name and set the
	// unique index on the hash instead of the full message. But for now, 255 seems
	// both sufficiently large to store the messages, and sufficiently small to
	// be used as-is for the unique index.
	//
	// issue_type is "warning" or "error", and is not called "type" to avoid using
	// a keyword that requires quoting.
	_, err := tx.Exec(`
	CREATE TABLE munki_issues (
		id         INT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
		name       VARCHAR(255) NOT NULL,
		issue_type VARCHAR(10) NOT NULL,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

		UNIQUE KEY idx_munki_issues_name (name)
	)`)
	if err != nil {
		return errors.Wrapf(err, "create munki_issues table")
	}
	return nil
}

func Down_20220810161445(tx *sql.Tx) error {
	return nil
}
