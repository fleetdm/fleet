package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260413175411, Down_20260413175411)
}

func Up_20260413175411(tx *sql.Tx) error {
	// host_scd_data stores per-entity host set as a daily-granularity Type-2 slowly-
	// changing dimension: one row per (dataset, entity_id, valid_from) day, with
	// host_bitmap encoding which hosts are associated with that entity on that day.
	// Unchanged state keeps the same row open (valid_to = sentinel) across days.
	// See server/chart/internal/mysql/scd.go for the upsert/query logic.
	_, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS host_scd_data (
			id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			dataset     VARCHAR(50)  NOT NULL,
			entity_id   VARCHAR(100) NOT NULL,
			host_bitmap MEDIUMBLOB   NOT NULL,
			valid_from  DATE         NOT NULL,
			valid_to    DATE         NOT NULL DEFAULT '9999-12-31',
			PRIMARY KEY (id),
			UNIQUE KEY uniq_entity_day (dataset, entity_id, valid_from),
			KEY idx_dataset_range      (dataset, valid_from, valid_to)
		)
	`)
	if err != nil {
		return fmt.Errorf("create host_scd_data table: %w", err)
	}

	return nil
}

func Down_20260413175411(tx *sql.Tx) error {
	return nil
}
