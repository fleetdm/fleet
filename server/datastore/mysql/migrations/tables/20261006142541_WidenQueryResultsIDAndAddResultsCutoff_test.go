package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20261006142541(t *testing.T) {
	db := applyUpToPrev(t)

	queryID := execNoErrLastID(t, db, `INSERT INTO queries (name, description, query) VALUES ('q', '', 'SELECT 1')`)
	// Close to the old INT UNSIGNED limit, so the next ids only fit after the migration.
	const nearIntMax = 4294967290
	execNoErr(t, db, `INSERT INTO query_results (id, query_id, host_id, last_fetched, data) VALUES (?, ?, 1, NOW(), '{"v": "1"}')`, nearIntMax, queryID)

	applyNext(t, db)

	var data string
	require.NoError(t, db.Get(&data, `SELECT data FROM query_results WHERE id = ?`, nearIntMax))
	require.JSONEq(t, `{"v": "1"}`, data)

	for range 10 {
		execNoErr(t, db, `INSERT INTO query_results (query_id, host_id, last_fetched, data) VALUES (?, 1, NOW(), '{"v": "2"}')`, queryID)
	}
	var maxID uint64
	require.NoError(t, db.Get(&maxID, `SELECT MAX(id) FROM query_results`))
	require.Equal(t, uint64(nearIntMax+10), maxID)

	var cutoff struct {
		ValidFromID uint64 `db:"results_valid_from_id"`
		Pending     bool   `db:"results_cleanup_pending"`
	}
	require.NoError(t, db.Get(&cutoff, `SELECT results_valid_from_id, results_cleanup_pending FROM queries WHERE id = ?`, queryID))
	require.Zero(t, cutoff.ValidFromID)
	require.False(t, cutoff.Pending)
}
