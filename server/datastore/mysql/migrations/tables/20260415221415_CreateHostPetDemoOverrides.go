package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260415221415, Down_20260415221415)
}

// Up_20260415221415 creates the host_pet_demo_overrides table. Each row holds
// per-host knobs that the pet derivation function reads on top of real host
// state when the demo build tag is enabled — used to drive demos without
// poisoning the real hosts / policies / vulnerabilities tables.
func Up_20260415221415(tx *sql.Tx) error {
	if _, err := tx.Exec(`
		CREATE TABLE host_pet_demo_overrides (
			host_id                INT UNSIGNED NOT NULL,
			seen_time_override     TIMESTAMP(6) NULL,
			time_offset_hours      INT NOT NULL DEFAULT 0,
			extra_failing_policies INT UNSIGNED NOT NULL DEFAULT 0,
			extra_critical_vulns   INT UNSIGNED NOT NULL DEFAULT 0,
			extra_high_vulns       INT UNSIGNED NOT NULL DEFAULT 0,
			created_at             TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
			updated_at             TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
			PRIMARY KEY (host_id),
			CONSTRAINT fk_host_pet_demo_overrides_host_id FOREIGN KEY (host_id) REFERENCES hosts (id) ON DELETE CASCADE
		)
	`); err != nil {
		return fmt.Errorf("creating host_pet_demo_overrides table: %w", err)
	}
	return nil
}

func Down_20260415221415(tx *sql.Tx) error {
	return nil
}
