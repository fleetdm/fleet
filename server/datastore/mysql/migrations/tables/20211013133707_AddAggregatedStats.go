package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20211013133707, Down_20211013133707)
}

func Up_20211013133707(tx *sql.Tx) error {
	aggregatedStatsTable := `
		CREATE TABLE IF NOT EXISTS aggregated_stats (
			id int(10) UNSIGNED NOT NULL,
			type VARCHAR(255) NOT NULL,
			json_value JSON NOT NULL,
			created_at timestamp DEFAULT CURRENT_TIMESTAMP,
			updated_at timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			PRIMARY KEY (id, type),
			INDEX idx_aggregated_stats_updated_at(updated_at)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
	`
	if _, err := tx.Exec(aggregatedStatsTable); err != nil {
		return errors.Wrap(err, "create aggregated stats table")
	}
	return nil
}

func Down_20211013133707(tx *sql.Tx) error {
	return nil
}
