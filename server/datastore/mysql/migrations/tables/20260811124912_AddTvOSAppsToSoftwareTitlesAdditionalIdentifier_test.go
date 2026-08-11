package tables

import (
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func requireSoftwareTitleAdditionalIdentifier(t *testing.T, db *sqlx.DB, titleID int64, want uint) {
	t.Helper()
	var additionalIdentifier *uint
	require.NoError(t, db.Get(&additionalIdentifier,
		`SELECT additional_identifier FROM software_titles WHERE id = ?`, titleID))
	require.NotNil(t, additionalIdentifier)
	require.Equal(t, want, *additionalIdentifier)
}

func TestUp_20260811124912(t *testing.T) {
	db := applyUpToPrev(t)

	const bundleID = "com.netflix.Netflix"
	macOSTitleID := execNoErrLastID(t, db,
		`INSERT INTO software_titles (name, source, bundle_identifier) VALUES ('Netflix', 'apps', ?)`, bundleID)

	// Before the migration a tvOS title is indistinguishable from a macOS one,
	// so the shared bundle identifier collides on the unique key.
	_, err := db.Exec(
		`INSERT INTO software_titles (name, source, bundle_identifier) VALUES ('Netflix', 'tvos_apps', ?)`, bundleID)
	require.ErrorContains(t, err, "Duplicate entry")

	applyNext(t, db)

	tvOSTitleID := execNoErrLastID(t, db,
		`INSERT INTO software_titles (name, source, bundle_identifier) VALUES ('Netflix', 'tvos_apps', ?)`, bundleID)
	requireSoftwareTitleAdditionalIdentifier(t, db, tvOSTitleID, 3)

	// The existing discriminators are unchanged, so the same bundle ID can still
	// hold one title per Apple platform.
	requireSoftwareTitleAdditionalIdentifier(t, db, macOSTitleID, 0)
	for source, want := range map[string]uint{"ios_apps": 1, "ipados_apps": 2} {
		titleID := execNoErrLastID(t, db,
			`INSERT INTO software_titles (name, source, bundle_identifier) VALUES ('Netflix', ?, ?)`, source, bundleID)
		requireSoftwareTitleAdditionalIdentifier(t, db, titleID, want)
	}

	// Titles without a bundle identifier still get NULL, which keeps them out of
	// the unique key entirely.
	noBundleID := execNoErrLastID(t, db,
		`INSERT INTO software_titles (name, source) VALUES ('some-deb', 'deb_packages')`)
	var additionalIdentifier *uint
	require.NoError(t, db.Get(&additionalIdentifier,
		`SELECT additional_identifier FROM software_titles WHERE id = ?`, noBundleID))
	require.Nil(t, additionalIdentifier)
}
