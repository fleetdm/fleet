package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260723181410, Down_20260723181410)
}

func Up_20260723181410(tx *sql.Tx) error {
	_, err := tx.Exec(`
		CREATE TABLE custom_host_vitals (
			id INT UNSIGNED NOT NULL AUTO_INCREMENT,
			name VARCHAR(255) COLLATE utf8mb4_unicode_ci NOT NULL,
			-- Using DATETIME instead of TIMESTAMP to prevent future Y2K38 issues.
			created_at DATETIME(6) NOT NULL DEFAULT NOW(6),
			updated_at DATETIME(6) NOT NULL DEFAULT NOW(6) ON UPDATE NOW(6),
			PRIMARY KEY (id),
			CONSTRAINT idx_custom_host_vitals_name UNIQUE (name)
		) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci`,
	)
	if err != nil {
		return fmt.Errorf("failed to create custom_host_vitals table: %w", err)
	}

	_, err = tx.Exec(`
		CREATE TABLE host_custom_host_vitals (
			id INT UNSIGNED NOT NULL AUTO_INCREMENT,
			host_id INT UNSIGNED NOT NULL,
			custom_host_vital_id INT UNSIGNED NOT NULL,
			value TEXT COLLATE utf8mb4_unicode_ci NOT NULL,
			created_at DATETIME(6) NOT NULL DEFAULT NOW(6),
			updated_at DATETIME(6) NOT NULL DEFAULT NOW(6) ON UPDATE NOW(6),
			PRIMARY KEY (id),
			CONSTRAINT idx_host_custom_host_vitals_host_vital UNIQUE (host_id, custom_host_vital_id),
			-- No FK on host_id (see handbook/engineering/scaling-fleet.md): rows are
			-- cleaned up on host deletion via the hostRefs list instead.
			CONSTRAINT fk_host_custom_host_vitals_custom_host_vital_id
				FOREIGN KEY (custom_host_vital_id) REFERENCES custom_host_vitals (id) ON DELETE CASCADE
		) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci`,
	)
	if err != nil {
		return fmt.Errorf("failed to create host_custom_host_vitals table: %w", err)
	}

	return nil
}

func Down_20260723181410(tx *sql.Tx) error {
	return nil
}
