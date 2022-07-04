package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20220704101843, Down_20220704101843)
}

func Up_20220704101843(tx *sql.Tx) error {
	var min int
	var max int

	const selectStmt = `
SELECT MIN(cve.id) AS min_id, MAX(cve.id) as max_id 
FROM software_cve AS cve
WHERE cve.software_id IS NULL;`
	if err := tx.QueryRow(selectStmt).Scan(&min, &max); err != nil {
		return errors.Wrap(err, "selecting min,max id")
	}

	// Update in batches
	const batchSize = 500
	const updateStmt = `
UPDATE software_cve AS cve 
INNER JOIN software_cpe AS cpe ON cve.cpe_id = cpe.id
SET cve.software_id = cpe.software_id 
WHERE cve.software_id IS NULL AND cve.id >= ? AND cve.id < ?;`
	start := min
	for {
		end := start + batchSize
		if end >= max {
			end = max + 1
		}

		_, err := tx.Exec(updateStmt, start, end)
		if err != nil {
			return errors.Wrapf(err, "updating software_cve")
		}

		start += batchSize
		if start >= max {
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
