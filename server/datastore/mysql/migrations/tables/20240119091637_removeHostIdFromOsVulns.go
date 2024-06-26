package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20240119091637, Down_20240119091637)
}

func Up_20240119091637(tx *sql.Tx) error {
	// operating_system_vulnerabilities is not previously used
	// truncating table is safe
	stmt := `
		TRUNCATE TABLE operating_system_vulnerabilities
	`
	if _, err := tx.Exec(stmt); err != nil {
		return fmt.Errorf("truncating operating_system_vulnerabilities: %w", err)
	}

	stmt = `
		ALTER TABLE operating_system_vulnerabilities
		DROP INDEX idx_operating_system_vulnerabilities_unq_cve
	`
	if _, err := tx.Exec(stmt); err != nil {
		return fmt.Errorf("dropping index idx_operating_system_vulnerabilities_unq_cve: %w", err)
	}

	stmt = `
		ALTER TABLE operating_system_vulnerabilities
		DROP INDEX idx_operating_system_vulnerabilities_operating_system_id_cve
	`
	if _, err := tx.Exec(stmt); err != nil {
		return fmt.Errorf("dropping index idx_operating_system_vulnerabilities_operating_system_id_cve: %w", err)
	}

	stmt = `
		ALTER TABLE operating_system_vulnerabilities
		DROP COLUMN host_id
	`
	if _, err := tx.Exec(stmt); err != nil {
		return fmt.Errorf("dropping host_id column from operating_system_vulnerabilities: %w", err)
	}

	stmt = `
		ALTER TABLE operating_system_vulnerabilities
		ADD UNIQUE INDEX idx_os_vulnerabilities_unq_os_id_cve (operating_system_id, cve)
	`
	if _, err := tx.Exec(stmt); err != nil {
		return fmt.Errorf("adding index idx_operating_system_vulnerabilities_unq_cve: %w", err)
	}

	return nil
}

func Down_20240119091637(tx *sql.Tx) error {
	return nil
}
