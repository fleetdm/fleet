package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20251028140400(t *testing.T) {
	db := applyUpToPrev(t)

	// Create in house app
	ihaID := execNoErrLastID(t, db, `INSERT INTO in_house_apps 
	(name, storage_id, platform) VALUES ('test.ipa', '111', 'ios')`)

	require.Equal(t, int64(1), ihaID)
	var filename string
	err := db.Get(&filename, "SELECT name FROM in_house_apps WHERE id = ?", ihaID)
	require.NoError(t, err)
	require.Equal(t, "test.ipa", filename)

	// Apply current migration.
	applyNext(t, db)

	assertRowCount(t, db, "in_house_apps", 1)

	// Get first in house app
	err = db.Get(&filename, "SELECT filename FROM in_house_apps WHERE id = ?", ihaID)
	require.NoError(t, err)
	require.Equal(t, "test.ipa", filename)

	// Create new in house app
	iha2ID := execNoErrLastID(t, db, `INSERT INTO in_house_apps 
	(filename, storage_id, platform) VALUES ('another_test.ipa', '222', 'ios')`)

	require.Equal(t, int64(2), iha2ID)
	err = db.Get(&filename, "SELECT filename FROM in_house_apps WHERE id = ?", iha2ID)
	require.NoError(t, err)
	require.Equal(t, "another_test.ipa", filename)

	assertRowCount(t, db, "in_house_apps", 2)
}
