package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260406213703, Down_20260406213703)
}

func Up_20260406213703(tx *sql.Tx) error {
	_, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS host_daily_data_bitmaps (
			host_id      INT UNSIGNED NOT NULL,
			dataset      VARCHAR(50) NOT NULL,
			entity_id    INT UNSIGNED NOT NULL DEFAULT 0,
			chart_date   DATE NOT NULL,
			hours_bitmap INT UNSIGNED NOT NULL DEFAULT 0,
			created_at   TIMESTAMP NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at   TIMESTAMP NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			PRIMARY KEY (host_id, dataset, entity_id, chart_date),
			INDEX idx_dataset_date (dataset, chart_date, entity_id)
		)
	`)
	if err != nil {
		return fmt.Errorf("create host_daily_data_bitmaps table: %w", err)
	}

	_, err = tx.Exec(`
		CREATE TABLE IF NOT EXISTS host_hourly_data_blobs (
			dataset      VARCHAR(50) NOT NULL,
			entity_id    INT UNSIGNED NOT NULL DEFAULT 0,
			chart_date   DATE NOT NULL,
			hour         TINYINT UNSIGNED NOT NULL,
			host_bitmap  MEDIUMBLOB NOT NULL,
			created_at   TIMESTAMP NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at   TIMESTAMP NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			PRIMARY KEY (dataset, entity_id, chart_date, hour)
		)
	`)
	if err != nil {
		return fmt.Errorf("create host_hourly_data_blobs table: %w", err)
	}

	return nil
}

func Down_20260406213703(tx *sql.Tx) error {
	return nil
}
