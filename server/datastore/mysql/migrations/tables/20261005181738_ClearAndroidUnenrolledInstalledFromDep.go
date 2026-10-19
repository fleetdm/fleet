package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20261005181738, Down_20261005181738)
}

// Turning off Android MDM left installed_from_dep = 1 on company-owned hosts, which reads as "Pending".
// Android has no DEP/ABM equivalent, so an unenrolled Android host is always "Off".
func Up_20261005181738(tx *sql.Tx) error {
	if _, err := tx.Exec(`
UPDATE host_mdm hm
JOIN hosts h ON h.id = hm.host_id
SET hm.installed_from_dep = 0
WHERE h.platform = 'android'
	AND hm.enrolled = 0
	AND hm.installed_from_dep = 1`); err != nil {
		return fmt.Errorf("clear installed_from_dep for unenrolled android hosts: %w", err)
	}
	return nil
}

func Down_20261005181738(tx *sql.Tx) error {
	return nil
}
