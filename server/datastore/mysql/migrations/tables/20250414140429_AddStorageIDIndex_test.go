package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20250414140429(t *testing.T) {
	db := applyUpToPrev(t)

	// Apply current migration
	applyNext(t, db)

	// Check if the index exists
	var indexExists bool
	err := db.QueryRow(`
		SELECT 1 FROM information_schema.statistics
		WHERE table_schema = DATABASE()
		AND table_name = 'software_installers'
		AND index_name = 'idx_software_installers_storage_id'
	`).Scan(&indexExists)

	require.NoError(t, err)
	require.True(t, indexExists, "index idx_software_installers_storage_id should exist")
}
