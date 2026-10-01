package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20261001114949, Down_20261001114949)
}

// A queued or in-flight install now blocks deleting its installer instead of
// having its reference nulled out, which left the install unresolvable.
func Up_20261001114949(tx *sql.Tx) error {
	const fk = "fk_software_install_upcoming_activities_software_installer_id"

	// DDL isn't transactional, so guard the drop to keep a failed run retryable.
	if fkExists(tx, "software_install_upcoming_activities", fk) {
		if _, err := tx.Exec(`ALTER TABLE software_install_upcoming_activities DROP CONSTRAINT ` + fk); err != nil {
			return fmt.Errorf("dropping software installer fk: %w", err)
		}
	}

	// No ON DELETE clause, so MySQL applies RESTRICT. ON UPDATE CASCADE stays
	// for installer id rewrites.
	if _, err := tx.Exec(`ALTER TABLE software_install_upcoming_activities
		ADD CONSTRAINT ` + fk + ` FOREIGN KEY (software_installer_id)
		REFERENCES software_installers (id) ON UPDATE CASCADE`); err != nil {
		return fmt.Errorf("adding restricted software installer fk: %w", err)
	}
	return nil
}

func Down_20261001114949(tx *sql.Tx) error {
	return nil
}
