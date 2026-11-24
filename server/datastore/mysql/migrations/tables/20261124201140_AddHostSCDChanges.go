package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20261124201140, Down_20261124201140)
}

// Up_20261124201140 adds the change storage for chart datasets: each entity's current host set, and what changed
// each hour, packed into a few rows per hour so that reading a time range reads few pages.
func Up_20261124201140(tx *sql.Tx) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS host_scd_entities (
			id int unsigned NOT NULL AUTO_INCREMENT,
			dataset varchar(50) COLLATE utf8mb4_unicode_ci NOT NULL,
			entity_id varchar(100) COLLATE utf8mb4_unicode_ci NOT NULL,
			PRIMARY KEY (id),
			UNIQUE KEY uniq_dataset_entity (dataset, entity_id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		`CREATE TABLE IF NOT EXISTS host_scd_current (
			dataset varchar(50) COLLATE utf8mb4_unicode_ci NOT NULL,
			entity_key int unsigned NOT NULL,
			host_bitmap mediumblob NOT NULL,
			encoding_type tinyint NOT NULL,
			PRIMARY KEY (dataset, entity_key)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		`CREATE TABLE IF NOT EXISTS host_scd_changes (
			dataset varchar(50) COLLATE utf8mb4_unicode_ci NOT NULL,
			changed_at datetime NOT NULL,
			seq int unsigned NOT NULL,
			changes mediumblob NOT NULL,
			encoding_type tinyint NOT NULL,
			PRIMARY KEY (dataset, changed_at, seq)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		`CREATE TABLE IF NOT EXISTS host_scd_change_state (
			dataset varchar(50) COLLATE utf8mb4_unicode_ci NOT NULL,
			changes_from datetime NOT NULL,
			covered_from datetime NOT NULL,
			convert_cursor datetime NULL DEFAULT NULL,
			created_at timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			PRIMARY KEY (dataset)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
	}
	for _, stmt := range stmts {
		if _, err := tx.Exec(stmt); err != nil {
			return fmt.Errorf("create chart change tables: %w", err)
		}
	}
	return nil
}

func Down_20261124201140(tx *sql.Tx) error {
	return nil
}
