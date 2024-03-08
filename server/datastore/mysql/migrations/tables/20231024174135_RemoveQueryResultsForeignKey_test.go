package tables

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestUp_20231024174135(t *testing.T) {
	db := applyUpToPrev(t)

	//
	// Insert data to test the migration
	queryID := insertQuery(t, db)

	// Insert a record into query_results
	insertStmt := `INSERT INTO query_results (
		query_id, host_id, osquery_version, error, last_fetched, data
	) VALUES (?, ?, ?, ?, ?, ?)`

	hostID := insertHost(t, db, nil)
	osqueryVersion := "5.9.1"
	lastFetched := time.Now().UTC()

	// Example JSON data for data field
	osqueryData := map[string]string{
		"model":  "USB Keyboard",
		"vendor": "Apple Inc.",
	}
	jsonData, err := json.Marshal(osqueryData)
	require.NoError(t, err)

	res, err := db.Exec(insertStmt, queryID, hostID, osqueryVersion, "", lastFetched, jsonData)
	require.NoError(t, err)
	id, _ := res.LastInsertId()

	// Apply current migration.
	applyNext(t, db)

	//
	// Check data, insert new entries, e.g. to verify migration is safe.
	//

	// Delete the query we just created to test that constraint is gone
	deleteQueryStmt := `DELETE FROM queries WHERE id = ?`
	_, err = db.Exec(deleteQueryStmt, queryID)
	require.NoError(t, err)

	var count int
	err = db.Get(&count, "SELECT COUNT(*) FROM query_results WHERE id = ?", id)
	require.NoError(t, err)
	require.Equal(t, 1, count)
}
