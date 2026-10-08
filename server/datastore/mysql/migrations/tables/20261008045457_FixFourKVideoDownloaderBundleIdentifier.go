package tables

import (
	"database/sql"
	"errors"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20261008045457, Down_20261008045457)
}

// The 4K Video Downloader macOS Fleet-maintained app used "com.4kdownload.ApplicationDirectories"
// as its unique identifier, which is the name of one of the app's preferences files. The app
// itself reports "com.openmedia.4kvideodownloader", so the FMA's software title never matched
// inventory and its automatic-install policy never passed.
//
// The catalog now ships the real identifier and fleet_maintained_apps self-heals on the next
// catalog sync. The software title an already-added installer is bound to, and the
// automatic-install policy query copied at add time, do not, so repair them here. This follows
// 20260810152924_FixDockerDesktopBundleIdentifier.
func Up_20261008045457(tx *sql.Tx) error {
	const (
		oldBundleID = "com.4kdownload.ApplicationDirectories"
		newBundleID = "com.openmedia.4kvideodownloader"
	)

	staleTitleID, err := fourKVideoDownloaderTitleID(tx, oldBundleID)
	if err != nil {
		return err
	}
	if staleTitleID == 0 {
		// Never added the FMA, or already migrated.
		return nil
	}

	staleInstallers, err := fourKVideoDownloaderInstallers(tx, staleTitleID)
	if err != nil {
		return err
	}

	// Only the exact query Fleet generated is rewritten; an admin-edited policy is left alone.
	for _, si := range staleInstallers {
		if _, err := tx.Exec(
			`UPDATE policies SET query = ?, updated_at = updated_at WHERE software_installer_id = ? AND query = ?`,
			fmt.Sprintf("SELECT 1 FROM apps WHERE bundle_identifier = '%s';", newBundleID),
			si.id,
			fmt.Sprintf("SELECT 1 FROM apps WHERE bundle_identifier = '%s';", oldBundleID),
		); err != nil {
			return fmt.Errorf("updating 4K Video Downloader automatic install policy query: %w", err)
		}
	}

	targetTitleID, err := fourKVideoDownloaderTitleID(tx, newBundleID)
	if err != nil {
		return err
	}

	if targetTitleID == 0 {
		// No host reported the app in inventory yet. Relabel the stale title in place so every
		// row already pointing at it stays correct.
		if _, err := tx.Exec(
			`UPDATE software_titles SET bundle_identifier = ? WHERE id = ?`,
			newBundleID, staleTitleID,
		); err != nil {
			return fmt.Errorf("relabeling 4K Video Downloader title: %w", err)
		}
		return nil
	}

	// Inventory already created the correct title, so merge the stale one into it. Teams that
	// already have an installer on the target title are skipped, as in the Docker Desktop
	// migration: moving the FMA installer would leave two installers on one title for one team.
	blockedTeams := make(map[uint]struct{})
	targetInstallers, err := fourKVideoDownloaderInstallers(tx, targetTitleID)
	if err != nil {
		return err
	}
	for _, si := range targetInstallers {
		blockedTeams[si.teamID] = struct{}{}
	}

	for _, si := range staleInstallers {
		if _, blocked := blockedTeams[si.teamID]; blocked {
			continue
		}
		if _, err := tx.Exec(
			`UPDATE software_installers SET title_id = ?, updated_at = updated_at WHERE id = ?`,
			targetTitleID, si.id,
		); err != nil {
			return fmt.Errorf("re-pointing 4K Video Downloader installer %d: %w", si.id, err)
		}
	}

	for _, t := range []titleRefColumn{
		{"host_software_installs", "software_title_id", true},
		{"software_install_upcoming_activities", "software_title_id", true},
		{"software", "title_id", false},
	} {
		if _, err := tx.Exec(
			fmt.Sprintf(`UPDATE %s SET %s = ?%s WHERE %s = ?`,
				t.table, t.column, t.preserveUpdatedAt(), t.column),
			targetTitleID, staleTitleID,
		); err != nil {
			return fmt.Errorf("re-pointing %s.%s: %w", t.table, t.column, err)
		}
	}

	// Patch policies are unique per (team, title). A team whose installer stayed behind can
	// already have one on the target, so IGNORE leaves its stale policy where it is (the stale
	// title then survives below, keeping that policy) instead of failing the migration.
	if _, err := tx.Exec(
		`UPDATE IGNORE policies SET patch_software_title_id = ?, updated_at = updated_at WHERE patch_software_title_id = ?`,
		targetTitleID, staleTitleID,
	); err != nil {
		return fmt.Errorf("re-pointing policies.patch_software_title_id: %w", err)
	}

	// Per-team settings and notification links are unique per title. UPDATE IGNORE moves the
	// ones the target title does not already have; whatever stays behind duplicates one the
	// target already carries, so drop it.
	for _, t := range []titleRefColumn{
		{"software_title_icons", "software_title_id", false},
		{"software_title_display_names", "software_title_id", false},
		{"software_title_team_pins", "title_id", true},
		{"software_update_schedules", "title_id", false},
		{"patch_notification_apps", "software_title_id", false},
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
		return fmt.Errorf("deleting stale 4K Video Downloader host counts: %w", err)
	}

	// Only drop the stale title once nothing depends on it: deleting it would null out a
	// remaining installer's title_id and cascade-delete a remaining patch policy.
	var remaining int
	if err := tx.QueryRow(`
		SELECT
			(SELECT COUNT(*) FROM software_installers WHERE title_id = ?) +
			(SELECT COUNT(*) FROM policies WHERE patch_software_title_id = ?)`,
		staleTitleID, staleTitleID,
	).Scan(&remaining); err != nil {
		return fmt.Errorf("counting remaining 4K Video Downloader dependents: %w", err)
	}
	if remaining > 0 {
		return nil
	}

	if _, err := tx.Exec(`DELETE FROM software_titles WHERE id = ?`, staleTitleID); err != nil {
		return fmt.Errorf("deleting stale 4K Video Downloader title: %w", err)
	}

	return nil
}

type fourKVideoDownloaderInstaller struct {
	id     uint
	teamID uint
}

func fourKVideoDownloaderInstallers(tx *sql.Tx, titleID uint) ([]fourKVideoDownloaderInstaller, error) {
	rows, err := tx.Query(
		`SELECT id, global_or_team_id FROM software_installers WHERE title_id = ?`, titleID,
	)
	if err != nil {
		return nil, fmt.Errorf("listing installers on title %d: %w", titleID, err)
	}
	defer rows.Close()

	var installers []fourKVideoDownloaderInstaller
	for rows.Next() {
		var si fourKVideoDownloaderInstaller
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

func fourKVideoDownloaderTitleID(tx *sql.Tx, bundleID string) (uint, error) {
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

func Down_20261008045457(tx *sql.Tx) error {
	return nil
}
