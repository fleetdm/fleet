package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20221115104546(t *testing.T) {
	db := applyUpToPrev(t)

	applyNext(t, db)

	query := `
		INSERT INTO cron_stats (
			name,
			instance,
			stats_type,
			status
		)
		VALUES (?, ?, ?, ?)
	`
	_, err := db.Exec(query, "test_cron", "test_instance", "scheduled", "pending")
	require.NoError(t, err)
}
