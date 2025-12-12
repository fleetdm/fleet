package tables

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestUp_20220908181826(t *testing.T) {
	db := applyUpToPrev(t)

	zeroTime := time.Unix(0, 0).Add(24 * time.Hour)
	sqlInsert := `
				INSERT INTO hosts (
					detail_updated_at,
					label_updated_at,
					policy_updated_at,
					osquery_host_id,
					node_key,
					team_id,
					refetch_requested
				) VALUES (?, ?, ?, ?, ?, ?, ?)
			`
	_, err := db.Exec(sqlInsert, zeroTime, zeroTime, zeroTime, "host_id", "node_key", nil, 1)
	require.NoError(t, err)

	applyNext(t, db)

	sqlUpdate := `UPDATE hosts SET orbit_node_key = ? WHERE osquery_host_id = ?`
	_, err = db.Exec(sqlUpdate, "orbit_node_key", "host_id")
	require.NoError(t, err)
}
