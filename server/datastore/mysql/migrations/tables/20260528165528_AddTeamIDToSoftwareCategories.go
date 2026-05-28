package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20260528165528, Down_20260528165528)
}

func Up_20260528165528(tx *sql.Tx) error {
	return withSteps([]migrationStep{
		// Column changes:
		//   - widen `name` from VARCHAR(63) to VARCHAR(255) to match the API docs
		//   - add `team_id NOT NULL DEFAULT 0` so existing seeded rows become the
		//     "Unassigned"/team_id=0 set without changing their IDs
		//   - add created_at / updated_at timestamps
		basicMigrationStep(`
ALTER TABLE software_categories
	MODIFY COLUMN name VARCHAR(255) NOT NULL,
	ADD COLUMN team_id INT UNSIGNED NOT NULL DEFAULT 0,
	ADD COLUMN created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	ADD COLUMN updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
`, "adding team scoping columns to software_categories"),

		// Index changes: drop the old name-only unique index (names will repeat
		// across teams) and add the new (team_id, name) unique index.
		basicMigrationStep(`
ALTER TABLE software_categories
	DROP INDEX idx_software_categories_name,
	ADD UNIQUE KEY idx_software_categories_team_id_name (team_id, name)
`, "swapping uniqueness scope to (team_id, name)"),

		// Rename the previously seeded defaults to their emoji-prefixed forms.
		// IDs are preserved.
		basicMigrationStep(`
UPDATE software_categories
SET name = CASE name
	WHEN 'Browsers'         THEN '🌎 Browsers'
	WHEN 'Communication'    THEN '👬 Communication'
	WHEN 'Developer tools'  THEN '🧰 Developer tools'
	WHEN 'Productivity'     THEN '💻 Productivity'
	WHEN 'Security'         THEN '🔐 Security'
	WHEN 'Utilities'        THEN '🛟 Support'
	ELSE name
END
WHERE team_id = 0
`, "renaming default categories to emoji-prefixed names"),

		// Backfill: give every existing fleet its own copy of the 6 defaults.
		// CROSS JOIN against teams produces one row per (fleet, default name).
		// Ordering by t.id then by FIELD(...) enforces canonical name order so
		// each fleet's 6 new rows land in a contiguous, sequential block of
		// auto-increment IDs — same order new fleets get via the team-creation
		// hook (which iterates fleet.DefaultSelfServiceCategoryNames).
		basicMigrationStep(`
INSERT INTO software_categories (name, team_id)
SELECT sc.name, t.id
FROM software_categories sc
CROSS JOIN teams t
WHERE sc.team_id = 0
ORDER BY t.id, FIELD(sc.name,
	'🌎 Browsers',
	'👬 Communication',
	'🧰 Developer tools',
	'💻 Productivity',
	'🔐 Security',
	'🛟 Support')
`, "backfilling per-fleet default categories"),

		// Re-point each link from the team_id=0 source row to the team's row
		// with the same name (joining old_sc and new_sc on name).
		basicMigrationStep(`
UPDATE software_installer_software_categories sisc
JOIN software_installers si ON sisc.software_installer_id = si.id
JOIN software_categories old_sc ON sisc.software_category_id = old_sc.id
JOIN software_categories new_sc ON new_sc.team_id = si.global_or_team_id AND new_sc.name = old_sc.name
SET sisc.software_category_id = new_sc.id
WHERE si.global_or_team_id != 0 AND old_sc.team_id = 0
`, "re-pointing software installer category links"),

		// Same for VPP app category links.
		basicMigrationStep(`
UPDATE vpp_app_team_software_categories vatsc
JOIN vpp_apps_teams vat ON vatsc.vpp_app_team_id = vat.id
JOIN software_categories old_sc ON vatsc.software_category_id = old_sc.id
JOIN software_categories new_sc ON new_sc.team_id = vat.global_or_team_id AND new_sc.name = old_sc.name
SET vatsc.software_category_id = new_sc.id
WHERE vat.global_or_team_id != 0 AND old_sc.team_id = 0
`, "re-pointing VPP app category links"),

		// Same for in-house app category links.
		basicMigrationStep(`
UPDATE in_house_app_software_categories ihasc
JOIN in_house_apps iha ON ihasc.in_house_app_id = iha.id
JOIN software_categories old_sc ON ihasc.software_category_id = old_sc.id
JOIN software_categories new_sc ON new_sc.team_id = iha.global_or_team_id AND new_sc.name = old_sc.name
SET ihasc.software_category_id = new_sc.id
WHERE iha.global_or_team_id != 0 AND old_sc.team_id = 0
`, "re-pointing in-house app category links"),
	}, tx)
}

func Down_20260528165528(tx *sql.Tx) error {
	return nil
}
