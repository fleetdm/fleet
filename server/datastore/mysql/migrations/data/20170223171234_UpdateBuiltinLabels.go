package data

import (
	"database/sql"

	"github.com/fleetdm/fleet/server/datastore/internal/appstate"
)

func init() {
	MigrationClient.AddMigration(Up_20170223171234, Down_20170223171234)
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

	for _, label := range appstate.Labels2() {
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

	for _, label := range appstate.Labels2() {
		_, err := tx.Exec(sql, label.Name, label.LabelType, label.Query)
		if err != nil {
			return err
		}
	}

	// Insert the old labels
	Up_20161229171615(tx)

	return nil
}
