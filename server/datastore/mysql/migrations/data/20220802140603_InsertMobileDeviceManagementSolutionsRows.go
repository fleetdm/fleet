package data

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20220802140603, Down_20220802140603)
}

func Up_20220802140603(tx *sql.Tx) error {
	_, err := tx.Exec(`
		INSERT INTO mobile_device_management_solutions (
			name,
			server_url
		) VALUES
			('Kandji', ''),
			('Jamf', ''),
			('VMware Workspace ONE', ''),
			('Intune', ''),
			('SimpleMDM', '')
		`)
	return err
}

func Down_20220802140603(tx *sql.Tx) error {
	return nil
}
