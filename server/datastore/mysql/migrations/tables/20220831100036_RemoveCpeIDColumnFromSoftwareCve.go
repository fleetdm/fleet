package tables

import (
	"database/sql"
	"fmt"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20220831100036, Down_20220831100036)
}

func Up_20220831100036(tx *sql.Tx) error {
	// Remove in batches
	const deleteStmt = `DELETE FROM software_cpe WHERE cpe LIKE 'none:%' LIMIT 10000`

	for {

		res, err := tx.Exec(deleteStmt)
		if err != nil {
			return errors.Wrapf(err, "deleting dummy software_cpe entries")
		}

		affected, err := res.RowsAffected()
		if err != nil {
			return errors.Wrapf(err, "deleting dummy software_cpe entries")
		}

		if affected == 0 {
			break
		}
	}

	// The name for the FK from software_cve to software_cpe changes depending on whether the user
	// is running MySQL or MariaDB.
	fkNames := []string{"software_cve_ibfk_1", "fk_software_cve_cpe_id"}
	for _, fkName := range fkNames {
		if fkExists(tx, "software_cve", fkName) {
			removeFkStmt := fmt.Sprintf(`
				ALTER TABLE software_cve DROP FOREIGN KEY %s, ALGORITHM=INPLACE, LOCK=NONE;
			`, fkName)
			_, err := tx.Exec(removeFkStmt)
			if err != nil {
				return errors.Wrapf(err, "removing cpe_id FK from software_cve")
			}
			break
		}
	}

	const removeUnqStmt = `
ALTER TABLE software_cve DROP INDEX unique_cpe_cve, ALGORITHM=INPLACE, LOCK=NONE;
`
	_, err := tx.Exec(removeUnqStmt)
	if err != nil {
		return errors.Wrapf(err, "removing uniq cpe_id constraint from software_cve")
	}

	const removeColStmt = `
ALTER TABLE software_cve DROP COLUMN cpe_id, ALGORITHM=INPLACE, LOCK=NONE;
`
	_, err = tx.Exec(removeColStmt)
	if err != nil {
		return errors.Wrapf(err, "removing cpe_id column from software_cve")
	}

	return nil
}

func Down_20220831100036(tx *sql.Tx) error {
	return nil
}
