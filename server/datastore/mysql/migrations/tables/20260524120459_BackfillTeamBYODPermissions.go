package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260524120459, Down_20260524120459)
}

func Up_20260524120459(tx *sql.Tx) error {
	// Backfill allow_byod_wipe and allow_byod_lock on existing teams' MDM
	// config. These fields are plain bool in TeamMDM and JSON-decode to false
	// when absent — pre-PR teams predate the keys, so without this backfill
	// every existing team would silently switch to "wipe and lock are not
	// allowed" the moment the new BYOD permission gate goes live. Default
	// true matches the new-team initialization path in ee/server/service/teams.go
	// and the global AppConfig MarshalJSON default.
	//
	// JSON_SET inserts when the key is absent and replaces when it is
	// present. Restrict by JSON_EXTRACT IS NULL so explicitly-set values
	// (e.g. from a gitops apply between this PR's migrations) are preserved.
	_, err := tx.Exec(`
		UPDATE teams
		SET config = JSON_SET(config, '$.mdm.allow_byod_wipe', CAST('true' AS JSON))
		WHERE JSON_EXTRACT(config, '$.mdm.allow_byod_wipe') IS NULL
	`)
	if err != nil {
		return fmt.Errorf("backfill teams.config.mdm.allow_byod_wipe: %w", err)
	}

	_, err = tx.Exec(`
		UPDATE teams
		SET config = JSON_SET(config, '$.mdm.allow_byod_lock', CAST('true' AS JSON))
		WHERE JSON_EXTRACT(config, '$.mdm.allow_byod_lock') IS NULL
	`)
	if err != nil {
		return fmt.Errorf("backfill teams.config.mdm.allow_byod_lock: %w", err)
	}

	return nil
}

func Down_20260524120459(tx *sql.Tx) error {
	return nil
}
