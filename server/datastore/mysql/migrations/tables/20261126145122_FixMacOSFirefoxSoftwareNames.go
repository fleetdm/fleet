package tables

import (
	"bytes"
	"crypto/md5" //nolint:gosec // software checksums are a lookup key, not a security control
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/reflectx"
)

func init() {
	MigrationClient.AddMigration(Up_20261126145122, Down_20261126145122)
}

// The macos_firefox software override reported these names until it was changed to drop .app
// like every other macOS app.
var firefoxRenames20261126145122 = []struct{ old, new string }{
	{"Firefox.app", "Firefox"},
	{"Firefox ESR.app", "Firefox ESR"},
}

type firefoxSoftware20261126145122 struct {
	ID               uint64  `db:"id"`
	Name             string  `db:"name"`
	Version          string  `db:"version"`
	Source           string  `db:"source"`
	BundleIdentifier string  `db:"bundle_identifier"`
	Release          string  `db:"release"`
	Arch             string  `db:"arch"`
	Vendor           string  `db:"vendor"`
	ExtensionFor     string  `db:"extension_for"`
	ExtensionID      string  `db:"extension_id"`
	ApplicationID    *string `db:"application_id"`
	UpgradeCode      *string `db:"upgrade_code"`
	Checksum         []byte  `db:"checksum"`
}

// Up_20261126145122 moves the Firefox software rows the override created onto the names it now
// reports, so hosts keep linking to the same rows. Rows are found by checksum, not name: the
// Fleet-maintained app reconcile job renames rows without changing their checksum, so a row
// called "Mozilla Firefox" can still be keyed on "Firefox.app".
func Up_20261126145122(tx *sql.Tx) error {
	txx := sqlx.Tx{Tx: tx, Mapper: reflectx.NewMapperFunc("db", sqlx.NameMapper)}

	// One title per bundle identifier, so this touches at most one row.
	if _, err := tx.Exec(`
		UPDATE software_titles
		SET name = CASE name WHEN 'Firefox.app' THEN 'Firefox' ELSE 'Firefox ESR' END
		WHERE bundle_identifier = 'org.mozilla.firefox' AND source = 'apps'
			AND name IN ('Firefox.app', 'Firefox ESR.app')`); err != nil {
		return fmt.Errorf("renaming Firefox software titles: %w", err)
	}

	var rows []firefoxSoftware20261126145122
	if err := txx.Select(&rows, "SELECT id, name, version, source, bundle_identifier, `release`, arch, vendor, "+
		"extension_for, extension_id, application_id, upgrade_code, checksum "+
		"FROM software FORCE INDEX (idx_software_bundle_identifier) "+
		"WHERE bundle_identifier = 'org.mozilla.firefox' AND source = 'apps'"); err != nil {
		return fmt.Errorf("selecting Firefox software: %w", err)
	}

	type move struct {
		row         firefoxSoftware20261126145122
		oldName     string
		newName     string
		newChecksum []byte
	}
	var moves []move
	for _, r := range rows {
		for _, rename := range firefoxRenames20261126145122 {
			if bytes.Equal(r.Checksum, firefoxChecksum20261126145122(r, rename.old)) {
				moves = append(moves, move{r, rename.old, rename.new, firefoxChecksum20261126145122(r, rename.new)})
				break
			}
		}
	}

	var merged, renamed int
	for _, m := range moves {
		var survivorID uint64
		err := tx.QueryRow(`SELECT id FROM software WHERE checksum = ?`, m.newChecksum).Scan(&survivorID)
		switch {
		case errors.Is(err, sql.ErrNoRows):
			// Keep a name the reconcile job gave the row, only replace the reported one.
			if _, err := tx.Exec(`UPDATE software SET checksum = ?, name = IF(name = ?, ?, name) WHERE id = ?`,
				m.newChecksum, m.oldName, m.newName, m.row.ID); err != nil {
				return fmt.Errorf("renaming Firefox software %d: %w", m.row.ID, err)
			}
			renamed++
		case err != nil:
			return fmt.Errorf("looking up Firefox software for %d: %w", m.row.ID, err)
		default:
			if err := mergeFirefoxSoftware20261126145122(tx, m.row.ID, survivorID); err != nil {
				return err
			}
			merged++
		}
	}

	if renamed > 0 || merged > 0 {
		logger.Info.Printf("Fixed Firefox software names: renamed %d software rows, merged %d into existing rows", renamed, merged)
	}
	return nil
}

