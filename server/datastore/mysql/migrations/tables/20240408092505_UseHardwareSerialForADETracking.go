package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20240408092505, Down_20240408092505)
}

func Up_20240408092505(tx *sql.Tx) error {

	_, err := tx.Exec(`
ALTER TABLE host_dep_assignments
-- The column host_hardware_serial is linked to the hardware_serial column in
-- the hosts table, reflecting the hardware serial number of the host registered
-- in Fleet within ABM. We opt for serial numbers over host IDs because, the
-- assignment in ABM, determined by serial number, might remain even after a host
-- is deleted.
--
-- VARCHAR(255) is not the most efficient type to store serial numbers, but
-- it's what the hosts table uses.
ADD COLUMN host_hardware_serial VARCHAR(255) COLLATE utf8mb4_unicode_ci AFTER host_id`)
	if err != nil {
		return fmt.Errorf("adding host_hardware_serial to host_dep_assignments: %w", err)
	}

	_, err = tx.Exec(`
UPDATE host_dep_assignments hda
JOIN hosts h ON hda.host_id = h.id
SET hda.host_hardware_serial = h.hardware_serial`)
	if err != nil {
		return fmt.Errorf("updating serial numbers in host_dep_assignments: %w", err)
	}

	_, err = tx.Exec(`
DELETE FROM host_dep_assignments
WHERE host_hardware_serial IS NULL`)
	if err != nil {
		return fmt.Errorf("deleting rows with empty serial numbers in host_dep_assignments: %w", err)
	}

	_, err = tx.Exec(`
ALTER TABLE host_dep_assignments
MODIFY host_hardware_serial VARCHAR(255) COLLATE utf8mb4_unicode_ci NOT NULL`)
	if err != nil {
		return fmt.Errorf("setting host_hardware_serial to NOT NULL in host_dep_assignments: %w", err)
	}

	_, err = tx.Exec(`
ALTER TABLE host_dep_assignments
DROP PRIMARY KEY`)
	if err != nil {
		return fmt.Errorf("deleting host_id column from host_dep_assignments: %w", err)
	}

	_, err = tx.Exec(`
ALTER TABLE host_dep_assignments
ADD PRIMARY KEY (host_hardware_serial)`)
	if err != nil {
		return fmt.Errorf("adding host_hardware_serial as primary key for host_dep_assignments: %w", err)
	}

	_, err = tx.Exec(`
ALTER TABLE host_dep_assignments
DROP COLUMN host_id`)
	if err != nil {
		return fmt.Errorf("deleting host_id column from host_dep_assignments: %w", err)
	}

	// trigger a full ABM sync on the next cron run
	_, err = tx.Exec(`UPDATE nano_dep_names SET syncer_cursor = NULL`)
	if err != nil {
		return fmt.Errorf("setting NULL syncer cursor for ABM: %w", err)
	}

	return nil
}

func Down_20240408092505(tx *sql.Tx) error {
	return nil
}
