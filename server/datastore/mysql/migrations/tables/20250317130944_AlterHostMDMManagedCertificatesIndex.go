package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20250317130944, Down_20250317130944)
}

func Up_20250317130944(tx *sql.Tx) error {
	_, err := tx.Exec(`
	ALTER TABLE host_mdm_managed_certificates
	DROP PRIMARY KEY,
	ADD PRIMARY KEY (host_uuid, profile_uuid, ca_name)
	`)
	if err != nil {
		return fmt.Errorf("failed to update primary key in host_mdm_managed_certificates table: %s", err)
	}
	return nil
}

func Down_20250317130944(_ *sql.Tx) error {
	return nil
}
