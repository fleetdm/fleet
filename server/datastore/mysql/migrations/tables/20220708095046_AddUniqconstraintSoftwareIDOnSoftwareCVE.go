package tables

import (
	"database/sql"
	"time"

	"github.com/fleetdm/fleet/v4/server"
	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20220708095046, Down_20220708095046)
}

func removeDups(tx *sql.Tx) error {
	const deleteStmt = `
delete sc
from
  software_cve sc
  join (
    select
      max(id) as max_id,
      cve,
      software_id
    from
      software_cve
    group by
      cve,
      software_id
    having
      count(*) > 1
  ) sc2 on sc2.cve = sc.cve AND sc2.software_id = sc.software_id
where
  sc.id < sc2.max_id;`

	if _, err := tx.Exec(deleteStmt); err != nil {
		return errors.Wrap(err, "removing duplicated rows")
	}

	return nil
}

func addUniqConstraint(tx *sql.Tx) error {
	_, err := tx.Exec(`
	ALTER TABLE software_cve ADD CONSTRAINT unq_software_id_cve UNIQUE (software_id, cve), ALGORITHM=INPLACE, LOCK=NONE;
`)
	if err != nil {
		return errors.Wrapf(err, "adding unique constraint to software_id on software_cve")
	}
	return nil
}

func acquireLock(tx *sql.Tx, identifier string) (bool, error) {
	logger.Info.Println("Trying to acquire lock...")
	name := "vulnerabilities"

	_, err := tx.Exec(
		`DELETE FROM locks WHERE expires_at < CURRENT_TIMESTAMP and name = ?`,
		name,
	)
	if err != nil {
		return false, errors.Wrapf(err, "trying to acquire lock")
	}

	r, err := tx.Exec(
		`INSERT IGNORE INTO locks (name, owner, expires_at) VALUES (?, ?, ?)`,
		name, identifier, time.Now().Add(30*time.Minute),
	)
	if err != nil {
		return false, errors.Wrapf(err, "trying to acquire lock")
	}
	rowsAffected, err := r.RowsAffected()
	if err != nil {
		return false, errors.Wrapf(err, "trying to acquire lock")
	}
	if rowsAffected > 0 {
		logger.Info.Println("Lock acquired...")
		return true, nil
	}

	return false, nil
}

func releaseLock(tx *sql.Tx, identifier string) error {
	if _, err := tx.Exec(`DELETE FROM locks WHERE name = ? and owner = ?`, "vulnerabilities", identifier); err != nil {
		return errors.Wrapf(err, "trying to release lock")
	}
	return nil
}

func Up_20220708095046(tx *sql.Tx) error {
	// Since we will be adding a uniqueness constrain on (software_id, cve) - we need to remove any
	// possible duplicates. Also because there's a chance we remove the duplicate rows before adding
	// the constraint and new duplicates get generated in between, we need to try to acquire the
	// vulnerability lock. In case the lock can't be acquired a warning is issued and the migration
	// will proceed without it.
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

	if err := removeDups(tx); err != nil {
		return err
	}

	if err := addUniqConstraint(tx); err != nil {
		return err
	}

	if err := releaseLock(tx, identifier); err != nil {
		return err
	}

	return nil
}

func Down_20220708095046(tx *sql.Tx) error {
	return nil
}
