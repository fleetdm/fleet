package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20220818101352, Down_20220818101352)
}

func Up_20220818101352(tx *sql.Tx) error {
	//-----------------
	// Add temp column.
	//-----------------
	if !columnExists(tx, "software", "vendor_wide") {
		if _, err := tx.Exec(
			`ALTER TABLE software ADD COLUMN vendor_wide varchar(114) DEFAULT '' NOT NULL, ALGORITHM=INPLACE, LOCK=NONE`); err != nil {
			return errors.Wrapf(err, "creating temp column for vendor")
		}
	}

	//------------------
	// Perform update
	//------------------
	const updateStmt = `UPDATE software SET vendor_wide = vendor WHERE vendor <> ''`
	_, err := tx.Exec(updateStmt)
	if err != nil {
		return errors.Wrapf(err, "updating temp vendor column")
	}

	//---------------------
	// Add uniq constraint
	//---------------------
	if _, err := tx.Exec(
		"ALTER TABLE software ADD constraint unq_name UNIQUE (name, version, source, `release`, vendor_wide, arch)"); err != nil {
		return errors.Wrapf(err, "adding new uniquess constraint")
	}

	//----------------
	// Drop old index
	//----------------
	if _, err := tx.Exec(`ALTER TABLE software DROP KEY name`); err != nil {
		return errors.Wrapf(err, "dropping old index")
	}

	//------------------
	// Rename old column
	//------------------
	if _, err := tx.Exec(`ALTER TABLE software CHANGE vendor vendor_old varchar(32) DEFAULT '' NOT NULL, ALGORITHM=INPLACE, LOCK=NONE`); err != nil {
		return errors.Wrapf(err, "renaming old column")
	}

	// ---------------
	// Rename new column
	// ---------------
	if _, err := tx.Exec(
		`ALTER TABLE software CHANGE vendor_wide vendor varchar(114) DEFAULT '' NOT NULL, ALGORITHM=INPLACE, LOCK=NONE`); err != nil {
		return errors.Wrapf(err, "renaming new column")
	}

	return nil
}

func Down_20220818101352(tx *sql.Tx) error {
	return nil
}
