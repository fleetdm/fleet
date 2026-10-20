package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20261016181351(t *testing.T) {
	db := applyUpToPrev(t)

	var count int
	require.NoError(t, db.Get(&count, `SELECT COUNT(*) FROM fleet_variables WHERE name LIKE 'FLEET_VAR_EJBCA_%'`))
	require.Equal(t, 0, count)

	applyNext(t, db)

	// The variables that link a profile to its EJBCA CA are registered as prefixes.
	var vars []struct {
		Name     string `db:"name"`
		IsPrefix bool   `db:"is_prefix"`
	}
	require.NoError(t, db.Select(&vars, `SELECT name, is_prefix FROM fleet_variables WHERE name LIKE 'FLEET_VAR_EJBCA_%' ORDER BY name`))
	require.Len(t, vars, 2)
	require.Equal(t, "FLEET_VAR_EJBCA_DATA_", vars[0].Name)
	require.True(t, vars[0].IsPrefix)
	require.Equal(t, "FLEET_VAR_EJBCA_PASSWORD_", vars[1].Name)
	require.True(t, vars[1].IsPrefix)
}
