package tables

import (
	"database/sql"
	"time"

	"github.com/fleetdm/fleet/v4/server"
	"github.com/pkg/errors"
)

var logger *Logger

func init() {
	MigrationClient.AddMigration(Up_20220708095046, Down_20220708095046)
	logger = NewLogger()
}

func removeDups(tx *sql.Tx) error {
	logger.Info.Println("Removing duplicates in the software_cve table")

	const selectStmt = `
SELECT software_id, cve, COUNT(1) 
FROM software_cve GROUP BY software_id, cve 
HAVING COUNT(1) > 1;`

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

		logger.Info.Printf("Found duplicated row software_id: %d, cve:%s\n", softwareID, cve)
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

	return nil
}

func addUniqConstraint(tx *sql.Tx) error {
	logger.Info.Println("Adding unique constraint on (cve, software_id) to software_cve table...")
	_, err := tx.Exec(`
	ALTER TABLE software_cve ADD CONSTRAINT unq_software_id_cve UNIQUE (software_id, cve), ALGORITHM=INPLACE, LOCK=NONE;
`)
	if err != nil {
		return errors.Wrapf(err, "adding unique constraint to software_id on software_cve")
	}
	logger.Info.Println("Done adding unique constraint on (cve, software_id) to software_cve table...")
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
	identifier, err := server.GenerateRandomText(64)
	if err != nil {
		logger.Warn.Println("Could not generate identifier for lock, might not be able to remove duplicates in a reliable way...")
	} else {
		locked, err := acquireLock(tx, identifier)
		if !locked || err != nil {
			logger.Warn.Println("Could not acquire lock, might not be able to remove duplicates in a reliable way...")
		} else {
			defer releaseLock(tx, identifier)
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
