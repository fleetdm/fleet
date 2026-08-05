package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260805011013, Down_20260805011013)
}

func Up_20260805011013(tx *sql.Tx) error {
	// fleet_managed_key marks policies whose query Fleet owns and may rewrite
	// (e.g. macOS OS-currency policies driven by Apple's GDMF catalog).
	// NULL means user-owned. Ownership is set only via an explicit
	// fleet_managed_key in GitOps/API — this migration does not claim policies
	// by name (aliases can collide on the unique key, and customized policies
	// must not be rewritten).
	//
	// Uniqueness is per team scope, including global (team_id IS NULL): a plain
	// UNIQUE(team_id, key) would not enforce that because MySQL treats NULLs as
	// distinct in unique indexes.
	_, err := tx.Exec(`
ALTER TABLE policies
  ADD COLUMN fleet_managed_key VARCHAR(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  ADD COLUMN fleet_managed_team_key VARCHAR(96) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci
    GENERATED ALWAYS AS (
      IF(fleet_managed_key IS NULL, NULL,
         CONCAT_WS(':', IF(team_id IS NULL, 'global', CONVERT(team_id, CHAR)), fleet_managed_key))
    ) STORED,
  ADD UNIQUE KEY idx_policies_fleet_managed_team_key (fleet_managed_team_key),
  ADD KEY idx_policies_fleet_managed_key (fleet_managed_key)
`)
	if err != nil {
		return fmt.Errorf("adding policies.fleet_managed_key: %w", err)
	}

	return nil
}

func Down_20260805011013(tx *sql.Tx) error {
	return nil
}
