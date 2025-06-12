package tables

import (
	"fmt"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestUp_20250410104321(t *testing.T) {
	db := applyUpToPrev(t)

	softwares := []fleet.Software{
		{Name: "MacApp.app", Source: "apps", BundleIdentifier: "com.example.foo", Version: "1"},
		{Name: "MacApp Duplicate.app", Source: "apps", BundleIdentifier: "com.example.foo", Version: "1"},
		{Name: "MacApp Duplicate 2.app", Source: "apps", BundleIdentifier: "com.example.foo", Version: "1"},
		{Name: "MacApp Duplicate 3.app", Source: "apps", BundleIdentifier: "com.example.foo", Version: "1"},
		{Name: "no_bundle_id.app", Source: "apps", BundleIdentifier: "", Version: "42"},
		{Name: "no_bundle_id_2.app", Source: "apps", BundleIdentifier: "", Version: "24"},
		{Name: "MacApp2.app", Source: "apps", BundleIdentifier: "com.example.foo2", Version: "2"},
		{Name: "Chrome Extension", Source: "chrome_extensions", Browser: "chrome", Version: "3"},
		{Name: "Microsoft Teams.exe", Source: "programs", Version: "4"},
	}

	// add some software titles
	dataStmt := `INSERT INTO software_titles (name, source, browser, bundle_identifier) VALUES (?, ?, ?, ?)`

	for i, s := range softwares {
		if i > 0 && s.BundleIdentifier == "com.example.foo" {
			continue
		}
		var bid any = ptr.String(s.BundleIdentifier)
		if s.BundleIdentifier == "" {
			bid = nil
		}
		id := execNoErrLastID(t, db, dataStmt, s.Name, s.Source, s.Browser, bid)
		if s.BundleIdentifier == "com.example.foo" {
			// All the duplicates should map to the same title ID
			for i := range 4 {
				softwares[i].TitleID = ptr.Uint(uint(id)) //nolint:gosec // dismiss G115
			}
			continue
		}

		softwares[i].TitleID = ptr.Uint(uint(id)) //nolint:gosec // dismiss G115
	}

	// add some software entries and host_software entries
	dataStmt = `INSERT INTO software
		(name, version, source, bundle_identifier, ` + "`release`" + `, arch, vendor, browser, extension_id, checksum, title_id)
	VALUES
		(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	var softwareIDs []uint
	for i, s := range softwares {
		id := execNoErrLastID(t, db, dataStmt, s.Name, s.Version, s.Source, s.BundleIdentifier, "", "", "", s.Browser, "", fmt.Sprintf("foo%d", i), s.TitleID)
		softwareIDs = append(softwareIDs, uint(id)) //nolint:gosec // dismiss G115
		softwares[i].ID = uint(id)                  //nolint:gosec // dismiss G115

		hostID := uint(i + 1) //nolint:gosec // dismiss G115
		if s.Name == "MacApp Duplicate 3.app" {
			// Map to the same host software
			hostID = uint(i) //nolint:gosec // dismiss G115
		}
		execNoErr(t, db, "INSERT INTO host_software (host_id, software_id) VALUES (?, ?)", hostID, uint(id)) //nolint:gosec // dismiss G115
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
	err = db.Select(&gotSoftware, `SELECT id, name, checksum, name_source FROM software`)
	require.NoError(t, err)
	require.Len(t, gotSoftware, 6)

	var gotSoftwareTitles []fleet.SoftwareTitle
	err = db.Select(&gotSoftwareTitles, "SELECT id, name, source, browser, bundle_identifier FROM software_titles")
	require.NoError(t, err)
	require.Len(t, gotSoftwareTitles, 6)

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
			require.NotContains(t, got.Name, ".app")
		}
	}

	var count int
	err = db.Get(&count, "SELECT COUNT(*) FROM software_cve")
	require.NoError(t, err)
	require.Equal(t, 6, count)

	err = db.Get(&count, "SELECT COUNT(*) FROM host_software")
	require.NoError(t, err)
	require.Equal(t, 8, count)

	err = db.Get(&count, `SELECT COUNT(*) FROM host_software WHERE software_id IN (?, ?)`, softwareIDs[1], softwareIDs[2])
	require.NoError(t, err)
	require.Zero(t, count)

	err = db.Get(&count, `SELECT COUNT(*) FROM host_software WHERE software_id = ?`, softwareIDs[0])
	require.NoError(t, err)
	require.Equal(t, count, 3)
}
