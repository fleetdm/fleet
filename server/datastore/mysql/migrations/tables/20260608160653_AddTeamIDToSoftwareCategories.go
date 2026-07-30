package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20260608160653, Down_20260608160653)
}

func Up_20260608160653(tx *sql.Tx) error {
	return withSteps([]migrationStep{
		// Table update: add team scoping columns, swap unique index, rename defaults.
		func(tx *sql.Tx) error {
			if _, err := tx.Exec(`
ALTER TABLE software_categories
	MODIFY COLUMN name VARCHAR(255) NOT NULL,
	ADD COLUMN team_id INT UNSIGNED NOT NULL DEFAULT 0,
	ADD COLUMN created_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
	ADD COLUMN updated_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6)
`); err != nil {
				return errors.Wrap(err, "adding team scoping columns to software_categories")
			}

			// Index changes: drop the old name-only unique index (names will repeat
			// across teams) and add the new (team_id, name) unique index.
			if _, err := tx.Exec(`
ALTER TABLE software_categories
	DROP INDEX idx_software_categories_name,
	ADD UNIQUE KEY idx_software_categories_team_id_name (team_id, name)
`); err != nil {
				return errors.Wrap(err, "swapping uniqueness scope to (team_id, name)")
			}

			// Rename the previously seeded defaults to their emoji-prefixed forms.
			// IDs are preserved.
			if _, err := tx.Exec(`
UPDATE software_categories
SET name = CASE name
	WHEN 'Browsers'         THEN '🌎 Browsers'
	WHEN 'Communication'    THEN '👬 Communication'
	WHEN 'Developer tools'  THEN '🧰 Developer tools'
	WHEN 'Productivity'     THEN '🖥️ Productivity'
	WHEN 'Security'         THEN '🔐 Security'
	WHEN 'Utilities'        THEN '🛠️ Utilities'
	ELSE name
END
WHERE team_id = 0
`); err != nil {
				return errors.Wrap(err, "renaming default categories to emoji-prefixed names")
			}
			// Pin both timestamps to a constant so the generated schema is
			// deterministic across repeated runs of make dump-test-schema.
			// The previous UPDATE bumped updated_at via ON UPDATE CURRENT_TIMESTAMP
			// and the ADD COLUMN filled created_at with the migration run time.
			if _, err := tx.Exec(`UPDATE software_categories SET created_at = '2026-05-29 00:00:00', updated_at = '2026-05-29 00:00:00' WHERE team_id = 0`); err != nil {
				return errors.Wrap(err, "pinning timestamps for schema dump stability")
			}
			return nil
		},

		// Backfill: copy defaults per fleet and re-point existing category links.
		func(tx *sql.Tx) error {
			// give every existing fleet its own copy of the 6 defaults.
			if _, err := tx.Exec(`
INSERT INTO software_categories (name, team_id)
SELECT sc.name, t.id
FROM software_categories sc
CROSS JOIN teams t
WHERE sc.team_id = 0
ORDER BY t.id, FIELD(sc.name,
	'🌎 Browsers',
	'👬 Communication',
	'🧰 Developer tools',
	'🖥️ Productivity',
	'🔐 Security',
	'🛠️ Utilities')
`); err != nil {
				return errors.Wrap(err, "backfilling per-fleet default categories")
			}

			// Re-point each link from the team_id=0 source row to the team's row
			// with the same name (joining old_sc and new_sc on name).
			if _, err := tx.Exec(`
UPDATE software_installer_software_categories sisc
JOIN software_installers si ON sisc.software_installer_id = si.id
JOIN software_categories old_sc ON sisc.software_category_id = old_sc.id
JOIN software_categories new_sc ON new_sc.team_id = si.global_or_team_id AND new_sc.name = old_sc.name
SET sisc.software_category_id = new_sc.id
WHERE si.global_or_team_id != 0 AND old_sc.team_id = 0
`); err != nil {
				return errors.Wrap(err, "re-pointing software installer category links")
			}

			// Same for VPP app category links.
			if _, err := tx.Exec(`
UPDATE vpp_app_team_software_categories vatsc
JOIN vpp_apps_teams vat ON vatsc.vpp_app_team_id = vat.id
JOIN software_categories old_sc ON vatsc.software_category_id = old_sc.id
JOIN software_categories new_sc ON new_sc.team_id = vat.global_or_team_id AND new_sc.name = old_sc.name
SET vatsc.software_category_id = new_sc.id
WHERE vat.global_or_team_id != 0 AND old_sc.team_id = 0
`); err != nil {
				return errors.Wrap(err, "re-pointing VPP app category links")
			}

			// Same for in-house app category links.
			if _, err := tx.Exec(`
UPDATE in_house_app_software_categories ihasc
JOIN in_house_apps iha ON ihasc.in_house_app_id = iha.id
JOIN software_categories old_sc ON ihasc.software_category_id = old_sc.id
JOIN software_categories new_sc ON new_sc.team_id = iha.global_or_team_id AND new_sc.name = old_sc.name
SET ihasc.software_category_id = new_sc.id
WHERE iha.global_or_team_id != 0 AND old_sc.team_id = 0
`); err != nil {
				return errors.Wrap(err, "re-pointing in-house app category links")
			}
			return nil
		},
	}, tx)
}

func Down_20260608160653(tx *sql.Tx) error {
	return nil
}
