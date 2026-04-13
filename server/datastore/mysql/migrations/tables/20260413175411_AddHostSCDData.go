package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260413175411, Down_20260413175411)
}

func Up_20260413175411(tx *sql.Tx) error {
	_, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS host_scd_data (
			id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			dataset     VARCHAR(50)  NOT NULL,
			host_id     INT UNSIGNED NOT NULL,
			entity_id   VARCHAR(100) NOT NULL,
			valid_from  DATETIME     NOT NULL,
			valid_to    DATETIME     NOT NULL DEFAULT '9999-12-31 23:59:59',
			PRIMARY KEY (id),
			UNIQUE KEY uniq_active     (dataset, host_id, entity_id, valid_to),
			KEY idx_dataset_time       (dataset, valid_to, valid_from),
			KEY idx_dataset_entity     (dataset, entity_id, valid_to, valid_from)
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
