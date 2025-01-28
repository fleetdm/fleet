package tables

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/require"
)

func TestUp_20240709124958(t *testing.T) {
	db := applyUpToPrev(t)

	dataStmts := `
	  INSERT INTO script_contents (id, md5_checksum, contents) VALUES
	    (1, 'checksum', 'script content');

	  INSERT INTO software_titles (id, name, source, browser) VALUES
	    (1, 'Foo.app', 'apps', ''),
	    (2, 'Foo dupe.app', 'apps', ''),
	    (3, 'Chrome Extension', 'chrome_extensions', 'chrome'),
	    (4, 'Go', 'deb_packages', ''),
	    (5, 'Microsoft Teams.exe', 'programs', ''),
	    (6, 'Safari extension', 'safari_extensions', 'safari'),
	    (7, 'Go', 'rpm_packages', ''),
	    (8, 'Bar.app', 'apps', ''),
	    (9, 'Bar from installer.app', 'apps', ''),
	    (10, 'Fizz.app', 'apps', ''),
	    (11, 'Fizz from installer with ref.app', 'apps', '');

	  INSERT INTO software (checksum, name, version, source, browser, bundle_identifier, title_id) VALUES
	    ('checksum_01', 'Foo.app', '1.1', 'apps', '', 'com.example.foo', 1),
	    ('checksum_02', 'Foo dupe.app', '1.1', 'apps', '', 'com.example.foo', 2),
	    ('checksum_03', 'Chrome Extension', '1.1', 'chrome_extensions', 'chrome', '', 3),
	    ('checksum_04', 'Go', '1.1', 'deb_packages', '', '', 4),
	    ('checksum_05', 'Microsoft Teams.exe', '1.1', 'programs', '', '', 5),
	    ('checksum_06', 'Safari extension', '1.1', 'safari_extensions', 'safari', '', 6),
	    ('checksum_07', 'Go', '1.1', 'rpm_packages', '', '', 7),
	    ('checksum_08', 'Bar.app', '1.1', 'apps', '', 'com.example.bar', 8),
	    ('checksum_09', 'Fiz.app', '1.1', 'apps', '', 'com.example.fizz', 11),
	    ('checksum_10', 'Fiz.app', '2.2', 'apps', '', 'com.example.fizz', 10),
	    ('checksum_11', 'No title.app', '2.2', 'apps', '', 'com.example.notitle', NULL);

	  INSERT INTO software_installers
	    (id, title_id, filename, version, platform, install_script_content_id, storage_id)
	  VALUES
	    (1, 9, 'bar-installer.pkg', '1.2', 'darwin', 1, 'storage-id'),
	    (2, 11, 'fizz-installer.pkg', '1.2', 'darwin', 1, 'storage-id');
	`

	_, err := db.Exec(dataStmts)
	require.NoError(t, err)
	applyNext(t, db)

	type softwareTitle struct {
		Name             string `db:"name"`
		Source           string `db:"source"`
		Browser          string `db:"browser"`
		BundleIdentifier string `db:"bundle_identifier"`
	}

	var titles []softwareTitle
	err = db.Select(&titles, `SELECT name, source, browser, COALESCE(bundle_identifier, '') as bundle_identifier FROM software_titles`)
	require.NoError(t, err)
	require.ElementsMatch(t, []softwareTitle{
		{"Foo.app", "apps", "", "com.example.foo"},
		{"Chrome Extension", "chrome_extensions", "chrome", ""},
		{"Go", "deb_packages", "", ""},
		{"Microsoft Teams.exe", "programs", "", ""},
		{"Safari extension", "safari_extensions", "safari", ""},
		{"Go", "rpm_packages", "", ""},
		{"Bar.app", "apps", "", "com.example.bar"},
		{"Bar from installer.app", "apps", "", ""},
		{"Fizz.app", "apps", "", "com.example.fizz"},
	}, titles)

	type softwareInstaller struct {
		TitleID  uint   `db:"title_id"`
		Filename string `db:"filename"`
		Version  string `db:"version"`
		Platform string `db:"platform"`
	}

	var installers []softwareInstaller
	err = db.Select(&installers, `SELECT title_id, filename, version, platform FROM software_installers`)
	require.NoError(t, err)
	require.ElementsMatch(t, []softwareInstaller{
		{9, "bar-installer.pkg", "1.2", "darwin"},
		{10, "fizz-installer.pkg", "1.2", "darwin"},
	}, installers)

	type softwareRow struct {
		Name             string `db:"name"`
		Version          string `db:"version"`
		Source           string `db:"source"`
		Browser          string `db:"browser"`
		BundleIdentifier string `db:"bundle_identifier"`
		TitleID          *uint  `db:"title_id"`
	}

	var software []softwareRow
	err = db.Select(&software, `SELECT name, version, source, browser, COALESCE(bundle_identifier, '') as bundle_identifier, title_id FROM software`)
	require.NoError(t, err)
	require.ElementsMatch(t, []softwareRow{
		{"Foo.app", "1.1", "apps", "", "com.example.foo", ptr.Uint(1)},
		{"Foo dupe.app", "1.1", "apps", "", "com.example.foo", ptr.Uint(2)},
		{"Chrome Extension", "1.1", "chrome_extensions", "chrome", "", ptr.Uint(3)},
		{"Go", "1.1", "deb_packages", "", "", ptr.Uint(4)},
		{"Microsoft Teams.exe", "1.1", "programs", "", "", ptr.Uint(5)},
		{"Safari extension", "1.1", "safari_extensions", "safari", "", ptr.Uint(6)},
		{"Go", "1.1", "rpm_packages", "", "", ptr.Uint(7)},
		{"Bar.app", "1.1", "apps", "", "com.example.bar", ptr.Uint(8)},
		{"Fiz.app", "1.1", "apps", "", "com.example.fizz", ptr.Uint(11)},
		{"Fiz.app", "2.2", "apps", "", "com.example.fizz", ptr.Uint(10)},
		{"No title.app", "2.2", "apps", "", "com.example.notitle", nil},
	}, software)
}
