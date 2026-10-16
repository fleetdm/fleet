package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20261002165514, Down_20261002165514)
}

// A queued or in-flight install now blocks deleting its installer instead of
// having its reference nulled out, which left the install unresolvable.
func Up_20261002165514(tx *sql.Tx) (err error) {
	const (
		fk    = "fk_software_install_upcoming_activities_software_installer_id"
		tmpFK = "fk_siua_software_installer_id_tmp"
	)

	// With FK checks on, MySQL can only add the constraint by copying the table and blocking writes. Existing
	// rows already satisfy it, since the constraint being replaced enforces the same reference.
	if _, err := tx.Exec(`SET FOREIGN_KEY_CHECKS = 0`); err != nil {
		return fmt.Errorf("disabling foreign key checks: %w", err)
	}
	defer func() {
		if _, execErr := tx.Exec(`SET FOREIGN_KEY_CHECKS = 1`); execErr != nil && err == nil {
			err = fmt.Errorf("re-enabling foreign key checks: %w", execErr)
		}
	}()

	// Each statement drops one constraint and adds its replacement atomically, so concurrent writes are never
	// unconstrained. MySQL rejects reusing the name within one statement, hence the temporary name. No ON
	// DELETE clause, so MySQL applies RESTRICT. ON UPDATE CASCADE stays for installer id rewrites.
	swap := func(from, to string) error {
		_, err := tx.Exec(`ALTER TABLE software_install_upcoming_activities DROP FOREIGN KEY ` + from + `,
			ADD CONSTRAINT ` + to + ` FOREIGN KEY (software_installer_id)
			REFERENCES software_installers (id) ON UPDATE CASCADE`)
		return err
	}
	// DDL isn't transactional; a retry after the first swap resumes at the second.
	if !fkExists(tx, "software_install_upcoming_activities", tmpFK) {
		if err := swap(fk, tmpFK); err != nil {
			return fmt.Errorf("swapping software installer fk to restricted temporary fk: %w", err)
		}
	}
	if err := swap(tmpFK, fk); err != nil {
		return fmt.Errorf("swapping restricted temporary fk back to software installer fk: %w", err)
	}
	return nil
}

func Down_20261002165514(tx *sql.Tx) error {
	return nil
}
