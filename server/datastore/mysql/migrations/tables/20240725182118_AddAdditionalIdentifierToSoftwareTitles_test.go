package tables

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUp_20240725182118(t *testing.T) {
	db := applyUpToPrev(t)

	// Data before ios and ipados apps were added
	dataStmt := `
	  INSERT INTO software_titles (id, name, source, browser, bundle_identifier) VALUES
	    (1, 'Foo.app', 'apps', '', 'com.example.foo'),
	    (2, 'Foo2.app', 'apps', '', 'com.example.foo2'),
	    (3, 'Chrome Extension', 'chrome_extensions', 'chrome', NULL),
	    (4, 'Microsoft Teams.exe', 'programs', '', NULL);
	`

	_, err := db.Exec(dataStmt)
	require.NoError(t, err)
	applyNext(t, db)

	type softwareTitle struct {
		Name                 string  `db:"name"`
		Source               string  `db:"source"`
		Browser              string  `db:"browser"`
		BundleIdentifier     *string `db:"bundle_identifier"`
		AdditionalIdentifier *uint32 `db:"additional_identifier"`
	}

	var titles []softwareTitle
	err = db.Select(&titles, `SELECT name, source, browser, bundle_identifier, additional_identifier FROM software_titles`)
	require.NoError(t, err)
	zero := uint32(0)
	expectedTitles := []softwareTitle{
		{"Foo.app", "apps", "", ptr.String("com.example.foo"), &zero},
		{"Foo2.app", "apps", "", ptr.String("com.example.foo2"), &zero},
		{"Chrome Extension", "chrome_extensions", "chrome", nil, nil},
		{"Microsoft Teams.exe", "programs", "", nil, nil},
	}
	assert.ElementsMatch(t, expectedTitles, titles)

	// Ensure that the unique key is enforced
	dataStmt = `
	  INSERT INTO software_titles (id, name, source, browser, bundle_identifier) VALUES
	    (100, 'Foo3', 'foo', '', 'com.example.foo');
	`
	_, err = db.Exec(dataStmt)
	assert.ErrorContains(t, err, "Duplicate entry")

	// Add ios and ipados apps
	dataStmt = `
	  INSERT INTO software_titles (id, name, source, browser, bundle_identifier) VALUES
	    (5, 'Foo', 'ios_apps', '', 'com.example.foo'),
	    (6, 'Foo', 'ipados_apps', '', 'com.example.foo'),
	    (7, 'Bar-Pocket', 'ios_apps', '', 'com.example.bar-pocket'),
	    (8, 'Bar', 'ipados_apps', '', 'com.example.bar');
	`
	_, err = db.Exec(dataStmt)
	require.NoError(t, err)

	err = db.Select(&titles, `SELECT name, source, browser, bundle_identifier, additional_identifier FROM software_titles`)
	require.NoError(t, err)
	one := uint32(1)
	two := uint32(2)
	expectedTitles = append(expectedTitles, []softwareTitle{
		{"Foo", "ios_apps", "", ptr.String("com.example.foo"), &one},
		{"Foo", "ipados_apps", "", ptr.String("com.example.foo"), &two},
		{"Bar-Pocket", "ios_apps", "", ptr.String("com.example.bar-pocket"), &one},
		{"Bar", "ipados_apps", "", ptr.String("com.example.bar"), &two},
	}...)
	assert.ElementsMatch(t, expectedTitles, titles)

	// Ensure that the unique key is enforced
	dataStmt = `
	  INSERT INTO software_titles (id, name, source, browser, bundle_identifier) VALUES
	    (200, 'Foo-Pocket', 'ipados_apps', '', 'com.example.foo');
	`
	_, err = db.Exec(dataStmt)
	assert.ErrorContains(t, err, "Duplicate entry")

}
