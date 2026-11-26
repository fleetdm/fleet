package tables

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

func TestUp_20261126145122(t *testing.T) {
	db := applyUpToPrev(t)

	firefox := func(name, version string) fleet.Software {
		return fleet.Software{Name: name, Version: version, Source: "apps", BundleIdentifier: "org.mozilla.firefox"}
	}
	checksum := func(sw fleet.Software) []byte {
		sum, err := sw.ComputeRawChecksum()
		require.NoError(t, err)
		return sum
	}
	titleID := execNoErrLastID(t, db,
		`INSERT INTO software_titles (name, source, bundle_identifier) VALUES ('Firefox.app', 'apps', 'org.mozilla.firefox')`)
	otherTitleID := execNoErrLastID(t, db,
		`INSERT INTO software_titles (name, source, bundle_identifier) VALUES ('Thunderbird', 'apps', 'org.mozilla.thunderbird')`)
	// storedName is what the row is called now, reported is what the host reported when the row
	// was created, which is what its checksum is computed from.
	insert := func(storedName string, reported fleet.Software) int64 {
		return execNoErrLastID(t, db,
			`INSERT INTO software (name, version, source, bundle_identifier, title_id, checksum) VALUES (?, ?, ?, ?, ?, ?)`,
			storedName, reported.Version, reported.Source, reported.BundleIdentifier, titleID, checksum(reported))
	}
	linkHost := func(hostID uint, softwareID int64) {
		execNoErr(t, db, `INSERT INTO host_software (host_id, software_id) VALUES (?, ?)`, hostID, softwareID)
		execNoErr(t, db, `INSERT INTO host_software_installed_paths (host_id, software_id, installed_path) VALUES (?, ?, '/Applications/Firefox.app')`,
			hostID, softwareID)
	}
	addCVE := func(softwareID int64, cve string) {
		execNoErr(t, db, `INSERT INTO software_cve (software_id, cve) VALUES (?, ?)`, softwareID, cve)
		execNoErr(t, db, `INSERT INTO software_cpe (software_id, cpe) VALUES (?, 'cpe:2.3:a:mozilla:firefox')`, softwareID)
		execNoErr(t, db, `INSERT INTO software_host_counts (software_id, hosts_count, team_id, global_stats) VALUES (?, 1, 0, 1)`, softwareID)
	}

	// Reported only by hosts with parse_ini: renamed in place.
	onlyApp := insert("Firefox.app", firefox("Firefox.app", "120.0"))
	linkHost(1, onlyApp)
	addCVE(onlyApp, "CVE-2026-0001")
	onlyESRApp := insert("Firefox ESR.app", firefox("Firefox ESR.app", "128.0"))
	linkHost(1, onlyESRApp)

	// Reported by hosts with and without parse_ini: merged into the row without .app.
	dupApp := insert("Firefox.app", firefox("Firefox.app", "121.0"))
	dup := insert("Firefox", firefox("Firefox", "121.0"))
	linkHost(1, dupApp)
	linkHost(2, dupApp)
	linkHost(2, dup)
	linkHost(3, dup)
	addCVE(dupApp, "CVE-2026-0002")
	addCVE(dup, "CVE-2026-0003")

	// Renamed to the Fleet-maintained app's name, with a checksum still computed from .app.
	fmaApp := insert("Mozilla Firefox", firefox("Firefox.app", "122.0"))
	linkHost(1, fmaApp)
	fmaDupApp := insert("Mozilla Firefox", firefox("Firefox.app", "123.0"))
	fmaDup := insert("Mozilla Firefox", firefox("Firefox", "123.0"))
	linkHost(1, fmaDupApp)
	linkHost(2, fmaDup)

	// Not reported by the override query.
	plain := insert("Firefox", firefox("Firefox", "124.0"))
	other := execNoErrLastID(t, db,
		`INSERT INTO software (name, version, source, bundle_identifier, title_id, checksum) VALUES ('Thunderbird.app', '1.0', 'apps', 'org.mozilla.thunderbird', ?, ?)`,
		otherTitleID, checksum(fleet.Software{Name: "Thunderbird.app", Version: "1.0", Source: "apps", BundleIdentifier: "org.mozilla.thunderbird"}))

	applyNext(t, db)

	type row struct {
		Name     string `db:"name"`
		Checksum []byte `db:"checksum"`
	}
	software := func(id int64) *row {
		var rows []row
		require.NoError(t, db.Select(&rows, `SELECT name, checksum FROM software WHERE id = ?`, id))
		if len(rows) == 0 {
			return nil
		}
		return &rows[0]
	}
	ids := func(query string, args ...any) []int64 {
		var out []int64
		require.NoError(t, db.Select(&out, query, args...))
		return out
	}
	strs := func(query string, args ...any) []string {
		var out []string
		require.NoError(t, db.Select(&out, query, args...))
		return out
	}

	// Renamed in place, so hosts keep the software ID and its CVEs.
	require.Equal(t, &row{"Firefox", checksum(firefox("Firefox", "120.0"))}, software(onlyApp))
	require.Equal(t, []int64{1}, ids(`SELECT host_id FROM host_software WHERE software_id = ?`, onlyApp))
	require.Equal(t, []string{"CVE-2026-0001"}, strs(`SELECT cve FROM software_cve WHERE software_id = ?`, onlyApp))
	require.Equal(t, &row{"Firefox ESR", checksum(firefox("Firefox ESR", "128.0"))}, software(onlyESRApp))

	// Merged: the survivor gets every host and path, the duplicate and its data are gone.
	require.Nil(t, software(dupApp))
	require.Equal(t, &row{"Firefox", checksum(firefox("Firefox", "121.0"))}, software(dup))
	require.Equal(t, []int64{1, 2, 3}, ids(`SELECT host_id FROM host_software WHERE software_id = ? ORDER BY host_id`, dup))
	// Host 2 was on both rows, so it keeps the path it reported for the survivor only.
	require.Equal(t, []int64{1, 2, 3}, ids(`SELECT host_id FROM host_software_installed_paths WHERE software_id = ? ORDER BY host_id`, dup))
	require.Equal(t, []string{"CVE-2026-0003"}, strs(`SELECT cve FROM software_cve WHERE software_id = ?`, dup))
	for _, table := range []string{"host_software", "host_software_installed_paths", "software_cve", "software_cpe", "software_host_counts"} {
		require.Empty(t, ids(`SELECT software_id FROM `+table+` WHERE software_id = ?`, dupApp), table)
	}

	// The Fleet-maintained app's name is kept, only the checksum moves to the new reported name.
	require.Equal(t, &row{"Mozilla Firefox", checksum(firefox("Firefox", "122.0"))}, software(fmaApp))
	require.Nil(t, software(fmaDupApp))
	require.Equal(t, []int64{1, 2}, ids(`SELECT host_id FROM host_software WHERE software_id = ? ORDER BY host_id`, fmaDup))

	require.Equal(t, &row{"Firefox", checksum(firefox("Firefox", "124.0"))}, software(plain))
	require.Equal(t, "Thunderbird.app", software(other).Name)

	require.Equal(t, []string{"Firefox"}, strs(`SELECT name FROM software_titles WHERE id = ?`, titleID))
	require.Equal(t, []string{"Thunderbird"}, strs(`SELECT name FROM software_titles WHERE id = ?`, otherTitleID))
}

func TestUp_20261126145122_ESRTitle(t *testing.T) {
	db := applyUpToPrev(t)

	titleID := execNoErrLastID(t, db,
		`INSERT INTO software_titles (name, source, bundle_identifier) VALUES ('Firefox ESR.app', 'apps', 'org.mozilla.firefox')`)

	applyNext(t, db)

	var name string
	require.NoError(t, db.Get(&name, `SELECT name FROM software_titles WHERE id = ?`, titleID))
	require.Equal(t, "Firefox ESR", name)
}
