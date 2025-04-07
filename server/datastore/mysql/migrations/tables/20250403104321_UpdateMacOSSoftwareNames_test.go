package tables

import (
	"crypto/md5"
	"fmt"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/require"
)

func TestUp_20250403104321(t *testing.T) {
	db := applyUpToPrev(t)

	softwareTitles := []fleet.SoftwareTitle{
		{Name: "MacApp.app", Source: "apps", BundleIdentifier: ptr.String("com.example.foo")},
		{Name: "MacApp2.app", Source: "apps", BundleIdentifier: ptr.String("com.example.foo2")},
		{Name: "Chrome Extension", Source: "chrome_extensions", Browser: "chrome"},
		{Name: "Microsoft Teams.exe", Source: "programs"},
	}

	// insert some software titles
	dataStmt := `INSERT INTO software_titles (name, source, browser, bundle_identifier) VALUES (?, ?, ?, ?)`

	for _, s := range softwareTitles {
		_, err := db.Exec(dataStmt, s.Name, s.Source, s.Browser, s.BundleIdentifier)
		require.NoError(t, err)
	}

	// add some software entries
	dataStmt = `INSERT INTO software
		(name, version, source, bundle_identifier, ` + "`release`" + `, arch, vendor, browser, extension_id, checksum)
	VALUES
		(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	for i, st := range softwareTitles {
		var bid string
		if st.BundleIdentifier != nil {
			bid = *st.BundleIdentifier
		}
		execNoErr(t, db, dataStmt, st.Name, fmt.Sprint(i), st.Source, bid, "", "", "", st.Browser, "", fmt.Sprintf("foo%d", i))
	}

	// execNoErr(t, db, dataStmt, "Foo.app", "1.0", "apps", "com.ex", "", "", "", "", "", "foo1")
	// execNoErr(t, db, dataStmt, "Foo2.app", "2.0", "apps", "", "", "", "", "", "", "foo2")
	// execNoErr(t, db, dataStmt, "Chrome Extension", "3.0", "chrome_extensions", "", "", "", "", "", "", "foo3")
	// execNoErr(t, db, dataStmt, "Microsoft Teams.exe", "4.0", "programs", "", "", "", "", "", "",
	// "foo4")

	// Apply current migration.
	applyNext(t, db)

	// macOS apps should be modified, others should not

	err := db.Select(&softwareTitles, "SELECT name, source, browser, bundle_identifier FROM software_titles")
	require.NoError(t, err)

	expectedNames := map[string]struct{}{
		"MacApp":              {},
		"MacApp2":             {},
		"Chrome Extension":    {},
		"Microsoft Teams.exe": {},
	}

	for _, title := range softwareTitles {
		_, ok := expectedNames[title.Name]
		require.True(t, ok)
	}

	var software []fleet.Software
	for i, st := range softwareTitles {
		sw := fleet.Software{
			Name:    st.Name,
			Version: fmt.Sprint(i),
			Source:  st.Source,
			Browser: st.Browser,
		}
		if st.BundleIdentifier != nil {
			sw.BundleIdentifier = *st.BundleIdentifier
		}
		software = append(software, sw)
	}

	getChecksum := func(sw fleet.Software) []byte {
		h := md5.New() //nolint:gosec // This hash is used as a DB optimization for software row lookup, not security
		cols := []string{sw.Name, sw.Version, sw.Source, sw.BundleIdentifier, sw.Release, sw.Arch, sw.Vendor, sw.Browser, sw.ExtensionID}
		_, err := fmt.Fprint(h, strings.Join(cols, "\x00"))
		require.NoError(t, err)
		return h.Sum(nil)
	}

	expectedChecksums := map[string]string{
		"MacApp":              string(getChecksum(software[0])),
		"MacApp2":             string(getChecksum(software[1])),
		"Chrome Extension":    "foo2",
		"Microsoft Teams.exe": "foo3",
	}

	var gotSoftware []struct {
		Name     string `db:"name"`
		Checksum []byte `db:"checksum"`
	}

	err = db.Select(&gotSoftware, `SELECT name, checksum FROM software`)
	require.NoError(t, err)

	for _, sw := range gotSoftware {
		_, ok := expectedNames[sw.Name]
		require.True(t, ok)

		expectedCS, ok := expectedChecksums[sw.Name]
		require.True(t, ok)
		require.NotNil(t, sw.Checksum, "software without checksum: %s", sw.Name)
		require.Equal(t, expectedCS, string(sw.Checksum), "software with wrong checksum: %s", sw.Name)

	}
}
