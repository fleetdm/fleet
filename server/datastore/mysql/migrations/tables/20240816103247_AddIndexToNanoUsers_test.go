package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20240816103247(t *testing.T) {
	db := applyUpToPrev(t)

	// Apply current migration
	applyNext(t, db)

	// Check if the index exists
	var indexExists bool
	err := db.QueryRow(`
		SELECT 1 FROM information_schema.statistics
		WHERE table_schema = DATABASE()
		AND table_name = 'nano_users'
		AND index_name = 'idx_unique_id'
	`).Scan(&indexExists)

	require.NoError(t, err)
	require.True(t, indexExists, "Index idx_unique_id should exist")
}
