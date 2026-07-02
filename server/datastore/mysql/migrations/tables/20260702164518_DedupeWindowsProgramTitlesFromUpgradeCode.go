package tables

import (
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/reflectx"
)

func init() {
	MigrationClient.AddMigration(Up_20260702164518, Down_20260702164518)
}

func Up_20260702164518(tx *sql.Tx) error {
	txx := sqlx.Tx{Tx: tx, Mapper: reflectx.NewMapperFunc("db", sqlx.NameMapper)}

	// Windows programs were duplicated into two software_titles for the same name: one backed by an
	// installer and one reported by a host, where exactly one of the two carries the MSI
	// upgrade_code. This happened when the gitops/FMA batch add matched titles by unique_identifier
	// instead of name. Collapse each pair into the installer-backed title.
	//
	// keep = the installer-backed title (the one admins manage); when both titles have an installer,
	//        keep the one that carries the upgrade_code.
	// drop = the same-named title being merged away.
	const dupePairs = `
		SELECT
			keep.id AS keep_id,
			keep.name AS name,
			dropped.id AS drop_id,
			COALESCE(NULLIF(keep.upgrade_code, ''), dropped.upgrade_code) AS upgrade_code
		FROM software_titles keep
		-- pair each title with another title in the same name/source/extension group
		JOIN software_titles dropped
			ON dropped.name = keep.name
			AND dropped.source = keep.source
			AND dropped.extension_for = keep.extension_for
			AND dropped.id != keep.id
		WHERE keep.source = 'programs' AND keep.extension_for = ''
			-- keep is the title we merge into, so it must have an installer
			AND EXISTS (SELECT 1 FROM software_installers si WHERE si.title_id = keep.id)
			-- if dropped also has an installer, keep the upgrade-code title (picks the survivor; the pair is still merged)
			AND (NOT EXISTS (SELECT 1 FROM software_installers si WHERE si.title_id = dropped.id) OR (keep.upgrade_code IS NOT NULL AND keep.upgrade_code != ''))
			-- exactly one of the two titles carries a non-empty upgrade code
			AND (keep.upgrade_code IS NOT NULL AND keep.upgrade_code != '') != (dropped.upgrade_code IS NOT NULL AND dropped.upgrade_code != '')
			-- no team has an installer under both titles, which would collide when moved
			AND NOT EXISTS (SELECT 1 FROM software_installers ki JOIN software_installers di ON di.global_or_team_id = ki.global_or_team_id WHERE ki.title_id = keep.id AND di.title_id = dropped.id)
			-- the name group is exactly these two titles
			AND (SELECT COUNT(*) FROM software_titles g WHERE g.name = keep.name AND g.source = keep.source AND g.extension_for = keep.extension_for) = 2`

	var pairs []struct {
		KeepID      uint   `db:"keep_id"`
		Name        string `db:"name"`
		DropID      uint   `db:"drop_id"`
		UpgradeCode string `db:"upgrade_code"`
	}
	if err := txx.Select(&pairs, dupePairs); err != nil {
		return fmt.Errorf("selecting duplicate Windows program titles: %w", err)
	}

	// Re-point everything that references the drop title over to the keep title. The plain UPDATEs
	// have no unique key on the title column. policies (patch) and software_update_schedules cascade
	// on title delete, so they must move before the delete below. The UPDATE IGNOREs have a unique
	// key on (team, title) where a drop row can collide with an existing keep row; move what fits
	// and let the drop-title delete cascade away any leftovers.
	repoint := []string{
		`UPDATE software SET title_id = ? WHERE title_id = ?`,
		`UPDATE software_installers SET title_id = ? WHERE title_id = ?`,
		`UPDATE host_software_installs SET software_title_id = ? WHERE software_title_id = ?`,
		`UPDATE software_install_upcoming_activities SET software_title_id = ? WHERE software_title_id = ?`,
		`UPDATE policies SET patch_software_title_id = ? WHERE patch_software_title_id = ?`,
		`UPDATE software_update_schedules SET title_id = ? WHERE title_id = ?`,
		`UPDATE IGNORE software_title_display_names SET software_title_id = ? WHERE software_title_id = ?`,
		`UPDATE IGNORE software_title_icons SET software_title_id = ? WHERE software_title_id = ?`,
		`UPDATE IGNORE software_title_team_pins SET title_id = ? WHERE title_id = ?`,
	}

	for _, p := range pairs {
		for _, stmt := range repoint {
			if _, err := tx.Exec(stmt, p.KeepID, p.DropID); err != nil {
				return fmt.Errorf("re-pointing references off duplicate title %q: %w", p.Name, err)
			}
		}

		// Delete the drop title before setting the keep's upgrade_code, otherwise the two collide on
		// the (unique_identifier, source, extension_for) key.
		if _, err := tx.Exec(`DELETE FROM software_titles WHERE id = ?`, p.DropID); err != nil {
			return fmt.Errorf("deleting duplicate title %q: %w", p.Name, err)
		}
		if _, err := tx.Exec(`UPDATE software_titles SET upgrade_code = ? WHERE id = ?`, p.UpgradeCode, p.KeepID); err != nil {
			return fmt.Errorf("setting upgrade_code on kept title %q: %w", p.Name, err)
		}

		fmt.Printf("Deduplicated Windows program title %q: merged software title %d into %d (upgrade_code %s)\n", p.Name, p.DropID, p.KeepID, p.UpgradeCode)
	}

	return nil
}

func Down_20260702164518(tx *sql.Tx) error {
	return nil
}
