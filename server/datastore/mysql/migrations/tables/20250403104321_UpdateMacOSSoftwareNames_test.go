package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20250403104321(t *testing.T) {
	db := applyUpToPrev(t)

	// insert some software titles
	dataStmt := `
	  INSERT INTO software_titles (id, name, source, browser, bundle_identifier) VALUES
	    (1, 'Foo.app', 'apps', '', 'com.example.foo'),
	    (2, 'Foo2.app', 'apps', '', 'com.example.foo2'),
	    (3, 'Chrome Extension', 'chrome_extensions', 'chrome', NULL),
	    (4, 'Microsoft Teams.exe', 'programs', '', NULL);
	`

	_, err := db.Exec(dataStmt)
	require.NoError(t, err)

	// Apply current migration.
	applyNext(t, db)

	// macOS apps should be modified, others should not
	var titles []struct {
		Name   string `db:"name"`
		Source string `db:"source"`
	}
	err = db.Select(&titles, "SELECT name, source FROM software_titles")
	require.NoError(t, err)

	expectedNames := map[string]struct{}{
		"Foo":                 {},
		"Foo2":                {},
		"Chrome Extension":    {},
		"Microsoft Teams.exe": {},
	}

	for _, title := range titles {
		_, ok := expectedNames[title.Name]
		require.True(t, ok)
	}
}
