package tables

import (
	"fmt"
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

	// add some software titles
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

	// Apply current migration.
	applyNext(t, db)

	// macOS apps should be modified, others should not

	var gotSoftware []struct {
		Name     string `db:"name"`
		Checksum []byte `db:"checksum"`
	}

	err := db.Select(&gotSoftware, `SELECT name, checksum FROM software`)
	require.NoError(t, err)

	err = db.Select(&softwareTitles, "SELECT name, source, browser, bundle_identifier FROM software_titles")
	require.NoError(t, err)

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

	macCS1, err := software[0].ComputeRawChecksum()
	require.NoError(t, err)
	macCS2, err := software[1].ComputeRawChecksum()
	require.NoError(t, err)

	expectedNames := map[string]string{
		"MacApp":              string(macCS1),
		"MacApp2":             string(macCS2),
		"Chrome Extension":    "foo2",
		"Microsoft Teams.exe": "foo3",
	}

	for _, title := range softwareTitles {
		_, ok := expectedNames[title.Name]
		require.True(t, ok)
	}

	for _, got := range gotSoftware {
		expectedCS, ok := expectedNames[got.Name]
		require.True(t, ok)
		require.NotNil(t, got.Checksum, "software without checksum: %s", got.Name)
		require.Len(t, got.Checksum, 16) // it's a BINARY value so it's right-padded with 0s
		require.Equal(t, expectedCS, string(got.Checksum[:len(expectedCS)]), "software with wrong checksum: %s", got.Name)

	}
}
