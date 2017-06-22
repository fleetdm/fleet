package data

import (
	"database/sql"

	"github.com/kolide/fleet/server/datastore/internal/appstate"
	"github.com/kolide/fleet/server/kolide"
)

func init() {
	MigrationClient.AddMigration(Up_20161223115449, Down_20161223115449)
}

func Up_20161223115449(tx *sql.Tx) error {
	sqlStatement := `
		INSERT INTO options (
			name,
			type,
			value,
			read_only
		) VALUES (?, ?, ?, ?)
	`

	for _, opt := range appstate.Options() {
		ov := kolide.Option{
			Name:     opt.Name,
			ReadOnly: opt.ReadOnly,
			Type:     opt.Type,
			Value: kolide.OptionValue{
				Val: opt.Value,
			},
		}
		_, err := tx.Exec(sqlStatement, ov.Name, ov.Type, ov.Value, ov.ReadOnly)
		if err != nil {
			return err
		}

	}
	return nil
}

func Down_20161223115449(tx *sql.Tx) error {
	sqlStatement := `
		DELETE FROM options
		WHERE name = ?
	`
	for _, opt := range appstate.Options() {
		_, err := tx.Exec(sqlStatement, opt.Name)
		if err != nil {
			return err
		}
	}
	return nil
}
