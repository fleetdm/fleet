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
	const fk = "fk_software_install_upcoming_activities_software_installer_id"

	// DDL isn't transactional, so guard the drop to keep a failed run retryable.
	if fkExists(tx, "software_install_upcoming_activities", fk) {
		if _, err := tx.Exec(`ALTER TABLE software_install_upcoming_activities DROP CONSTRAINT ` + fk); err != nil {
			return fmt.Errorf("dropping software installer fk: %w", err)
		}
	}

	// With FK checks on, MySQL can only add the constraint by copying the table and blocking writes. Existing
	// rows already satisfy it, since the dropped constraint enforced the same reference.
	if _, err := tx.Exec(`SET FOREIGN_KEY_CHECKS = 0`); err != nil {
		return fmt.Errorf("disabling foreign key checks: %w", err)
	}
	defer func() {
		if _, execErr := tx.Exec(`SET FOREIGN_KEY_CHECKS = 1`); execErr != nil && err == nil {
			err = fmt.Errorf("re-enabling foreign key checks: %w", execErr)
		}
	}()

	// No ON DELETE clause, so MySQL applies RESTRICT. ON UPDATE CASCADE stays
	// for installer id rewrites.
	if _, err := tx.Exec(`ALTER TABLE software_install_upcoming_activities
		ADD CONSTRAINT ` + fk + ` FOREIGN KEY (software_installer_id)
		REFERENCES software_installers (id) ON UPDATE CASCADE`); err != nil {
		return fmt.Errorf("adding restricted software installer fk: %w", err)
	}
	return nil
}

func Down_20261002165514(tx *sql.Tx) error {
	return nil
}
