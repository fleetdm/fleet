package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20240814135330(t *testing.T) {
	db := applyUpToPrev(t)

	// Apply current migration
	applyNext(t, db)

	// Check if the index exists
	var indexExists bool
	err := db.QueryRow(`
		SELECT 1 FROM information_schema.statistics
		WHERE table_schema = DATABASE()
		AND table_name = 'query_results'
		AND index_name = 'idx_query_id_host_id_last_fetched'
	`).Scan(&indexExists)

	require.NoError(t, err)
	require.True(t, indexExists, "Index idx_query_id_host_id_last_fetched should exist")

}
