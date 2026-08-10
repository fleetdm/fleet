package tables

import (
	"database/sql"
	"errors"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260810152924, Down_20260810152924)
}

// The Docker Desktop macOS Fleet-maintained app used "com.electron.dockerdesktop" as its
// unique identifier. That identifier belongs to the embedded Electron bundle at
// "/Applications/Docker.app/Contents/MacOS/Docker Desktop.app"; the installed app is
// "/Applications/Docker.app", which reports "com.docker.docker". Since the software
// inventory query started excluding embedded bundles (path NOT LIKE '%.app/Contents/%'),
// nothing in inventory carries the embedded identifier, so the FMA's software title had no
// installed versions: no installed version, no "Installed" status, and "Install" offered on
// hosts that already had Docker Desktop.
//
// The catalog itself now ships "com.docker.docker" and fleet_maintained_apps self-heals on
// the next catalog sync. What does not self-heal is the software title an already-added
// Docker Desktop installer is bound to, so re-point it here.
func Up_20260810152924(tx *sql.Tx) error {
	const (
		oldBundleID = "com.electron.dockerdesktop"
		newBundleID = "com.docker.docker"
	)

	staleTitleID, err := dockerDesktopTitleID(tx, oldBundleID)
	if err != nil {
		return err
	}
	if staleTitleID == 0 {
		// Never added the FMA, or already migrated.
		return nil
	}

	targetTitleID, err := dockerDesktopTitleID(tx, newBundleID)
	if err != nil {
		return err
	}

	if targetTitleID == 0 {
		// No title carries the correct identifier yet (no host reported Docker Desktop in
		// inventory). Relabel the stale title in place so every row already pointing at it
		// stays correct.
		if _, err := tx.Exec(
			`UPDATE software_titles SET bundle_identifier = ? WHERE id = ?`,
			newBundleID, staleTitleID,
		); err != nil {
			return fmt.Errorf("relabeling Docker Desktop title: %w", err)
		}
		return nil
	}

	// Inventory already created the correct title, so merge the stale one into it.
	//
	// Teams that already have an installer on the target title are skipped: moving the FMA
	// installer there would leave two installers on one title for one team, which the
	// service layer does not expect (the dedup key is (team, title, dedup_token), and an
	// FMA's dedup_token is its version, so it would not collide). Those teams keep the
	// pre-migration state rather than having data silently reshaped.
	//
	// Which installers move is decided in Go and applied one row at a time: this is an
	// UPDATE of software_installers that has to consult software_installers, which MySQL
	// only tolerates via a guaranteed-materialized derived table, and the row count here is
	// at most one installer per team.
	blockedTeams, err := dockerDesktopInstallerTeams(tx, targetTitleID)
	if err != nil {
		return err
	}
	staleInstallers, err := dockerDesktopInstallers(tx, staleTitleID)
	if err != nil {
		return err
	}

	for _, si := range staleInstallers {
		if _, blocked := blockedTeams[si.teamID]; blocked {
			continue
		}
		if _, err := tx.Exec(
			`UPDATE software_installers SET title_id = ?, updated_at = updated_at WHERE id = ?`,
			targetTitleID, si.id,
		); err != nil {
			return fmt.Errorf("re-pointing Docker Desktop installer %d: %w", si.id, err)
		}
	}

	// Install history and queued installs have no title-scoped unique key, so they move
	// wholesale.
	for _, t := range []titleRefColumn{
		{"host_software_installs", "software_title_id", true},
		{"software_install_upcoming_activities", "software_title_id", true},
		{"software", "title_id", false},
		{"policies", "patch_software_title_id", true},
	} {
		if _, err := tx.Exec(
			fmt.Sprintf(`UPDATE %s SET %s = ?%s WHERE %s = ?`,
				t.table, t.column, t.preserveUpdatedAt(), t.column),
			targetTitleID, staleTitleID,
		); err != nil {
			return fmt.Errorf("re-pointing %s.%s: %w", t.table, t.column, err)
		}
	}

	// Per-team settings are unique per (team, title). UPDATE IGNORE moves the ones the
	// target title does not already have; whatever stays behind is a duplicate of a setting
	// the target already carries, so drop it.
	for _, t := range []titleRefColumn{
		{"software_title_icons", "software_title_id", false},
		{"software_title_display_names", "software_title_id", false},
		{"software_title_team_pins", "title_id", true},
		{"software_update_schedules", "title_id", false},
	} {
		if _, err := tx.Exec(
			fmt.Sprintf(`UPDATE IGNORE %s SET %s = ?%s WHERE %s = ?`,
				t.table, t.column, t.preserveUpdatedAt(), t.column),
			targetTitleID, staleTitleID,
		); err != nil {
			return fmt.Errorf("re-pointing %s.%s: %w", t.table, t.column, err)
		}
		if _, err := tx.Exec(
			fmt.Sprintf(`DELETE FROM %s WHERE %s = ?`, t.table, t.column), staleTitleID,
		); err != nil {
			return fmt.Errorf("cleaning up %s.%s: %w", t.table, t.column, err)
		}
	}

	// Host counts are recomputed by the cron that owns them.
	if _, err := tx.Exec(
		`DELETE FROM software_titles_host_counts WHERE software_title_id = ?`, staleTitleID,
	); err != nil {
		return fmt.Errorf("deleting stale Docker Desktop host counts: %w", err)
	}

	// Only drop the stale title once nothing depends on it. The FK is ON DELETE SET NULL,
	// so deleting it while an installer still points at it would orphan that installer.
	var remaining int
	if err := tx.QueryRow(
		`SELECT COUNT(*) FROM software_installers WHERE title_id = ?`, staleTitleID,
	).Scan(&remaining); err != nil {
		return fmt.Errorf("counting remaining Docker Desktop installers: %w", err)
	}
	if remaining > 0 {
		return nil
	}

	if _, err := tx.Exec(`DELETE FROM software_titles WHERE id = ?`, staleTitleID); err != nil {
		return fmt.Errorf("deleting stale Docker Desktop title: %w", err)
	}

	return nil
}

