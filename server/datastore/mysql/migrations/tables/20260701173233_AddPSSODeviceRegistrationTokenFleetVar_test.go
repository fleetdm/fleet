package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20260701173233(t *testing.T) {
	db := applyUpToPrev(t)

	var count int
	err := db.Get(&count, `SELECT COUNT(*) FROM fleet_variables WHERE name = 'FLEET_VAR_PSSO_DEVICE_REGISTRATION_TOKEN'`)
	require.NoError(t, err)
	require.Equal(t, 0, count)

	applyNext(t, db)

	err = db.Get(&count, `SELECT COUNT(*) FROM fleet_variables WHERE name = 'FLEET_VAR_PSSO_DEVICE_REGISTRATION_TOKEN'`)
	require.NoError(t, err)
	require.Equal(t, 1, count)
}
