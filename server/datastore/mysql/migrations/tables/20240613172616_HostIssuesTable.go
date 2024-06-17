package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20240613172616, Down_20240613172616)
}

func Up_20240613172616(tx *sql.Tx) error {
	_, err := tx.Exec(
		`
		CREATE TABLE host_issues (
			host_id INT(10) UNSIGNED NOT NULL PRIMARY KEY,
			failing_policies_count INT(10) UNSIGNED NOT NULL DEFAULT 0,
			critical_vulnerabilities_count INT(10) UNSIGNED NOT NULL DEFAULT 0,
			total_issues_count INT(10) UNSIGNED NOT NULL DEFAULT 0, -- could use generated column for MySQL 8+
			created_at TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3), -- millisecond precision
			updated_at TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3), -- millisecond precision
			INDEX (total_issues_count)
		)`,
	)
	if err != nil {
		return fmt.Errorf("failed to create host_issues table: %w", err)
	}
	return nil
}

func Down_20240613172616(_ *sql.Tx) error {
	return nil
}
