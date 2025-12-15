package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20251124090450(t *testing.T) {
	db := applyUpToPrev(t)

	// look up table, and see it does not contain FLEET_VAR_HOST_PLATFORM
	var count int
	err := db.Get(&count, `SELECT COUNT(*) FROM fleet_variables WHERE name = 'FLEET_VAR_HOST_PLATFORM'`)
	require.NoError(t, err)
	require.Equal(t, 0, count)

	// Apply current migration.
	applyNext(t, db)

	// look up table, and see it now contains FLEET_VAR_HOST_PLATFORM
	err = db.Get(&count, `SELECT COUNT(*) FROM fleet_variables WHERE name = 'FLEET_VAR_HOST_PLATFORM'`)
	require.NoError(t, err)
	require.Equal(t, 1, count)
}
