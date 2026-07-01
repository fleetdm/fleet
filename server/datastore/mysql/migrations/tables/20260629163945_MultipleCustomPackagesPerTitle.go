package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260629163945, Down_20260629163945)
}

func Up_20260629163945(tx *sql.Tx) error {
	// A title can now hold several packages. dedup_token drives the new unique key. Custom
	// rows resolve it to storage_id so they dedupe by content hash, letting different builds of
	// one version coexist. FMA rows resolve it to version, leaving the per-version rows that
	// back version pinning unchanged. VIRTUAL keeps the add in-place. The collation is pinned
	// to match storage_id and version so the migration matches what fresh installs get.
	if _, err := tx.Exec(`
		ALTER TABLE software_installers
			ADD COLUMN dedup_token VARCHAR(255) COLLATE utf8mb4_unicode_ci
				GENERATED ALWAYS AS (IF(fleet_maintained_app_id IS NULL, storage_id, version)) VIRTUAL
	`); err != nil {
		return fmt.Errorf("adding dedup_token column: %w", err)
	}

	// Collapse rows that would violate the new key: keep the lowest id per group and delete
	// the rest. Re-point policies off the deleted rows first, since
	// policies.software_installer_id is RESTRICT. Keep policies.updated_at so this
	// content-identical swap doesn't read as a policy edit.
	const dupGroups = `
		SELECT global_or_team_id, title_id, dedup_token, MIN(id) AS keep_id
		FROM software_installers
		WHERE title_id IS NOT NULL
		GROUP BY global_or_team_id, title_id, dedup_token
		HAVING COUNT(*) > 1`

	if _, err := tx.Exec(fmt.Sprintf(`
		UPDATE policies p
		JOIN software_installers si ON si.id = p.software_installer_id
		JOIN (%s) dup
			ON si.global_or_team_id = dup.global_or_team_id
			AND si.title_id = dup.title_id
			AND si.dedup_token = dup.dedup_token
		SET p.software_installer_id = dup.keep_id, p.updated_at = p.updated_at
		WHERE si.id != dup.keep_id`, dupGroups)); err != nil {
		return fmt.Errorf("re-pointing policies off duplicate installers: %w", err)
	}

	if _, err := tx.Exec(fmt.Sprintf(`
		DELETE si FROM software_installers si
		JOIN (%s) dup
			ON si.global_or_team_id = dup.global_or_team_id
			AND si.title_id = dup.title_id
			AND si.dedup_token = dup.dedup_token
		WHERE si.id != dup.keep_id`, dupGroups)); err != nil {
		return fmt.Errorf("deleting duplicate installers: %w", err)
	}

	if _, err := tx.Exec(`
		ALTER TABLE software_installers
			DROP INDEX idx_software_installers_team_title_version,
			ADD UNIQUE KEY idx_software_installers_dedup (global_or_team_id, title_id, dedup_token)
	`); err != nil {
		return fmt.Errorf("swapping software_installers unique key: %w", err)
	}

	return nil
}

func Down_20260629163945(tx *sql.Tx) error {
	return nil
}
