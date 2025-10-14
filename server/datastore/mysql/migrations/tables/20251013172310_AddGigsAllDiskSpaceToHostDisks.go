package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20251013172310, Down_20251013172310)
}

func Up_20251013172310(tx *sql.Tx) error {
	// NULLable since only relevant for Linux hosts
	stmt := `
		ALTER TABLE host_disks
		ADD COLUMN gigs_all_disk_space decimal(10,2)	
	`
	if _, err := tx.Exec(stmt); err != nil {
		return fmt.Errorf("add gigs_all_disk_space to host_disks: %w", err)
	}
	return nil
}

func Down_20251013172310(tx *sql.Tx) error {
	return nil
}
