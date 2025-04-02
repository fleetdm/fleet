package tables

import (
	"database/sql"
	"fmt"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20220704101843, Down_20220704101843)
}

func Up_20220704101843(tx *sql.Tx) error {
	if !columnExists(tx, "software_cve", "software_id") {

		_, err := tx.Exec(`
	ALTER TABLE software_cve ADD COLUMN software_id bigint(20) UNSIGNED NULL, ALGORITHM=INPLACE, LOCK=NONE;
`)
		if err != nil {
			return errors.Wrapf(err, "adding software_id to software_cve")
		}

	}

	var minVal int
	var maxVal int

	const selectStmt = `
SELECT COALESCE(MIN(cve.id), 0) AS min_id, COALESCE(MAX(cve.id), 0) as max_id 
FROM software_cve AS cve
WHERE cve.software_id IS NULL;`
	if err := tx.QueryRow(selectStmt).Scan(&minVal, &maxVal); err != nil {
		return errors.Wrap(err, "selecting min,max id")
	}

	// Update in batches
	const batchSize = 500
	const updateStmt = `
UPDATE software_cve AS cve 
INNER JOIN software_cpe AS cpe ON cve.cpe_id = cpe.id
SET cve.software_id = cpe.software_id 
WHERE cve.software_id IS NULL AND cve.id >= ? AND cve.id < ?;`

	if minVal != 0 || maxVal != 0 {
		fmt.Printf("Updating aprox %d records... \n", maxVal-minVal)
	}

	start := minVal
	for {
		end := start + batchSize
		if end >= maxVal {
			end = maxVal + 1
		}

		_, err := tx.Exec(updateStmt, start, end)
		if err != nil {
			return errors.Wrapf(err, "updating software_cve")
		}

		start += batchSize
		if start >= maxVal {
			break
		}
	}

	const indexStmt = `
ALTER TABLE software_cve ADD INDEX software_cve_software_id (software_id), ALGORITHM=INPLACE, LOCK=NONE;`
	_, err := tx.Exec(indexStmt)
	if err != nil {
		return errors.Wrapf(err, "adding index to software_id on software_cve table")
	}

	return nil
}

func Down_20220704101843(tx *sql.Tx) error {
	return nil
}
