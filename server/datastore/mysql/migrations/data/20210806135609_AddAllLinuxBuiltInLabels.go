package data

import (
	"database/sql"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

func init() {
	MigrationClient.AddMigration(Up_20210806135609, Down_20210806135609)
}

func Up_20210806135609(tx *sql.Tx) error {
	_, err := tx.Exec(`
		INSERT INTO labels (
			name,
			description,
			query,
			platform,
			label_type
		) VALUES (?, ?, ?, ?, ?)`,
		"All Linux", "All Linux distributions",
		"SELECT 1 FROM osquery_info WHERE build_platform LIKE '%ubuntu%' OR build_distro LIKE '%centos%';",
		"",
		fleet.LabelTypeBuiltIn,
	)
	if err != nil {
		return err
	}

	return nil
}

func Down_20210806135609(tx *sql.Tx) error {
	return nil
}
