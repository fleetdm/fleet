package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20210622160235, Down_20210622160235)
}

func Up_20210622160235(tx *sql.Tx) error {
	// Analytics default to off when migrating an existing installation. New
	// installations will have this set to on during the setup process.
	sql := `
		ALTER TABLE app_configs
		ADD COLUMN enable_analytics tinyint(1) NOT NULL DEFAULT 0
	`
	if _, err := tx.Exec(sql); err != nil {
		return errors.Wrap(err, "add analytics")
	}
	return nil
}

func Down_20210622160235(tx *sql.Tx) error {
	return nil
}
