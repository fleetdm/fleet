package tables

import (
	"database/sql"
	"fmt"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20220708095046, Down_20220708095046)
}

func Up_20220708095046(tx *sql.Tx) error {
	fmt.Println("Removing duplicates in the software_cve table")

	const selectStmt = `
SELECT software_id, cve, COUNT(1) FROM software_cve GROUP BY software_id, cve HAVING COUNT(1) > 1;
`
	rows, err := tx.Query(selectStmt)
	if err != nil {
		return errors.Wrap(err, "selecting duplicates")
	}
	defer rows.Close()

	type criteria struct {
		softwareID uint
		cve        string
		count      uint
	}

	var criterias []criteria
	for rows.Next() {
		var softwareID uint
		var cve string
		var count uint

		if err = rows.Scan(&softwareID, &cve, &count); err != nil {
			return errors.Wrap(err, "scanning duplicate rows")
		}

		fmt.Printf("Found duplicated row software_id: %d, cve:%s\n", softwareID, cve)
		criterias = append(criterias, criteria{softwareID: softwareID, cve: cve, count: count})
	}

	if err := rows.Err(); err != nil {
		return errors.Wrap(err, "scanning duplicate rows")
	}
	rows.Close()

	for _, c := range criterias {
		if _, err := tx.Exec(
			`DELETE FROM software_cve WHERE software_id = ? AND cve = ? LIMIT ?`,
			c.softwareID, c.cve, c.count-1,
		); err != nil {
			return errors.Wrap(err, "removing duplicated row")
		}
	}

	fmt.Println("Adding unique constraint on (cve, software_id) to software_cve table...")
	_, err = tx.Exec(`
	ALTER TABLE software_cve ADD CONSTRAINT unq_software_id_cve UNIQUE (software_id, cve), ALGORITHM=INPLACE, LOCK=NONE;
`)
	if err != nil {
		return errors.Wrapf(err, "adding unique constraint to software_id on software_cve")
	}
	fmt.Println("Done Adding unique constraint on (cve, software_id) to software_cve table...")

	return nil
}

func Down_20220708095046(tx *sql.Tx) error {
	return nil
}
