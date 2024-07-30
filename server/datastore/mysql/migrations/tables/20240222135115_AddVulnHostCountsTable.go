package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20240222135115, Down_20240222135115)
}

func Up_20240222135115(tx *sql.Tx) error {
	createStmt := `
		CREATE TABLE IF NOT EXISTS vulnerability_host_counts (
			cve VARCHAR(20) NOT NULL,
			team_id int(10) UNSIGNED NOT NULL DEFAULT 0,
			host_count int(10) UNSIGNED NOT NULL DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			UNIQUE KEY cve_team_id (cve, team_id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
			`
	_, err := tx.Exec(createStmt)
	if err != nil {
		return err
	}

	return nil
}

func Down_20240222135115(_ *sql.Tx) error {
	return nil
}
