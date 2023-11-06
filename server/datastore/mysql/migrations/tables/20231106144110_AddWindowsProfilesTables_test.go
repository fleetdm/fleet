package tables

import (
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestUp_20231106144110(t *testing.T) {
	db := applyUpToPrev(t)

	var oldStatuses []string
	err := sqlx.Select(db, &oldStatuses, "SELECT status FROM mdm_apple_delivery_status ORDER BY 1")
	require.NoError(t, err)
	require.NotEmpty(t, oldStatuses)

	var oldOps []string
	err = sqlx.Select(db, &oldOps, "SELECT operation_type FROM mdm_apple_operation_types ORDER BY 1")
	require.NoError(t, err)
	require.NotEmpty(t, oldOps)

	applyNext(t, db)

	// check that the status/operation types are still present
	var newStatuses []string
	err = sqlx.Select(db, &newStatuses, "SELECT status FROM mdm_delivery_status ORDER BY 1")
	require.NoError(t, err)
	require.Equal(t, oldStatuses, newStatuses)

	var newOps []string
	err = sqlx.Select(db, &newOps, "SELECT operation_type FROM mdm_operation_types ORDER BY 1")
	require.NoError(t, err)
	require.Equal(t, oldOps, newOps)
}
