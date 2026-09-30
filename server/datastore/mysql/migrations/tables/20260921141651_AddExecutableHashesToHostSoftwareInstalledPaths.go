package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260921141651, Down_20260921141651)
}

func Up_20260921141651(tx *sql.Tx) error {
	if columnExists(tx, "host_software_installed_paths", "executable_hashes") {
		return nil
	}
	// The executables a Homebrew keg installs, as {"<version>/bin/<binary>": "<sha256>"}. NULL for
	// every other source, and for a keg until the host reports it. ALGORITHM=INSTANT because this
	// is one of the largest tables in the schema; a rebuild would lock it for the length of a copy.
	if _, err := tx.Exec(`
		ALTER TABLE host_software_installed_paths
		ADD COLUMN executable_hashes JSON NULL, ALGORITHM=INSTANT
	`); err != nil {
		return fmt.Errorf("adding executable_hashes to host_software_installed_paths: %w", err)
	}
	return nil
}

func Down_20260921141651(tx *sql.Tx) error {
	return nil
}
