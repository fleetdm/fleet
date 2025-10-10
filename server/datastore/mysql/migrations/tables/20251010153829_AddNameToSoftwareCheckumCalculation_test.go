package tables

import (
	"crypto/md5" //nolint:gosec // MD5 is used for checksums, not security
	"database/sql"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20251010153829(t *testing.T) {
	db := applyUpToPrev(t)

	computeOldChecksum := func(name, version, source, bundleID, release, arch, vendor, extensionFor, extensionID string) []byte {
		h := md5.New() //nolint:gosec
		cols := []string{version, source, bundleID, release, arch, vendor, extensionFor, extensionID}
		if source != "apps" {
			cols = append([]string{name}, cols...)
		}
		_, _ = fmt.Fprint(h, strings.Join(cols, "\x00"))
		return h.Sum(nil)
	}

	computeNewChecksum := func(name, version, source, bundleID, release, arch, vendor, extensionFor, extensionID string) []byte {
		h := md5.New() //nolint:gosec
		cols := []string{version, source, bundleID, release, arch, vendor, extensionFor, extensionID, name}
		_, _ = fmt.Fprint(h, strings.Join(cols, "\x00"))
		return h.Sum(nil)
	}

	insertTitle := `INSERT INTO software_titles (name, source, extension_for, bundle_identifier) VALUES (?, ?, ?, ?)`
	result, err := db.Exec(insertTitle, "Test App", "apps", "", "com.test.app")
	require.NoError(t, err)
	titleID, err := result.LastInsertId()
	require.NoError(t, err)

	insertSoftware := `INSERT INTO software
		(name, version, source, bundle_identifier, ` + "`release`" + `, arch, vendor, extension_for, extension_id, checksum, title_id)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	// software with bundle_identifier should be updated
	app1Name := "GoLand.app"
	app1BundleID := "com.jetbrains.goland"
	app1OldChecksum := computeOldChecksum(app1Name, "2023.1", "apps", app1BundleID, "", "x86_64", "JetBrains", "", "")
	app1NewChecksum := computeNewChecksum(app1Name, "2023.1", "apps", app1BundleID, "", "x86_64", "JetBrains", "", "")
	_, err = db.Exec(insertSoftware, app1Name, "2023.1", "apps", app1BundleID, "", "x86_64", "JetBrains", "", "", app1OldChecksum, titleID)
	require.NoError(t, err)

	app2Name := "GoLand 2.app"
	app2BundleID := "com.jetbrains.goland"
	app2OldChecksum := computeOldChecksum(app2Name, "2023.2", "apps", app2BundleID, "", "x86_64", "JetBrains", "", "")
	app2NewChecksum := computeNewChecksum(app2Name, "2023.2", "apps", app2BundleID, "", "x86_64", "JetBrains", "", "")
	_, err = db.Exec(insertSoftware, app2Name, "2023.2", "apps", app2BundleID, "", "x86_64", "JetBrains", "", "", app2OldChecksum, titleID)
	require.NoError(t, err)

	// software without bundle_identifier - no update
	app3Name := "SomeApp.app"
	app3OldChecksum := computeOldChecksum(app3Name, "1.0", "apps", "", "", "x86_64", "Vendor", "", "")
	_, err = db.Exec(insertSoftware, app3Name, "1.0", "apps", nil, "", "x86_64", "Vendor", "", "", app3OldChecksum, titleID)
	require.NoError(t, err)

	// Windows software - no update
	winName := "Notepad++"
	winOldChecksum := computeOldChecksum(winName, "8.5.0", "programs", "", "", "x86_64", "Don Ho", "", "")
	_, err = db.Exec(insertSoftware, winName, "8.5.0", "programs", nil, "", "x86_64", "Don Ho", "", "", winOldChecksum, titleID)
	require.NoError(t, err)

	// Linux software - no update
	linuxName := "vim"
	linuxOldChecksum := computeOldChecksum(linuxName, "8.2", "deb_packages", "", "1ubuntu1", "amd64", "Ubuntu", "", "")
	_, err = db.Exec(insertSoftware, linuxName, "8.2", "deb_packages", nil, "1ubuntu1", "amd64", "Ubuntu", "", "", linuxOldChecksum, titleID)
	require.NoError(t, err)

	applyNext(t, db)

	type softwareRow struct {
		Name             string         `db:"name"`
		Source           string         `db:"source"`
		BundleIdentifier sql.NullString `db:"bundle_identifier"`
		Checksum         []byte         `db:"checksum"`
	}

	var software []softwareRow
	err = db.Select(&software, `SELECT name, source, bundle_identifier, checksum FROM software ORDER BY name`)
	require.NoError(t, err)
	require.Len(t, software, 5)

	for _, sw := range software {
		switch sw.Name {
		case app1Name:
			require.Equal(t, app1NewChecksum, sw.Checksum)
			require.True(t, sw.BundleIdentifier.Valid)
			require.Equal(t, app1BundleID, sw.BundleIdentifier.String)
		case app2Name:
			require.Equal(t, app2NewChecksum, sw.Checksum)
			require.True(t, sw.BundleIdentifier.Valid)
			require.Equal(t, app2BundleID, sw.BundleIdentifier.String)
		case app3Name:
			require.Equal(t, app3OldChecksum, sw.Checksum)
			require.False(t, sw.BundleIdentifier.Valid)
		case winName:
			require.Equal(t, winOldChecksum, sw.Checksum)
			require.Equal(t, "programs", sw.Source)
		case linuxName:
			require.Equal(t, linuxOldChecksum, sw.Checksum)
			require.Equal(t, "deb_packages", sw.Source)
		default:
			t.Fatalf("Unexpected software entry: %s", sw.Name)
		}
	}
}
