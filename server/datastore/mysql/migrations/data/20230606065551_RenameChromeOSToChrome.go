package data

import (
	"database/sql"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20230606065551, Down_20230606065551)
}

func Up_20230606065551(tx *sql.Tx) error {
	sql := "UPDATE labels SET name = 'chrome' WHERE label_type = ? AND name = 'ChromeOS'"
	if _, err := tx.Exec(sql, fleet.LabelTypeBuiltIn); err != nil {
		return errors.Wrap(err, "update labels")
	}
	return nil
}

func Down_20230606065551(tx *sql.Tx) error {
	return nil
}
