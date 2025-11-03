package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20251028140100(t *testing.T) {
	db := applyUpToPrev(t)

	// look up table, and see it does not contain FLEET_VAR_SCEP_WINDOWS_CERTIFICATE_ID
	var count int
	err := db.Get(&count, `SELECT COUNT(*) FROM fleet_variables WHERE name = 'FLEET_VAR_SCEP_WINDOWS_CERTIFICATE_ID'`)
	require.NoError(t, err)
	require.Equal(t, 0, count)

	// Apply current migration.
	applyNext(t, db)

	// look up table, and see it now contains FLEET_VAR_SCEP_WINDOWS_CERTIFICATE_ID
	err = db.Get(&count, `SELECT COUNT(*) FROM fleet_variables WHERE name = 'FLEET_VAR_SCEP_WINDOWS_CERTIFICATE_ID'`)
	require.NoError(t, err)
	require.Equal(t, 1, count)
}
