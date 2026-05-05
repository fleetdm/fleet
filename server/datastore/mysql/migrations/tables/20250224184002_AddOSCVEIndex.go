package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20250224184002, Down_20250224184002)
}

func Up_20250224184002(tx *sql.Tx) error {
	_, err := tx.Exec(`ALTER TABLE operating_system_vulnerabilities ADD INDEX idx_os_vulnerabilities_cve (cve);`)
	if err != nil {
		return fmt.Errorf("failed to add index to operating_system_vulnerabilities.cve: %w", err)
	}
	return nil
}

func Down_20250224184002(tx *sql.Tx) error {
	return nil
}
