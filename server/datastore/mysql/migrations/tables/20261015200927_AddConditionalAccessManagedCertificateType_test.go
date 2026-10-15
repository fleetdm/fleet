package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20261015200927(t *testing.T) {
	db := applyUpToPrev(t)

	const countStmt = `SELECT COUNT(*) FROM fleet_variables
		WHERE name IN ('FLEET_VAR_CONDITIONAL_ACCESS_SCEP_URL', 'FLEET_VAR_CONDITIONAL_ACCESS_SCEP_CHALLENGE') AND is_prefix = 0`
	var count int
	require.NoError(t, db.Get(&count, countStmt))
	require.Equal(t, 0, count)

	applyNext(t, db)

	require.NoError(t, db.Get(&count, countStmt))
	require.Equal(t, 2, count)
}
