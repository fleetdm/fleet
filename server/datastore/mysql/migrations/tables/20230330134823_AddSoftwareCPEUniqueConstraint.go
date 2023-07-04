package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20230330134823, Down_20230330134823)
}

// Since we will be adding a uniqueness constrain on (software_id) on the software_cpe table - we need to remove any
// possible duplicates.
func _20230329161600_remove_duplicates(tx *sql.Tx) error {
	const deleteStmt = `
DELETE sc
FROM software_cpe sc
	INNER JOIN (
		SELECT
			software_id,
			MAX(id) as max_id
		FROM software_cpe
		GROUP BY software_id
		HAVING COUNT(*) > 1
	) sc2 ON sc2.software_id = sc.software_id
	WHERE sc.id < sc2.max_id;
`

	if _, err := tx.Exec(deleteStmt); err != nil {
		return errors.Wrap(err, "removing duplicated rows")
	}

	return nil
}

func _20230329161600_add_unq_constraint(tx *sql.Tx) error {
	_, err := tx.Exec(`
	ALTER TABLE software_cpe ADD CONSTRAINT unq_software_id UNIQUE (software_id), ALGORITHM=INPLACE, LOCK=NONE;
`)
	if err != nil {
		return errors.Wrapf(err, "adding unique constraint to software_id on software_cpe")
	}
	return nil
}

func Up_20230330134823(tx *sql.Tx) error {
	if err := _20230329161600_remove_duplicates(tx); err != nil {
		return err
	}

	if err := _20230329161600_add_unq_constraint(tx); err != nil {
		return err
	}

	return nil
}

func Down_20230330134823(tx *sql.Tx) error {
	return nil
}
