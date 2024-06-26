package tables

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestUp_20231009094544(t *testing.T) {
	db := applyUpToPrev(t)

	// Apply current migration.
	applyNext(t, db)

	// Insert a record into query_results
	insertStmt := `INSERT INTO query_results (
		query_id, host_id, osquery_version, error, last_fetched, data
	) VALUES (?, ?, ?, ?, ?, ?)`

	queryID := insertQuery(t, db)
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

	// Insert a sample error result containing a NULL data field
	errorMessage := "Some error message"
	_, err = db.Exec(insertStmt, queryID, hostID, osqueryVersion, errorMessage, lastFetched, nil)
	require.NoError(t, err)

	type QueryResult struct {
		ID                uint             `db:"id"`
		QueryID           uint             `db:"query_id"`
		HostID            uint             `db:"host_id"`
		OsqueryVersion    string           `db:"osquery_version"`
		Error             string           `db:"error"`
		LastFetched       time.Time        `db:"last_fetched"`
		OsqueryResultData *json.RawMessage `db:"data"`
	}

	// Load the 1st result
	var queryReport []QueryResult
	selectStmt := `
		SELECT id, query_id, host_id, osquery_version, error, last_fetched, data
		FROM query_results
		WHERE query_id = ? AND host_id = ?
		ORDER BY id ASC
	`
	err = db.Select(&queryReport, selectStmt, queryID, hostID)
	require.NoError(t, err)

	require.Equal(t, queryID, queryReport[0].QueryID)
	require.Equal(t, hostID, queryReport[0].HostID)
	require.Equal(t, osqueryVersion, queryReport[0].OsqueryVersion)
	require.Empty(t, queryReport[0].Error)
	require.True(t, lastFetched.Sub(queryReport[0].LastFetched) < time.Second)
	require.JSONEq(t, string(jsonData), string(*queryReport[0].OsqueryResultData))

	// Error results should be loaded as well
	require.Equal(t, queryID, queryReport[1].QueryID)
	require.Equal(t, hostID, queryReport[1].HostID)
	require.Equal(t, osqueryVersion, queryReport[1].OsqueryVersion)
	require.Equal(t, errorMessage, queryReport[1].Error)
	require.True(t, lastFetched.Sub(queryReport[1].LastFetched) < time.Second) // allow a 1 sec difference to account for time to run the query
	require.Empty(t, queryReport[1].OsqueryResultData)

	// Delete the query we just created to test the ON DELETE CASCADE
	deleteQueryStmt := `DELETE FROM queries WHERE id = ?`
	_, err = db.Exec(deleteQueryStmt, queryID)
	require.NoError(t, err)

	// Verify that both query_result records were deleted
	var count int
	err = db.Get(&count, "SELECT COUNT(*) FROM query_results WHERE id = ?", id)
	require.NoError(t, err)
	require.Equal(t, 0, count)
}
