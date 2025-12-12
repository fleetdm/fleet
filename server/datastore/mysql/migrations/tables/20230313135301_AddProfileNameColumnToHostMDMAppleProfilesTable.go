package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20230313135301, Down_20230313135301)
}

func Up_20230313135301(tx *sql.Tx) error {
	if _, err := tx.Exec("ALTER TABLE `host_mdm_apple_profiles` ADD COLUMN `profile_name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT ''"); err != nil {
		return errors.Wrap(err, "adding profile_name column to `host_mdm_apple_profiles")
	}
	return nil
}

func Down_20230313135301(tx *sql.Tx) error {
	return nil
}
