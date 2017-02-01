package data

import (
	"database/sql"

	"github.com/kolide/kolide/server/datastore/internal/appstate"
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

	for _, label := range appstate.Labels() {
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
		WHERE name = ? AND label_type = ?
`

	for _, label := range appstate.Labels() {
		_, err := tx.Exec(sql, label.Name, label.LabelType)
		if err != nil {
			return err
		}
	}

	return nil
}
