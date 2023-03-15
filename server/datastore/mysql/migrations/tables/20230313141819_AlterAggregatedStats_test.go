package tables

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20230313141819(t *testing.T) {
	db := applyUpToPrev(t)

	oldInsertStmt := `
    INSERT INTO aggregated_stats (id, type, json_value)
		VALUES (?, ?, ?)
	`

	// insert a value
	execNoErr(t, db, oldInsertStmt, 0, "test-stats", `{"count":1}`)
	execNoErr(t, db, oldInsertStmt, 1, "test-stats", `{"count":0}`)
	execNoErr(t, db, oldInsertStmt, 2, "test-stats", `{"count":1}`)

	// Apply current migration.
	applyNext(t, db)

	var rows []struct {
		ID          uint            `db:"id"`
		Type        string          `db:"type"`
		GlobalStats bool            `db:"global_stats"`
		JSONValue   json.RawMessage `db:"json_value"`
	}
	err := db.Select(&rows, "SELECT id, type, global_stats, json_value FROM aggregated_stats ORDER BY id")
	require.NoError(t, err)
	require.Len(t, rows, 3)

	require.Equal(t, true, rows[0].GlobalStats)
	require.Equal(t, false, rows[1].GlobalStats)
	require.Equal(t, false, rows[2].GlobalStats)

	require.Equal(t, "test-stats", rows[0].Type)
	require.Equal(t, "test-stats", rows[1].Type)
	require.Equal(t, "test-stats", rows[2].Type)

	require.JSONEq(t, `{"count":1}`, string(rows[0].JSONValue))
	require.JSONEq(t, `{"count":0}`, string(rows[1].JSONValue))
	require.JSONEq(t, `{"count":1}`, string(rows[2].JSONValue))

	newInsertStmt := `
    INSERT INTO aggregated_stats (id, type, global_stats, json_value)
		VALUES (?, ?, ?, ?)
	`
	// can insert with id 0 but global stats to false, without conflict
	execNoErr(t, db, newInsertStmt, 0, "test-stats", false, `{"count":0}`)

	// but inserting again fails
	_, err = db.Exec(newInsertStmt, 0, "test-stats", true, `{"count":2}`)
	require.Error(t, err)
}
