package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260423161823, Down_20260423161823)
}

func Up_20260423161823(tx *sql.Tx) error {
	// host_scd_data is the unified storage for all chart datasets. Rows are
	// interval-based (valid_from, valid_to) bitmaps, written by one of two sample
	// strategies:
	//   - Accumulate: rows are explicitly closed at bucket boundaries; same-bucket
	//     samples are OR-merged into the existing row via ON DUPLICATE KEY UPDATE.
	//     Used for datasets like uptime where each sample is a partial observation.
	//   - Snapshot: rows stay open (valid_to = sentinel) until the bitmap changes,
	//     at which point the prior row is closed and a new one inserted. Used for
	//     datasets like CVE where each sample is the full state.
	// See server/chart/internal/mysql/data.go for the write and read paths.
	_, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS host_scd_data (
			id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			dataset     VARCHAR(50)  CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
			entity_id   VARCHAR(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
			host_bitmap MEDIUMBLOB   NOT NULL,
			valid_from  DATETIME     NOT NULL,
			valid_to    DATETIME     NOT NULL DEFAULT '9999-12-31 00:00:00',
			created_at  TIMESTAMP    NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at  TIMESTAMP    NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			PRIMARY KEY (id),
			UNIQUE KEY uniq_entity_bucket (dataset, entity_id, valid_from),
			KEY idx_dataset_range         (dataset, valid_from, valid_to)
		) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci
	`)
	if err != nil {
		return fmt.Errorf("create host_scd_data table: %w", err)
	}

	return nil
}

func Down_20260423161823(tx *sql.Tx) error {
	return nil
}
