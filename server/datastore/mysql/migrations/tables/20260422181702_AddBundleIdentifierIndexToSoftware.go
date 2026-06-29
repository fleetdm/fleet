package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260422181702, Down_20260422181702)
}

func Up_20260422181702(tx *sql.Tx) error {
	// Add an index on software.bundle_identifier so the hourly FMA sync cron
	// (UpsertMaintainedApp -> UPDATE software ... WHERE bundle_identifier = ?)
	// performs an indexed lookup instead of a full-table scan.
	//
	// A previous migration (20260326210603_UpdateSoftwareTitleNamesToFMANames)
	// intentionally skipped this index on the assumption that the scan would
	// happen only during a one-time migration and rare FMA additions. The FMA
	// sync cron actually runs hourly and issues the UPDATE per darwin FMA
	// (~224 full scans/hour today), which justifies the index.
	_, err := tx.Exec(`ALTER TABLE software ADD INDEX idx_software_bundle_identifier (bundle_identifier)`)
	if err != nil {
		return fmt.Errorf("failed to add bundle_identifier index to software: %w", err)
	}
	return nil
}

func Down_20260422181702(tx *sql.Tx) error {
	return nil
}
