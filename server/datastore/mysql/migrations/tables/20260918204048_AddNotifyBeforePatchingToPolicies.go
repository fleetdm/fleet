package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260918204048, Down_20260918204048)
}

func Up_20260918204048(tx *sql.Tx) error {
	if !columnExists(tx, "policies", "notify_before_patching") {
		if _, err := tx.Exec(`
			ALTER TABLE policies
			ADD COLUMN notify_before_patching TINYINT(1) NOT NULL DEFAULT 0,
			ALGORITHM=INSTANT
		`); err != nil {
			return fmt.Errorf("add notify_before_patching to policies: %w", err)
		}
	}

	if !columnExists(tx, "host_software_installs", "override_pre_install_query") {
		if _, err := tx.Exec(`
			ALTER TABLE host_software_installs
			ADD COLUMN override_pre_install_query TINYINT(1) NOT NULL DEFAULT 0,
			ALGORITHM=INSTANT
		`); err != nil {
			return fmt.Errorf("add override_pre_install_query to host_software_installs: %w", err)
		}
	}

	// Installs already in flight at upgrade were queued before this column existed. Stamp the
	// ones whose policy patches only when closed so fleetd keeps the app-open gate for them.
	// Finished rows keep their recorded outcome.
	if _, err := tx.Exec(`
		UPDATE host_software_installs
		SET override_pre_install_query = 1
		WHERE execution_status = 'pending_install'
			AND policy_id IN (SELECT id FROM policies WHERE patch_when_closed = 1)
	`); err != nil {
		return fmt.Errorf("backfill override_pre_install_query on in-flight host_software_installs: %w", err)
	}

	return nil
}

func Down_20260918204048(tx *sql.Tx) error {
	return nil
}
