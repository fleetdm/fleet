package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20260727084359(t *testing.T) {
	db := applyUpToPrev(t)

	var count int
	err := db.Get(&count, `SELECT COUNT(*) FROM fleet_variables WHERE name IN ('FLEET_VAR_HOST_TARGET_OS_VERSION', 'FLEET_VAR_HOST_TARGET_OS_DEADLINE')`)
	require.NoError(t, err)
	require.Equal(t, 0, count)

	applyNext(t, db)

	err = db.Get(&count, `SELECT COUNT(*) FROM fleet_variables WHERE name IN ('FLEET_VAR_HOST_TARGET_OS_VERSION', 'FLEET_VAR_HOST_TARGET_OS_DEADLINE')`)
	require.NoError(t, err)
	require.Equal(t, 2, count)

	var isPrefix bool
	err = db.Get(&isPrefix, `SELECT is_prefix FROM fleet_variables WHERE name = 'FLEET_VAR_HOST_TARGET_OS_VERSION'`)
	require.NoError(t, err)
	require.False(t, isPrefix)

	err = db.Get(&isPrefix, `SELECT is_prefix FROM fleet_variables WHERE name = 'FLEET_VAR_HOST_TARGET_OS_DEADLINE'`)
	require.NoError(t, err)
	require.False(t, isPrefix)
}