// mergeFirefoxSoftware20261126145122 moves every host on the stale software row onto the
// survivor, then deletes the stale row. software_cpe and software_metadata cascade, and the host
// count and vulnerability crons recompute the survivor's counts and CVEs.
func mergeFirefoxSoftware20261126145122(tx *sql.Tx, staleID, survivorID uint64) error {
	// host_software_installed_paths is only indexed on (host_id, software_id), so reach it
	// through host_software's software_id index. A host already on the survivor keeps the paths
	// it reported for it, and its paths for the stale row are deleted below.
	if _, err := tx.Exec(`
		UPDATE host_software hs
		STRAIGHT_JOIN host_software_installed_paths hsip FORCE INDEX (host_id_software_id_idx)
			ON hsip.host_id = hs.host_id AND hsip.software_id = hs.software_id
		LEFT JOIN host_software keep ON keep.host_id = hs.host_id AND keep.software_id = ?
		SET hsip.software_id = ?
		WHERE hs.software_id = ? AND keep.host_id IS NULL`, survivorID, survivorID, staleID); err != nil {
		return fmt.Errorf("moving installed paths from software %d to %d: %w", staleID, survivorID, err)
	}
	if _, err := tx.Exec(`
		DELETE hsip FROM host_software hs
		STRAIGHT_JOIN host_software_installed_paths hsip FORCE INDEX (host_id_software_id_idx)
			ON hsip.host_id = hs.host_id AND hsip.software_id = hs.software_id
		WHERE hs.software_id = ?`, staleID); err != nil {
		return fmt.Errorf("deleting installed paths of software %d: %w", staleID, err)
	}
	if _, err := tx.Exec(`
		INSERT IGNORE INTO host_software (host_id, software_id, last_opened_at)
		SELECT host_id, ?, last_opened_at FROM host_software WHERE software_id = ?`, survivorID, staleID); err != nil {
		return fmt.Errorf("moving hosts from software %d to %d: %w", staleID, survivorID, err)
	}
	for _, table := range []string{"host_software", "software_cve", "software_host_counts"} {
		if _, err := tx.Exec(`DELETE FROM `+table+` WHERE software_id = ?`, staleID); err != nil { //nolint:gosec // table names are constants
			return fmt.Errorf("deleting %s of software %d: %w", table, staleID, err)
		}
	}
	if _, err := tx.Exec(`DELETE FROM software WHERE id = ?`, staleID); err != nil {
		return fmt.Errorf("deleting software %d: %w", staleID, err)
	}
	return nil
}

// firefoxChecksum20261126145122 is fleet.Software.ComputeRawChecksum as of this migration, for
// the row reported under the given name. macOS apps never have an RPM epoch.
func firefoxChecksum20261126145122(r firefoxSoftware20261126145122, name string) []byte {
	cols := []string{r.Version, r.Source, r.BundleIdentifier, r.Release, r.Arch, r.Vendor, r.ExtensionFor, r.ExtensionID, name}
	if r.ApplicationID != nil && *r.ApplicationID != "" {
		cols = append(cols, *r.ApplicationID)
	}
	if r.UpgradeCode != nil && *r.UpgradeCode != "" {
		cols = append(cols, *r.UpgradeCode)
	}
	sum := md5.Sum([]byte(strings.Join(cols, "\x00"))) //nolint:gosec // see import
	return sum[:]
}

func Down_20261126145122(tx *sql.Tx) error {
	return nil
}
