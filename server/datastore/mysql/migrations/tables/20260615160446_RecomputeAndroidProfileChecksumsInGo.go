package tables

import (
	"bytes"
	"crypto/md5" // nolint:gosec // used only to hash for efficient comparisons, not security
	"database/sql"
	"encoding/json"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260615160446, Down_20260615160446)
}

// canonicalizeAndroidProfileJSON20260615160446 is a migration-local copy of the
// runtime canonicalizer (mysql.canonicalizeJSONForChecksum). It is duplicated so
// future changes to the runtime helper cannot alter this historical migration's
// behavior. It sorts object keys and strips insignificant whitespace; numbers are
// preserved as written and array order is kept.
func canonicalizeAndroidProfileJSON20260615160446(b []byte) ([]byte, error) {
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return nil, err
	}
	return json.Marshal(v)
}

// Up_20260615160446 switches the basis of mdm_android_configuration_profiles.checksum
// from md5(MySQL-normalized raw_json) — the value the original STORED generated
// column produced — to md5 of a Go-canonical JSON form. This lets the runtime
// compute the checksum in process (no per-write normalization round-trip to the
// DB). It recomputes both the desired (profile) checksums and the host copies, so
// that a profile whose content did not change is NOT flagged for re-delivery (the
// content is unchanged; only our checksum representation moved). A fresh install
// has empty tables, so this is a no-op there.
func Up_20260615160446(tx *sql.Tx) error {
	type androidProfile struct {
		uuid    string
		rawJSON []byte
	}
	var profiles []androidProfile
	rows, err := tx.Query(`SELECT profile_uuid, raw_json FROM mdm_android_configuration_profiles`)
	if err != nil {
		return fmt.Errorf("loading android profiles for checksum recompute: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var p androidProfile
		if err := rows.Scan(&p.uuid, &p.rawJSON); err != nil {
			return fmt.Errorf("scanning android profile for checksum recompute: %w", err)
		}
		profiles = append(profiles, p)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterating android profiles for checksum recompute: %w", err)
	}

	for _, p := range profiles {
		canonical, err := canonicalizeAndroidProfileJSON20260615160446(p.rawJSON)
		if err != nil {
			return fmt.Errorf("canonicalizing android profile %q for checksum recompute: %w", p.uuid, err)
		}
		sum := md5.Sum(canonical) // nolint:gosec
		if _, err := tx.Exec(`UPDATE mdm_android_configuration_profiles SET checksum = ? WHERE profile_uuid = ?`, sum[:], p.uuid); err != nil {
			return fmt.Errorf("recomputing android profile checksum: %w", err)
		}
	}

	// Re-point host checksums at the recomputed desired checksums so unchanged
	// profiles are not re-delivered. Orphan host rows (no matching profile) keep
	// their previous value; they are pending removal regardless.
	if _, err := tx.Exec(`
		UPDATE host_mdm_android_profiles hmap
			JOIN mdm_android_configuration_profiles macp ON macp.profile_uuid = hmap.profile_uuid
			SET hmap.checksum = macp.checksum`); err != nil {
		return fmt.Errorf("recomputing host android profile checksums: %w", err)
	}
	return nil
}

func Down_20260615160446(tx *sql.Tx) error {
	return nil
}
