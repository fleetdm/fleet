package tables

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestUp_20230927155121(t *testing.T) {
	db := applyUpToPrev(t)

	// Apply current migration.
	applyNext(t, db)

	// Insert a record into query_results
	insertStmt := `INSERT INTO query_results (
		query_id, host_id, osquery_version, error, last_fetched, data
	) VALUES (?, ?, ?, ?, ?, ?)`

	queryID := insertQuery(t, db)
	hostID := insertHost(t, db)
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

	type QueryReport struct {
		ID                uint      `db:"id"`
		QueryID           uint      `db:"query_id"`
		HostID            uint      `db:"host_id"`
		OsqueryVersion    string    `db:"osquery_version"`
		Error             string    `db:"error"`
		LastFetched       time.Time `db:"last_fetched"`
		OsqueryResultData string    `db:"data"`
	}

	// Load the report we just created
	var report QueryReport
	selectStmt := `SELECT id, query_id, host_id, osquery_version, error, last_fetched, data
	FROM query_results
	WHERE id = ?`
	err = db.Get(&report, selectStmt, id)
	require.NoError(t, err)

	require.Equal(t, uint(id), report.ID)
	require.Equal(t, queryID, report.QueryID)
	require.Equal(t, hostID, report.HostID)
	require.Equal(t, osqueryVersion, report.OsqueryVersion)
	require.Empty(t, report.Error)
	require.Equal(t, lastFetched.Truncate(time.Second), report.LastFetched.Truncate(time.Second))
	require.JSONEq(t, string(jsonData), report.OsqueryResultData)

	// Test cascading delete
	// Delete the query we just created
	deleteQueryStmt := `DELETE FROM queries WHERE id = ?`
	_, err = db.Exec(deleteQueryStmt, queryID)
	require.NoError(t, err)

	// Verify that the query_results record was deleted
	var count int
	err = db.Get(&count, "SELECT COUNT(*) FROM query_results WHERE id = ?", id)
	require.NoError(t, err)
	require.Equal(t, 0, count)
}
