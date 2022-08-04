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

	var min int
	var max int

	const selectStmt = `
SELECT COALESCE(MIN(scpe.id), 0) AS min_id, COALESCE(MAX(scpe.id), 0) as max_id 
FROM software_cpe AS scpe
WHERE scpe.cpe LIKE 'none:%';`
	if err := tx.QueryRow(selectStmt).Scan(&min, &max); err != nil {
		return errors.Wrap(err, "selecting min,max id")
	}

	// Remove in batches
	const batchSize = 500
	const deleteStmt = `
DELETE FROM software_cpe
WHERE cpe LIKE 'none:%' AND id >= ? AND id < ?;`

	if min == 0 && max == 0 {
		logger.Info.Println("Nothing to delete ...")
	} else {
		logger.Info.Printf("Deleting aprox %d records... \n", max-min)
	}

	start := min
	for {
		end := start + batchSize
		if end >= max {
			end = max + 1
		}

		_, err := tx.Exec(deleteStmt, start, end)
		if err != nil {
			return errors.Wrapf(err, "deleting dummy software_cpe entries")
		}

		start += batchSize
		if start >= max {
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
