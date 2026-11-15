package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20261115001714(t *testing.T) {
	db := applyUpToPrev(t)

	// Before the migration, only the (name, created_at) index exists.
	require.True(t, indexExists(db, "cron_stats", "idx_cron_stats_name_created_at"))
	require.False(t, indexExists(db, "cron_stats", "idx_cron_stats_name_stats_type_created_at"))

	applyNext(t, db)

	require.True(t, indexExists(db, "cron_stats", "idx_cron_stats_name_stats_type_created_at"))
	// The old index is kept: the new index cannot serve a name-only lookup
	// ordered by created_at because stats_type sits in between.
	require.True(t, indexExists(db, "cron_stats", "idx_cron_stats_name_created_at"))
}
