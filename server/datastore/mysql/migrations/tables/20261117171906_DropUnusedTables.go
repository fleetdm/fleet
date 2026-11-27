package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration("Drop unused tables", Up_20261117171906, Down_20261117171906)
}

func Up_20261117171906(tx *sql.Tx) error {
	for _, table := range []string{
		"network_interfaces",
		"osquery_options",
		"ca_config_assets",
		"legacy_host_filevault_profiles",
		"legacy_host_mdm_idp_accounts",
	} {
		if _, err := tx.Exec("DROP TABLE IF EXISTS `" + table + "`"); err != nil {
			return fmt.Errorf("dropping %s: %w", table, err)
		}
	}
	return nil
}

func Down_20261117171906(tx *sql.Tx) error {
	return nil
}
