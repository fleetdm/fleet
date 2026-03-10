package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260306120000, Down_20260306120000)
}

func Up_20260306120000(tx *sql.Tx) error {
	_, err := tx.Exec(`RENAME TABLE activities TO activity_past, host_activities TO activity_host_past`)
	if err != nil {
		return fmt.Errorf("rename activities tables: %w", err)
	}
	return nil
}

func Down_20260306120000(tx *sql.Tx) error {
	return nil
}
