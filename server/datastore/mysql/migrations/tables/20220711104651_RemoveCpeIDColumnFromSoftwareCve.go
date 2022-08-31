package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20220711104651, Down_20220711104651)
}

func Up_20220711104651(tx *sql.Tx) error {
	logger.Info.Println("Deleting dummy software_cpe entries...")
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
	logger.Info.Println("Done deleting dummy cpe_id entries...")

	logger.Info.Println("Removing cpe_id from software_cve...")
	const removeFkStmt = `
ALTER TABLE software_cve DROP FOREIGN KEY software_cve_ibfk_1, ALGORITHM=INPLACE, LOCK=NONE; 
`
	_, err := tx.Exec(removeFkStmt)
	if err != nil {
		return errors.Wrapf(err, "removing cpe_id FK from software_cve")
	}

	const removeUnqStmt = `
ALTER TABLE software_cve DROP INDEX unique_cpe_cve, ALGORITHM=INPLACE, LOCK=NONE;
`
	_, err = tx.Exec(removeUnqStmt)
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

	logger.Info.Println("Done removing cpe_id from software_cve...")

	return nil
}

func Down_20220711104651(tx *sql.Tx) error {
	return nil
}
