package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260629163945, Down_20260629163945)
}

func Up_20260629163945(tx *sql.Tx) error {
	// A title may now have more than one package. dedup_token makes uniqueness depend on
	// the kind of package: custom rows resolve it to storage_id so they dedupe by content
	// hash, so Arm and Intel of one version coexist while identical bytes are rejected.
	// FMA rows resolve it to version, so version-uniqueness is unchanged and the same
	// bytes can back several versions. A title holds only one kind, and a hash never
	// equals a version string, so the two token spaces don't collide. The column is
	// VIRTUAL so the add is in-place and its only consumer is the unique key below. Its
	// collation is pinned to match storage_id and version so the migration path does not
	// inherit the server default collation that fresh installs never see.
	if _, err := tx.Exec(`
		ALTER TABLE software_installers
			ADD COLUMN dedup_token VARCHAR(255) COLLATE utf8mb4_unicode_ci
				GENERATED ALWAYS AS (IF(fleet_maintained_app_id IS NULL, storage_id, version)) VIRTUAL
	`); err != nil {
		return fmt.Errorf("adding dedup_token column: %w", err)
	}

	// Where a (global_or_team_id, title_id, dedup_token) has more than one row, keep the
	// first-added (smallest id) as the survivor and delete the rest, so the unique key
	// below can be added. This collapses duplicate-active custom rows and any custom
	// same-hash duplicates. FMA rows already satisfy version-uniqueness. Re-point policies
	// off the deleted rows first, since policies.software_installer_id is RESTRICT.
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
		SET p.software_installer_id = dup.keep_id
		WHERE si.id <> dup.keep_id`, dupGroups)); err != nil {
		return fmt.Errorf("re-pointing policies off duplicate installers: %w", err)
	}

	if _, err := tx.Exec(fmt.Sprintf(`
		DELETE si FROM software_installers si
		JOIN (%s) dup
			ON si.global_or_team_id = dup.global_or_team_id
			AND si.title_id = dup.title_id
			AND si.dedup_token = dup.dedup_token
		WHERE si.id <> dup.keep_id`, dupGroups)); err != nil {
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