// titleRefColumn is a column pointing at a software title that has to follow the merge.
// hasUpdatedAt marks the tables whose updated_at is ON UPDATE CURRENT_TIMESTAMP: this
// migration re-points a foreign key rather than modifying the record, so those timestamps
// are assigned to themselves to keep MySQL from stamping them.
type titleRefColumn struct {
	table        string
	column       string
	hasUpdatedAt bool
}

func (t titleRefColumn) preserveUpdatedAt() string {
	if !t.hasUpdatedAt {
		return ""
	}
	return ", updated_at = updated_at"
}

type dockerDesktopInstaller struct {
	id     uint
	teamID uint
}

// dockerDesktopInstallers returns the software installers attached to the given title.
func dockerDesktopInstallers(tx *sql.Tx, titleID uint) ([]dockerDesktopInstaller, error) {
	rows, err := tx.Query(
		`SELECT id, global_or_team_id FROM software_installers WHERE title_id = ?`, titleID,
	)
	if err != nil {
		return nil, fmt.Errorf("listing installers on title %d: %w", titleID, err)
	}
	defer rows.Close()

	var installers []dockerDesktopInstaller
	for rows.Next() {
		var si dockerDesktopInstaller
		if err := rows.Scan(&si.id, &si.teamID); err != nil {
			return nil, fmt.Errorf("scanning installer on title %d: %w", titleID, err)
		}
		installers = append(installers, si)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating installers on title %d: %w", titleID, err)
	}
	return installers, nil
}

// dockerDesktopInstallerTeams returns the set of teams that already have an installer on the
// given title.
func dockerDesktopInstallerTeams(tx *sql.Tx, titleID uint) (map[uint]struct{}, error) {
	installers, err := dockerDesktopInstallers(tx, titleID)
	if err != nil {
		return nil, err
	}
	teams := make(map[uint]struct{}, len(installers))
	for _, si := range installers {
		teams[si.teamID] = struct{}{}
	}
	return teams, nil
}

// dockerDesktopTitleID returns the macOS app title for the given bundle identifier, or 0 if
// there is none.
func dockerDesktopTitleID(tx *sql.Tx, bundleID string) (uint, error) {
	var id uint
	err := tx.QueryRow(
		`SELECT id FROM software_titles WHERE source = 'apps' AND bundle_identifier = ?`,
		bundleID,
	).Scan(&id)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return 0, nil
	case err != nil:
		return 0, fmt.Errorf("looking up software title for %s: %w", bundleID, err)
	}
	return id, nil
}

func Down_20260810152924(tx *sql.Tx) error {
	return nil
}
