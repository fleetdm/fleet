package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20200512120000, Down_20200512120000)
}

func Up_20200512120000(tx *sql.Tx) error {
	_, err := tx.Exec(
		"CREATE TABLE `enroll_secrets` (" +
			"`name` VARCHAR(255) NOT NULL," +
			"`created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP," +
			"`secret` VARCHAR(255) NOT NULL," +
			"`active` TINYINT(1) DEFAULT TRUE," +
			"PRIMARY KEY (`name`)" +
			") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;",
	)
	if err != nil {
		return errors.Wrap(err, "create enroll_secrets table")
	}

	_, err = tx.Exec(
		"INSERT INTO `enroll_secrets` (`name`, `secret`, `active`)" +
			"SELECT 'default', `osquery_enroll_secret`, TRUE FROM `app_configs` WHERE osquery_enroll_secret != ''",
	)
	if err != nil {
		return errors.Wrap(err, "copy existing enroll secret")
	}

	_, err = tx.Exec(
		"ALTER TABLE `hosts`" +
			"ADD COLUMN `enroll_secret_name` VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT ''",
	)
	if err != nil {
		return errors.Wrap(err, "drop old secret column")
	}

	_, err = tx.Exec(
		"ALTER TABLE `app_configs`" +
			"DROP COLUMN `osquery_enroll_secret`",
	)
	if err != nil {
		return errors.Wrap(err, "drop old secret column")
	}

	return nil
}

func Down_20200512120000(tx *sql.Tx) error {
	return nil
}
