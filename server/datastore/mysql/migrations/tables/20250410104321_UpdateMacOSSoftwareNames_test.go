package tables

import (
	"crypto/md5" // nolint:gosec // used only to hash for efficient comparisons
	"fmt"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func computeRawChecksumIncludingName(sw fleet.Software) ([]byte, error) {
	h := md5.New() //nolint:gosec // This hash is used as a DB optimization for software row lookup, not security
	cols := []string{sw.Name, sw.Version, sw.Source, sw.BundleIdentifier, sw.Release, sw.Arch, sw.Vendor, sw.Browser, sw.ExtensionID}
	_, err := fmt.Fprint(h, strings.Join(cols, "\x00"))
	if err != nil {
		return nil, err
	}
	return h.Sum(nil), nil
}

func TestUp_20250410104321(t *testing.T) {
	db := applyUpToPrev(t)

	// 16 pieces of software, 14 (pre-dedupe) of which are macOS apps, 11 of which are unique (relative to software table)
	// Each piece of software is on a different host, other than MacApp Duplicate 3, which is on the same host
	// as another of the MacApps (same bundle ID). This means we'll start with 16 host_software entries and
	// expect one of those entries to go away.
	softwares := []fleet.Software{
		{Name: "MacApp.app", Source: "apps", BundleIdentifier: "com.example.foo", Version: "1"},
		{Name: "MacApp Duplicate.app", Source: "apps", BundleIdentifier: "com.example.foo", Version: "1"},
		{Name: "MacApp Duplicate 2.app", Source: "apps", BundleIdentifier: "com.example.foo", Version: "1"},
		{Name: "MacApp Duplicate 3.app", Source: "apps", BundleIdentifier: "com.example.foo", Version: "1"},
		{Name: "no_bundle_id.app", Source: "apps", BundleIdentifier: "", Version: "42"},
		{Name: "no_bundle_id_2.app", Source: "apps", BundleIdentifier: "", Version: "24"},
		{Name: "MacApp2.app", Source: "apps", BundleIdentifier: "com.example.foo2", Version: "2"},
		{Name: "MacApp2 2.app", Source: "apps", BundleIdentifier: "com.example.foo2", Version: "2"},
		{Name: "MacApp2.1.app", Source: "apps", BundleIdentifier: "com.example.foo2", Version: "2.1"},   // should be a different software post-migration
		{Name: "MacApp2.1 2.app", Source: "apps", BundleIdentifier: "com.example.foo2", Version: "2.1"}, // should be the same software as the line above
		{Name: "Chrome Extension", Source: "chrome_extensions", Browser: "chrome", Version: "3"},
		{Name: "Microsoft Teams.exe", Source: "programs", Version: "4"},
		{Name: "Live Captions.app", Source: "apps", BundleIdentifier: "com.apple.accessibility.LiveTranscriptionAgent", Version: "1.0"},
		{Name: "LiveTranscriptionAgent.app", Source: "apps", BundleIdentifier: "com.apple.accessibility.LiveTranscriptionAgent", Version: "1.0"},
		{Name: "Postman Helper (Renderer).app", Source: "apps", BundleIdentifier: "com.postmanlabs.mac.helper", Version: ""},
		{Name: "Postman Helper.app", Source: "apps", BundleIdentifier: "com.postmanlabs.mac.helper", Version: ""},
	}

	// add some software titles
	dataStmt := `INSERT INTO software_titles (name, source, browser, bundle_identifier) VALUES (?, ?, ?, ?)`
	for i, s := range softwares {
		if (i > 0 && s.BundleIdentifier == "com.example.foo") ||
			s.Name == "LiveTranscriptionAgent.app" ||
			s.Name == "Postman Helper.app" ||
			s.Version == "2.1" || s.Name == "MacApp2 2.app" {
			continue
		}
		var bid any = ptr.String(s.BundleIdentifier)
		if s.BundleIdentifier == "" {
			bid = nil
		}
		id := execNoErrLastID(t, db, dataStmt, s.Name, s.Source, s.Browser, bid)
		if s.BundleIdentifier == "com.example.foo" { // All the initial duplicates should map to the same title ID
			for i := range 4 {
				softwares[i].TitleID = ptr.Uint(uint(id)) //nolint:gosec // dismiss G115
			}
			continue
		}

		softwares[i].TitleID = ptr.Uint(uint(id)) //nolint:gosec // dismiss G115

		// More duplicate mapping to existing software title ID
		if s.BundleIdentifier == "com.example.foo2" {
			softwares[8].TitleID = ptr.Uint(uint(id)) //nolint:gosec // dismiss G115
		}
		if s.BundleIdentifier == "com.apple.accessibility.LiveTranscriptionAgent" {
			softwares[12].TitleID = ptr.Uint(uint(id)) //nolint:gosec // dismiss G115
		}
		if s.BundleIdentifier == "com.postmanlabs.mac.helper" {
			softwares[14].TitleID = ptr.Uint(999) // throwing in a broken title ID to reflect some deployed environments
		}
	}

	// add some software entries and host_software entries
	dataStmt = `INSERT INTO software
		(name, version, source, bundle_identifier, ` + "`release`" + `, arch, vendor, browser, extension_id, checksum, title_id)
	VALUES
		(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	var softwareIDs []uint
	for i, s := range softwares {
		checksum, err := computeRawChecksumIncludingName(softwares[i])
		require.NoError(t, err)

		id := execNoErrLastID(
			t,
			db,
			dataStmt,
			s.Name, s.Version, s.Source, s.BundleIdentifier, "", "", "", s.Browser, "",
			checksum,
			s.TitleID,
		)
		softwareIDs = append(softwareIDs, uint(id)) //nolint:gosec // dismiss G115
		softwares[i].ID = uint(id)                  //nolint:gosec // dismiss G115

		hostID := uint(i + 1) //nolint:gosec // dismiss G115
		if s.Name == "MacApp Duplicate 3.app" {
			// Map to the same host software
			hostID = uint(i) //nolint:gosec // dismiss G115
		}
		execNoErr(t, db, "INSERT INTO host_software (host_id, software_id) VALUES (?, ?)", hostID, uint(id)) //nolint:gosec // dismiss G115

		// insert installed paths for macOS apps to make sure all get migrated over
		if s.Source == "apps" {
			execNoErr(t, db, "INSERT INTO host_software_installed_paths (host_id, software_id, installed_path) VALUES (?, ?, ?)",
				hostID,
				uint(id), //nolint:gosec // dismiss G115
				"/Applications/"+s.Name,
			)
		}
	}

	noBundleID1 := softwares[4]
	noBundleID2 := softwares[5]

	// add some software_cve entries
	cveStmt := `INSERT INTO software_cve (cve, software_id) VALUES %s`
	cveStmt = fmt.Sprintf(cveStmt, strings.TrimRight(strings.Repeat("(?, ?),", len(softwareIDs)), ","))
	var args []any
	for _, id := range softwareIDs {
		args = append(args, uuid.NewString(), id)
	}
	_, err := db.Exec(cveStmt, args...)
	require.NoError(t, err)

	// Apply current migration.
	applyNext(t, db)

	// macOS apps should be modified, others should not

	var gotSoftware []fleet.Software
	err = db.Select(&gotSoftware, `SELECT id, name, checksum, name_source, version FROM software ORDER BY id ASC`)
	require.NoError(t, err)
	require.Len(t, gotSoftware, 9)

	var gotSoftwareTitles []fleet.SoftwareTitle
	err = db.Select(&gotSoftwareTitles, "SELECT id, name, source, browser, bundle_identifier FROM software_titles")
	require.NoError(t, err)
	require.Len(t, gotSoftwareTitles, 8) // two versions of MacApp2

	for _, got := range gotSoftwareTitles {
		switch got.ID {
		case *noBundleID1.TitleID:
			require.Equal(t, noBundleID1.Name, got.Name)
		case *noBundleID2.TitleID:
			require.Equal(t, noBundleID2.Name, got.Name)
		default:
			require.NotContains(t, got.Name, ".app")
		}
	}

	for _, got := range gotSoftware {
		switch got.ID {
		case noBundleID1.ID:
			require.Equal(t, noBundleID1.Name, got.Name)
		case noBundleID2.ID:
			require.Equal(t, noBundleID2.Name, got.Name)
		default:
			require.NotContains(t, got.Name, ".app")
			require.Equal(t, "basic", got.NameSource)
		}
	}

	// Rows in the set are MacApp (duplicates deleted), no_bundle_id, no_bundle_id_2, MacApp2 v2, MacApp2 v2.1, etc.
	require.Equal(t, "MacApp2", gotSoftware[3].Name)
	require.Equal(t, "2", gotSoftware[3].Version)
	require.Equal(t, "MacApp2.1", gotSoftware[4].Name)
	require.Equal(t, "2.1", gotSoftware[4].Version)

	var count int
	err = db.Get(&count, "SELECT COUNT(*) FROM software_cve")
	require.NoError(t, err)
	require.Equal(t, 9, count)

	err = db.Get(&count, "SELECT COUNT(*) FROM host_software")
	require.NoError(t, err)
	require.Equal(t, 15, count)

	// ensure no orphaned host software installed paths
	err = db.Get(&count, `SELECT COUNT(*) FROM host_software_installed_paths WHERE software_id NOT IN (SELECT id FROM software)`)
	require.NoError(t, err)
	require.Zero(t, count)

	err = db.Get(&count, `SELECT COUNT(*) FROM host_software_installed_paths WHERE software_id IN (SELECT id FROM software)`)
	require.NoError(t, err)
	require.Equal(t, 14, count) // one per install; one host has two paths

	// ensure we have the expected number of unique software IDs on host software installed paths (same as software count with source apps)
	err = db.Get(&count, `SELECT COUNT(DISTINCT software_id) FROM host_software_installed_paths`)
	require.NoError(t, err)
	require.Equal(t, 7, count)

	// ensure both copies of the same app are listed in installed paths for the host that has this case
	var getSoftwarePaths []fleet.HostSoftwareInstalledPath
	err = db.Select(&getSoftwarePaths, "SELECT host_id, software_id, installed_path FROM host_software_installed_paths WHERE host_id = 3 ORDER BY installed_path")
	require.NoError(t, err)
	require.Len(t, getSoftwarePaths, 2)
	require.Equal(t, getSoftwarePaths[0].SoftwareID, getSoftwarePaths[1].SoftwareID)
	require.Equal(t, "/Applications/MacApp Duplicate 2.app", getSoftwarePaths[0].InstalledPath)
	require.Equal(t, "/Applications/MacApp Duplicate 3.app", getSoftwarePaths[1].InstalledPath)

	err = db.Get(&count, `SELECT COUNT(*) FROM host_software WHERE software_id IN (?, ?)`, softwareIDs[1], softwareIDs[2])
	require.NoError(t, err)
	require.Zero(t, count)

	err = db.Get(&count, `SELECT COUNT(*) FROM host_software WHERE software_id = ?`, softwareIDs[0])
	require.NoError(t, err)
	require.Equal(t, count, 3)

	var hostIDs []uint

	// ensure MacApp2 v2 has the expected (two) hosts associated
	err = db.Select(&hostIDs, `SELECT host_id FROM host_software JOIN software ON software.id = host_software.software_id
			WHERE bundle_identifier = "com.example.foo2" AND version = "2" ORDER BY host_id`)
	require.NoError(t, err)
	require.Len(t, hostIDs, 2)
	require.Equal(t, uint(7), hostIDs[0])
	require.Equal(t, uint(8), hostIDs[1])

	// ensure installed paths map from the correct host to the correct software for MacApp2 v2
	err = db.Select(&getSoftwarePaths, `SELECT host_id, software_id, installed_path FROM host_software_installed_paths
    		JOIN software ON software.id = host_software_installed_paths.software_id
			WHERE bundle_identifier = "com.example.foo2" AND version = "2" ORDER BY host_id`)
	require.NoError(t, err)
	require.Len(t, getSoftwarePaths, 2)
	require.Equal(t, uint(7), getSoftwarePaths[0].HostID)
	require.Equal(t, gotSoftware[3].ID, getSoftwarePaths[0].SoftwareID)
	require.Equal(t, "/Applications/MacApp2.app", getSoftwarePaths[0].InstalledPath)
	require.Equal(t, uint(8), getSoftwarePaths[1].HostID)
	require.Equal(t, gotSoftware[3].ID, getSoftwarePaths[1].SoftwareID)
	require.Equal(t, "/Applications/MacApp2 2.app", getSoftwarePaths[1].InstalledPath)

	// ensure MacApp2 v2.1 has the expected (two) hosts associated
	err = db.Select(&hostIDs, `SELECT host_id FROM host_software JOIN software ON software.id = host_software.software_id
			WHERE bundle_identifier = "com.example.foo2" AND version = "2.1" ORDER BY host_id`)
	require.NoError(t, err)
	require.Len(t, hostIDs, 2)
	require.Equal(t, uint(9), hostIDs[0])
	require.Equal(t, uint(10), hostIDs[1])

	// ensure installed paths map from the correct host to the correct software for MacApp2 v2.1
	err = db.Select(&getSoftwarePaths, `SELECT host_id, software_id, installed_path FROM host_software_installed_paths
    		JOIN software ON software.id = host_software_installed_paths.software_id
			WHERE bundle_identifier = "com.example.foo2" AND version = "2.1" ORDER BY host_id`)
	require.NoError(t, err)
	require.Len(t, getSoftwarePaths, 2)
	require.Equal(t, uint(9), getSoftwarePaths[0].HostID)
	require.Equal(t, gotSoftware[4].ID, getSoftwarePaths[0].SoftwareID)
	require.Equal(t, "/Applications/MacApp2.1.app", getSoftwarePaths[0].InstalledPath)
	require.Equal(t, uint(10), getSoftwarePaths[1].HostID)
	require.Equal(t, gotSoftware[4].ID, getSoftwarePaths[1].SoftwareID)
	require.Equal(t, "/Applications/MacApp2.1 2.app", getSoftwarePaths[1].InstalledPath)
}
