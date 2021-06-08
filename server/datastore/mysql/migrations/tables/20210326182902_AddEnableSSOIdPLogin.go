package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20210326182902, Down_20210326182902)
}

func Up_20210326182902(tx *sql.Tx) error {
	if _, err := tx.Exec(
		"ALTER TABLE `app_configs` " +
			"ADD COLUMN `enable_sso_idp_login` tinyint(1) NOT NULL DEFAULT '0'",
	); err != nil {
		return errors.Wrap(err, "add enable_sso_idp_login")
	}

	return nil
}

func Down_20210326182902(tx *sql.Tx) error {
	return nil
}
