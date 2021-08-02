package data

import (
	"database/sql"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

func init() {
	MigrationClient.AddMigration(Up_20170223171234, Down_20170223171234)
}

func Labels2() []fleet.Label {
	return []fleet.Label{
		{
			Name:        "All Hosts",
			Query:       "select 1;",
			Description: "All hosts which have enrolled in Fleet",
			LabelType:   fleet.LabelTypeBuiltIn,
		},
		{
			Name:        "macOS",
			Query:       "select 1 from os_version where platform = 'darwin';",
			Description: "All macOS hosts",
			LabelType:   fleet.LabelTypeBuiltIn,
		},
		{
			Name:        "Ubuntu Linux",
			Query:       "select 1 from os_version where platform = 'ubuntu';",
			Description: "All Ubuntu hosts",
			LabelType:   fleet.LabelTypeBuiltIn,
		},
		{
			Name:        "CentOS Linux",
			Query:       "select 1 from os_version where platform = 'centos';",
			Description: "All CentOS hosts",
			LabelType:   fleet.LabelTypeBuiltIn,
		},
		{
			Name:        "MS Windows",
			Query:       "select 1 from os_version where platform = 'windows';",
			Description: "All Windows hosts",
			LabelType:   fleet.LabelTypeBuiltIn,
		},
	}
}

func Up_20170223171234(tx *sql.Tx) error {
	// Remove the old labels
	Down_20161229171615(tx)

	// Insert the new labels
	sql := `
		INSERT INTO labels (
			name,
			description,
			query,
			platform,
			label_type
		) VALUES (?, ?, ?, ?, ?)
`

	for _, label := range Labels2() {
		_, err := tx.Exec(sql, label.Name, label.Description, label.Query, label.Platform, label.LabelType)
		if err != nil {
			return err
		}
	}

	return nil
}

func Down_20170223171234(tx *sql.Tx) error {
	// Remove the new labels
	sql := `
		DELETE FROM labels
		WHERE name = ? AND label_type = ? AND QUERY = ?
`

	for _, label := range Labels2() {
		_, err := tx.Exec(sql, label.Name, label.LabelType, label.Query)
		if err != nil {
			return err
		}
	}

	// Insert the old labels
	Up_20161229171615(tx)

	return nil
}
