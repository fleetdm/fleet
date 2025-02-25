package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20240826111228, Down_20240826111228)
}

func Up_20240826111228(tx *sql.Tx) error {
	_, err := tx.Exec(`
	    ALTER TABLE hosts
	    DROP INDEX host_ip_mac_search
	`)
	if err != nil {
		return fmt.Errorf("dropping host_ip_mac_search index: %w", err)
	}

	_, err = tx.Exec(`
	    ALTER TABLE hosts
	    DROP INDEX hosts_search
	`)
	if err != nil {
		return fmt.Errorf("dropping hosts_search index: %w", err)
	}

	_, err = tx.Exec(`
	    ALTER TABLE hosts
	    ADD INDEX idx_hosts_uuid (uuid);
	`)
	if err != nil {
		return fmt.Errorf("adding hosts_uuid index: %w", err)
	}
	return nil
}

func Down_20240826111228(tx *sql.Tx) error {
	return nil
}
