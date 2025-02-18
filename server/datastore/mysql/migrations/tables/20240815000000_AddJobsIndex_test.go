package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20240815000000(t *testing.T) {
	db := applyUpToPrev(t)

	// Apply current migration
	applyNext(t, db)

	// Check if the index exists
	var indexExists bool
	err := db.QueryRow(`
		SELECT 1 FROM information_schema.statistics
		WHERE table_schema = DATABASE()
		AND table_name = 'jobs'
		AND index_name = 'idx_jobs_state_not_before_updated_at'
	`).Scan(&indexExists)

	require.NoError(t, err)
	require.True(t, indexExists, "Index idx_jobs_state_not_before_updated_at should exist")

}
