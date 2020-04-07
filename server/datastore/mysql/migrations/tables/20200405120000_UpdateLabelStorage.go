package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up20200405120000, Down20200405120000)
}

func Up20200405120000(tx *sql.Tx) error {
	if _, err := tx.Exec(
		"CREATE TABLE `label_membership` (" +
			"`created_at` timestamp DEFAULT CURRENT_TIMESTAMP," +
			"`updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP," +
			"`label_id` int(10) unsigned NOT NULL," +
			"`host_id` int(10) unsigned NOT NULL," +
			"PRIMARY KEY (`host_id`, `label_id`)," +
			"INDEX `idx_lm_label_id` (`label_id`)," +
			"CONSTRAINT `fk_lm_host_id` FOREIGN KEY (`host_id`) REFERENCES `hosts` (`id`) ON DELETE CASCADE ON UPDATE CASCADE," +
			"CONSTRAINT `fk_lm_label_id` FOREIGN KEY (`label_id`) REFERENCES `labels` (`id`) ON DELETE CASCADE ON UPDATE CASCADE" +
			") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;",
	); err != nil {
		return errors.Wrap(err, "create label_membership table")
	}

	if _, err := tx.Exec(
		"INSERT IGNORE INTO `label_membership` " +
			"(`created_at`, `updated_at`, `label_id`, `host_id`) " +
			"SELECT `created_at`, `updated_at`, `label_id`, `host_id` " +
			"FROM `label_query_executions` WHERE matches",
	); err != nil {
		return errors.Wrap(err, "copy data from label_query_executions")
	}

	if _, err := tx.Exec(
		"INSERT IGNORE INTO `label_membership` " +
			"(`host_id`, `label_id`) " +
			"SELECT `id`, (SELECT `id` FROM labels WHERE `name` = 'All Hosts' AND `label_type` = 1) FROM `hosts`",
	); err != nil {
		return errors.Wrap(err, "ensure all hosts are in all hosts label")
	}

	if _, err := tx.Exec(
		"DROP TABLE  `label_query_executions`",
	); err != nil {
		return errors.Wrap(err, "drop label_query_executions")
	}

	// MySQL is really particular about using zero values or old values for
	// timestamps, so we set a default value that is plenty far in the past, but
	// hopefully accepted by most MySQL configurations.
	if _, err := tx.Exec(
		"ALTER TABLE  `hosts` " +
			"ADD COLUMN `label_update_time` timestamp NOT NULL DEFAULT '2000-01-01 00:00:00'",
	); err != nil {
		return errors.Wrap(err, "drop label_query_executions")
	}

	return nil
}

func Down20200405120000(tx *sql.Tx) error {
	return nil
}
