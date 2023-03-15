package tables

import (
	"database/sql"

	"github.com/fleetdm/fleet/v4/server"
	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20230315163954, Down_20230315163954)
}

func Up_20230315163954(tx *sql.Tx) error {
	// Since we will be adding a uniqueness constrain on (software_id) on the  - we need to remove any
	// possible duplicates. Also because there's a chance we remove the duplicate rows before adding
	// the constraint and new duplicates get generated in between, we need to try to acquire the
	// vulnerability lock. In case the lock can't be acquired a warning is issued and the migration
	// will proceed without it.

	removeDuplicatedSoftwareCPEs := func(tx *sql.Tx) error {
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
WHERE sc.id < sc2.max_id;`

		if _, err := tx.Exec(deleteStmt); err != nil {
			return errors.Wrap(err, "removing duplicated rows")
		}

		return nil
	}

	addConstraint := func(tx *sql.Tx) error {
		_, err := tx.Exec(`
	ALTER TABLE software_cpe ADD CONSTRAINT unq_software_id UNIQUE (software_id), ALGORITHM=INPLACE, LOCK=NONE;
`)
		if err != nil {
			return errors.Wrapf(err, "adding unique constraint to software_id on software_cpe")
		}
		return nil
	}

	identifier, err := server.GenerateRandomText(64)

	if err != nil {
		logger.Warn.Println("Could not generate identifier for lock, might not be able to remove duplicates in a reliable way...")
	} else {
		locked, err := acquireLock(tx, identifier)
		if !locked || err != nil {
			logger.Warn.Println("Could not acquire lock, might not be able to remove duplicates in a reliable way...")
		} else {
			defer releaseLock(tx, identifier) //nolint:errcheck
		}
	}

	if err := removeDuplicatedSoftwareCPEs(tx); err != nil {
		return err
	}

	if err := addConstraint(tx); err != nil {
		return err
	}

	if err := releaseLock(tx, identifier); err != nil {
		return err
	}

	return nil
}

func Down_20230315163954(tx *sql.Tx) error {
	return nil
}
