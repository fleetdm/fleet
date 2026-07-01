package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20260701135344(t *testing.T) {
	db := applyUpToPrev(t)

	applyNext(t, db)

	// Verify column can be selected (guards against "Unknown column" errors).
	rows, err := db.Query(`SELECT token_invalid FROM abm_tokens LIMIT 0`)
	require.NoError(t, err)
	require.NoError(t, rows.Close())

	// Verify column structure.
	var colName, colType, isNullable, colDefault string
	err = db.QueryRow(`
		SELECT COLUMN_NAME, COLUMN_TYPE, IS_NULLABLE, COLUMN_DEFAULT
		FROM information_schema.COLUMNS
		WHERE TABLE_SCHEMA = DATABASE()
		  AND TABLE_NAME = 'abm_tokens'
		  AND COLUMN_NAME = 'token_invalid'
	`).Scan(&colName, &colType, &isNullable, &colDefault)
	require.NoError(t, err)
	require.Equal(t, "token_invalid", colName)
	require.Equal(t, "tinyint(1)", colType)
	require.Equal(t, "NO", isNullable)
	require.Equal(t, "0", colDefault)
}
