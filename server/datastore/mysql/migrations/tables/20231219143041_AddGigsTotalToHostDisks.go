package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20231219143041, Down_20231219143041)
}

func Up_20231219143041(tx *sql.Tx) error {
	stmt := `
		ALTER TABLE host_disks
		ADD COLUMN gigs_total_disk_space decimal(10,2) NOT NULL DEFAULT '0.00';
	`
	if _, err := tx.Exec(stmt); err != nil {
		return fmt.Errorf("add gigs_total_disk_space to host_disks: %w", err)
	}

	return nil
}

func Down_20231219143041(tx *sql.Tx) error {
	return nil
}
