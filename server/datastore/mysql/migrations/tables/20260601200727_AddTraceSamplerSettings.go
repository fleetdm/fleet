package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20260601200727, Down_20260601200727)
}

func Up_20260601200727(tx *sql.Tx) error {
	// Singleton settings row holding the runtime tunable trace sampling configuration. Operators flip these via PATCH
	// /debug/trace_sampler. The /debug auth log and the PATCH access log already record who made the change.
	_, err := tx.Exec(`
		CREATE TABLE trace_sampler_settings (
			id                TINYINT UNSIGNED NOT NULL PRIMARY KEY,
			high_volume_ratio DOUBLE NOT NULL DEFAULT 0.001,
			standard_ratio    DOUBLE NOT NULL DEFAULT 0.02,
			force_full        TINYINT(1) NOT NULL DEFAULT 0,
			updated_at        TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			CONSTRAINT ck_trace_sampler_settings_singleton  CHECK (id = 1),
			CONSTRAINT ck_trace_sampler_settings_high_range CHECK (high_volume_ratio BETWEEN 0 AND 1),
			CONSTRAINT ck_trace_sampler_settings_std_range  CHECK (standard_ratio BETWEEN 0 AND 1)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
	`)
	if err != nil {
		return err
	}

	// Insert the seed row with an explicit fixed updated_at instead of letting the column default to CURRENT_TIMESTAMP.
	// schema.sql captures the row's data; if updated_at varied per migration run, every developer would get a different schema
	// dump and `make test-schema` would never pass.
	_, err = tx.Exec(`INSERT INTO trace_sampler_settings (id, updated_at) VALUES (1, '2026-06-01 00:00:00')`)
	return err
}

func Down_20260601200727(tx *sql.Tx) error {
	return nil
}
