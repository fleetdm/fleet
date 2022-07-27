package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20220727093635, Down_20220727093635)
}

func Up_20220727093635(tx *sql.Tx) error {
	fmt.Println("Adding timestamps to 'host_device_auth'...")
	_, err := tx.Exec(`
		ALTER TABLE host_device_auth
			ADD COLUMN created_at timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
			ADD COLUMN updated_at timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
`)
	if err == nil {
		fmt.Println("Done adding timestamps to 'host_device_auth'...")
	}
	return err
}

func Down_20220727093635(tx *sql.Tx) error {
	return nil
}
