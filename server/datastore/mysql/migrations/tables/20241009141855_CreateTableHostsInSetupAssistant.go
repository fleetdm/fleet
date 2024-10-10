package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20241009141855, Down_20241009141855)
}

func Up_20241009141855(tx *sql.Tx) error {
	_, err := tx.Exec(`
CREATE TABLE hosts_in_setup_experience (
	host_uuid         VARCHAR(255) NOT NULL PRIMARY KEY,
	in_setup_assistant TINYINT(1) NOT NULL DEFAULT FALSE
)`)
	if err != nil {
		return fmt.Errorf("creating hosts_in_setup_experience table: %w", err)
	}

	return nil
}

func Down_20241009141855(tx *sql.Tx) error {
	return nil
}
