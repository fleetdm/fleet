package data

import (
	"database/sql"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

func init() {
	MigrationClient.AddMigration(Up_20161229171615, Down_20161229171615)
}

// Labels1 is the set of builtin labels that should be populated in the
// datastore
func Labels1() []fleet.Label {
	return []fleet.Label{
		{
			Name:      "All Hosts",
			Query:     "select 1;",
			LabelType: fleet.LabelTypeBuiltIn,
		},
		{
			Platform:  "darwin",
			Name:      "Mac OS X",
			Query:     "select 1 from osquery_info where build_platform = 'darwin';",
			LabelType: fleet.LabelTypeBuiltIn,
		},
		{
			Platform:  "ubuntu",
			Name:      "Ubuntu Linux",
			Query:     "select 1 from osquery_info where build_platform = 'ubuntu';",
			LabelType: fleet.LabelTypeBuiltIn,
		},
		{
			Platform:  "centos",
			Name:      "CentOS Linux",
			Query:     "select 1 from osquery_info where build_platform = 'centos';",
			LabelType: fleet.LabelTypeBuiltIn,
		},
		{
			Platform:  "windows",
			Name:      "MS Windows",
			Query:     "select 1 from osquery_info where build_platform = 'windows';",
			LabelType: fleet.LabelTypeBuiltIn,
		},
	}
}

func Up_20161229171615(tx *sql.Tx) error {
	sql := `
		INSERT INTO labels (
			name,
			description,
			query,
			platform,
			label_type
		) VALUES (?, ?, ?, ?, ?)
`

	for _, label := range Labels1() {
		_, err := tx.Exec(sql, label.Name, label.Description, label.Query, label.Platform, label.LabelType)
		if err != nil {
			return err
		}
	}

	return nil
}

func Down_20161229171615(tx *sql.Tx) error {
	sql := `
		DELETE FROM labels
		WHERE name = ? AND label_type = ? AND query = ?
`

	for _, label := range Labels1() {
		_, err := tx.Exec(sql, label.Name, label.LabelType, label.Query)
		if err != nil {
			return err
		}
	}

	return nil
}
