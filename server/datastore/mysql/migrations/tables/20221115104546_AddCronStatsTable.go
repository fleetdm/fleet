package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20221115104546, Down_20221115104546)
}

func Up_20221115104546(tx *sql.Tx) error {
	_, err := tx.Exec(`
		CREATE TABLE cron_stats (
			id INT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			instance VARCHAR(255) NOT NULL,
			stats_type VARCHAR(255) NOT NULL,
			status VARCHAR(255) NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			INDEX idx_cron_stats_name_created_at (name, created_at)
		)
			ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
	`)
	if err != nil {
		return errors.Wrapf(err, "create table")
	}

	return nil
}

func Down_20221115104546(tx *sql.Tx) error {
	return nil
}
