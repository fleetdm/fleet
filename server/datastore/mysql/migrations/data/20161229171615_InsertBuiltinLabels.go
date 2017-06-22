package data

import (
	"database/sql"

	"github.com/kolide/fleet/server/datastore/internal/appstate"
)

func init() {
	MigrationClient.AddMigration(Up_20161229171615, Down_20161229171615)
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

	for _, label := range appstate.Labels1() {
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

	for _, label := range appstate.Labels1() {
		_, err := tx.Exec(sql, label.Name, label.LabelType, label.Query)
		if err != nil {
			return err
		}
	}

	return nil
}
