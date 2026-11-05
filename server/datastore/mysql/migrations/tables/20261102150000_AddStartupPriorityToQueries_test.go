package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20261102150000(t *testing.T) {
	db := applyUpToPrev(t)

	// Simulate a partially applied migration; Up should add the missing column.
	_, err := db.Exec(`ALTER TABLE queries ADD COLUMN startup_priority BIGINT UNSIGNED NULL`)
	require.NoError(t, err)

	applyNext(t, db)

	tx, err := db.Begin()
	require.NoError(t, err)
	require.True(t, columnExists(tx, "queries", "startup_priority"))
	require.True(t, columnExists(tx, "scheduled_queries", "startup_priority"))
	require.NoError(t, tx.Rollback())
}
