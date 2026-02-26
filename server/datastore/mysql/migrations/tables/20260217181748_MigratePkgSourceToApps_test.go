package tables

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

func TestUp_20260217181748(t *testing.T) {
	db := applyUpToPrev(t)

	unaffectedTitleID := execNoErrLastID(t, db, `
		INSERT INTO software_titles (name, source)
		VALUES ('Unaffected App', 'pkg_packages')
	`)
	unaffectedTitleID2 := execNoErrLastID(t, db, `
		INSERT INTO software_titles (name, source, bundle_identifier)
		VALUES ('Unaffected App 2', 'pkg_packages', '')
	`)
	affectedTitleID := execNoErrLastID(t, db, `
		INSERT INTO software_titles (name, source, bundle_identifier)
		VALUES ('Affected App', 'pkg_packages', 'com.example')
	`)

	// Apply current migration.
	applyNext(t, db)

	t.Run("unaffected title no bundle id", func(t *testing.T) {
		var title fleet.SoftwareTitle
		err := db.Get(&title, `SELECT name, source, bundle_identifier FROM software_titles WHERE id = ?`, unaffectedTitleID)
		require.NoError(t, err)
		require.Equal(t, "Unaffected App", title.Name)
		require.Equal(t, "pkg_packages", title.Source)
		require.Nil(t, title.BundleIdentifier)
	})

	t.Run("unaffected title empty bundle id", func(t *testing.T) {
		var title fleet.SoftwareTitle
		err := db.Get(&title, `SELECT name, source, bundle_identifier FROM software_titles WHERE id = ?`, unaffectedTitleID2)
		require.NoError(t, err)
		require.Equal(t, "Unaffected App 2", title.Name)
		require.Equal(t, "pkg_packages", title.Source)
		require.NotNil(t, title.BundleIdentifier)
		require.Equal(t, "", *title.BundleIdentifier)
	})

	t.Run("affected title", func(t *testing.T) {
		var title fleet.SoftwareTitle
		err := db.Get(&title, `SELECT name, source, bundle_identifier FROM software_titles WHERE id = ?`, affectedTitleID)
		require.NoError(t, err)
		require.Equal(t, "Affected App", title.Name)
		require.Equal(t, "apps", title.Source)
		require.NotNil(t, title.BundleIdentifier)
		require.Equal(t, "com.example", *title.BundleIdentifier)
	})
}
