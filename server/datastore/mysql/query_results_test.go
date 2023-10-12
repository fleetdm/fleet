package mysql

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestQueryResults(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"Save", saveQueryResultRows},
		{"Get", getQueryResultRows},
		{"CountForQuery", testCountResultsForQuery},
		{"CountForQueryAndHost", testCountResultsForQueryAndHost},
		{"Overwrite", testOverwriteQueryResultRows},
		{"MaxRows", testQueryResultRowsDoNotExceedMaxRows},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)
			c.fn(t, ds)
		})
	}
}

func saveQueryResultRows(t *testing.T, ds *Datastore) {
	user := test.NewUser(t, ds, "Test User", "test@example.com", true)
	query := test.NewQuery(t, ds, nil, "New Query", "SELECT 1", user.ID, true)
	host := test.NewHost(t, ds, "hostname123", "192.168.1.100", "1234", "UI8XB1223", time.Now())

	mockTime := time.Now().UTC().Truncate(time.Second)

	resultRows := []*fleet.ScheduledQueryResultRow{
		{
			QueryID:     query.ID,
			HostID:      host.ID,
			LastFetched: mockTime,
			Data: json.RawMessage(
				`{"model": "USB Keyboard", "vendor": "Apple Inc."}`,
			),
		},
		{
			QueryID:     query.ID,
			HostID:      host.ID,
			LastFetched: mockTime,
			Data: json.RawMessage(
				`{"model": "USB Mouse", "vendor": "Logitech"}`,
			),
		},
	}

	err := ds.SaveQueryResultRows(context.Background(), resultRows)
	require.NoError(t, err)
}

func getQueryResultRows(t *testing.T, ds *Datastore) {
	user := test.NewUser(t, ds, "Test User", "test@example.com", true)
	query := test.NewQuery(t, ds, nil, "New Query", "SELECT 1", user.ID, true)
	host := test.NewHost(t, ds, "hostname123", "192.168.1.100", "1234", "UI8XB1223", time.Now())

	mockTime := time.Now().UTC().Truncate(time.Second)

	// Insert 2 Result Rows for Query1
	resultRows := []*fleet.ScheduledQueryResultRow{
		{
			QueryID:     query.ID,
			HostID:      host.ID,
			LastFetched: mockTime,
			Data: json.RawMessage(
				`{"model": "USB Keyboard", "vendor": "Apple Inc."}`,
			),
		},
		{
			QueryID:     query.ID,
			HostID:      host.ID,
			LastFetched: mockTime,
			Data: json.RawMessage(
				`{"model": "USB Mouse", "vendor": "Logitech"}`,
			),
		},
	}

	err := ds.SaveQueryResultRows(context.Background(), resultRows)
	require.NoError(t, err)

	// Insert Result Row for different Scheduled Query
	query2 := test.NewQuery(t, ds, nil, "New Query 2", "SELECT 1", user.ID, true)
	resultRow3 := []*fleet.ScheduledQueryResultRow{
		{
			QueryID:     query2.ID,
			HostID:      host.ID,
			LastFetched: mockTime,
			Data: json.RawMessage(
				`{"model": "USB Hub","vendor": "Logitech"}`,
			),
		},
	}

	err = ds.SaveQueryResultRows(context.Background(), resultRow3)
	require.NoError(t, err)

	// Assert that Query1 returns 2 results
	results, err := ds.QueryResultRowsForHost(context.Background(), resultRows[0].QueryID, resultRows[0].HostID)
	require.NoError(t, err)
	require.Len(t, results, 2)
	require.Equal(t, resultRows[0].QueryID, results[0].QueryID)
	require.Equal(t, resultRows[0].HostID, results[0].HostID)
	require.Equal(t, resultRows[0].LastFetched.Unix(), results[0].LastFetched.Unix())
	require.JSONEq(t, string(resultRows[0].Data), string(results[0].Data))
	require.Equal(t, resultRows[1].QueryID, results[1].QueryID)
	require.Equal(t, resultRows[1].HostID, results[1].HostID)
	require.Equal(t, resultRows[1].LastFetched.Unix(), results[1].LastFetched.Unix())
	require.JSONEq(t, string(resultRows[1].Data), string(results[1].Data))

	// Assert that Query2 returns 1 result
	results, err = ds.QueryResultRowsForHost(context.Background(), resultRow3[0].QueryID, resultRow3[0].HostID)
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Equal(t, resultRow3[0].QueryID, results[0].QueryID)
	require.Equal(t, resultRow3[0].HostID, results[0].HostID)
	require.Equal(t, resultRow3[0].LastFetched.Unix(), results[0].LastFetched.Unix())
	require.JSONEq(t, string(resultRow3[0].Data), string(results[0].Data))

	// Assert that QueryResultRows returns empty slice when no results are found
	results, err = ds.QueryResultRowsForHost(context.Background(), 999, 999)
	require.NoError(t, err)
	require.Len(t, results, 0)
}

func testCountResultsForQuery(t *testing.T, ds *Datastore) {
	user := test.NewUser(t, ds, "Test User", "test@example.com", true)
	query1 := test.NewQuery(t, ds, nil, "New Query", "SELECT 1", user.ID, true)
	query2 := test.NewQuery(t, ds, nil, "New Query 2", "SELECT 1", user.ID, true)
	host := test.NewHost(t, ds, "hostname123", "192.168.1.100", "1234", "UI8XB1223", time.Now())

	mockTime := time.Now().UTC().Truncate(time.Second)

	// Insert 1 Result Row for Query1
	resultRow := []*fleet.ScheduledQueryResultRow{
		{
			QueryID:     query1.ID,
			HostID:      host.ID,
			LastFetched: mockTime,
			Data: json.RawMessage(`{
				"model": "USB Keyboard",
				"vendor": "Apple Inc."
			}`),
		},
	}

	err := ds.SaveQueryResultRows(context.Background(), resultRow)
	require.NoError(t, err)

	// Insert 5 Result Rows for Query2
	resultRow2 := []*fleet.ScheduledQueryResultRow{
		{
			QueryID:     query2.ID,
			HostID:      host.ID,
			LastFetched: mockTime,
			Data: json.RawMessage(`{
				"model": "USB Mouse",
				"vendor": "Apple Inc."
			}`),
		},
	}
	for i := 0; i < 5; i++ {
		err = ds.SaveQueryResultRows(context.Background(), resultRow2)
		require.NoError(t, err)
	}

	// Assert that ResultCountForQuery returns 1
	count, err := ds.ResultCountForQuery(context.Background(), query1.ID)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	// Assert that ResultCountForQuery returns 5
	count, err = ds.ResultCountForQuery(context.Background(), query2.ID)
	require.NoError(t, err)
	require.Equal(t, 5, count)

	// Returns empty result when no results are found
	count, err = ds.ResultCountForQuery(context.Background(), 999)
	require.NoError(t, err)
	require.Equal(t, 0, count)
}

func testCountResultsForQueryAndHost(t *testing.T, ds *Datastore) {
	user := test.NewUser(t, ds, "Test User", "test@example.com", true)
	query1 := test.NewQuery(t, ds, nil, "New Query", "SELECT 1", user.ID, true)
	query2 := test.NewQuery(t, ds, nil, "New Query 2", "SELECT 1", user.ID, true)
	host := test.NewHost(t, ds, "host1", "192.168.1.100", "1234", "UI8XB1223", time.Now())
	host2 := test.NewHost(t, ds, "host2", "192.168.1.101", "4567", "UI8XB1224", time.Now())

	mockTime := time.Now().UTC().Truncate(time.Second)

	resultRows := []*fleet.ScheduledQueryResultRow{
		{
			QueryID:     query1.ID,
			HostID:      host.ID,
			LastFetched: mockTime,
			Data: json.RawMessage(`{
				"model": "USB Keyboard",
				"vendor": "Apple Inc."
			}`),
		},
		{
			QueryID:     query1.ID,
			HostID:      host.ID,
			LastFetched: mockTime,
			Data: json.RawMessage(`{
				"model": "USB Mouse",
				"vendor": "Logitech"
			}`),
		},
		{
			QueryID:     query1.ID,
			HostID:      host2.ID,
			LastFetched: mockTime,
			Data: json.RawMessage(`{
				"model": "USB Mouse",
				"vendor": "Logitech"
			}`),
		},
		{
			QueryID:     query2.ID,
			HostID:      host.ID,
			LastFetched: mockTime,
			Data: json.RawMessage(`{
				"foo": "bar"
			}`),
		},
	}

	err := ds.SaveQueryResultRows(context.Background(), resultRows)
	require.NoError(t, err)

	// Assert that Query1 returns 2
	count, err := ds.ResultCountForQueryAndHost(context.Background(), query1.ID, host.ID)
	require.NoError(t, err)
	require.Equal(t, 2, count)

	// Assert that ResultCountForQuery returns 1
	count, err = ds.ResultCountForQueryAndHost(context.Background(), query2.ID, host.ID)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	// Returns empty result when no results are found
	count, err = ds.ResultCountForQueryAndHost(context.Background(), 999, host.ID)
	require.NoError(t, err)
	require.Equal(t, 0, count)
}

func testOverwriteQueryResultRows(t *testing.T, ds *Datastore) {
	user := test.NewUser(t, ds, "Test User", "test@example.com", true)
	query := test.NewQuery(t, ds, nil, "Overwrite Test Query", "SELECT 1", user.ID, true)
	host := test.NewHost(t, ds, "hostname1234", "192.168.1.101", "12345", "UI8XB1224", time.Now())

	mockTime := time.Now().UTC().Truncate(time.Second)

	// Insert initial Result Rows
	initialRows := []*fleet.ScheduledQueryResultRow{
		{
			QueryID:     query.ID,
			HostID:      host.ID,
			LastFetched: mockTime,
			Data: json.RawMessage(
				`{"model": "USB Keyboard", "vendor": "Apple Inc."}`,
			),
		},
	}

	err := ds.SaveQueryResultRows(context.Background(), initialRows)
	require.NoError(t, err)

	// Overwrite Result Rows with new data
	newMockTime := mockTime.Add(2 * time.Minute)
	overwriteRows := []*fleet.ScheduledQueryResultRow{
		{
			QueryID:     query.ID,
			HostID:      host.ID,
			LastFetched: newMockTime,
			Data: json.RawMessage(
				`{"model": "USB Mouse", "vendor": "Logitech"}`,
			),
		},
	}

	err = ds.OverwriteQueryResultRows(context.Background(), overwriteRows)
	require.NoError(t, err)

	// Assert that we get the overwritten data (1 result with USB Mouse data)
	results, err := ds.QueryResultRowsForHost(context.Background(), overwriteRows[0].QueryID, overwriteRows[0].HostID)
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Equal(t, overwriteRows[0].QueryID, results[0].QueryID)
	require.Equal(t, overwriteRows[0].HostID, results[0].HostID)
	require.Equal(t, overwriteRows[0].LastFetched.Unix(), results[0].LastFetched.Unix())
	require.JSONEq(t, string(overwriteRows[0].Data), string(results[0].Data))
}

func testQueryResultRowsDoNotExceedMaxRows(t *testing.T, ds *Datastore) {
	user := test.NewUser(t, ds, "Test User", "test@example.com", true)
	query := test.NewQuery(t, ds, nil, "Overwrite Test Query", "SELECT 1", user.ID, true)
	host := test.NewHost(t, ds, "hostname1", "192.168.1.101", "12345", "UI8XB1224", time.Now())

	mockTime := time.Now().UTC().Truncate(time.Second)

	// Generate more than max rows
	rows := fleet.MaxQueryReportRows + 50
	largeBatchRows := make([]*fleet.ScheduledQueryResultRow, rows)
	for i := 0; i < rows; i++ {
		largeBatchRows[i] = &fleet.ScheduledQueryResultRow{
			QueryID:     query.ID,
			HostID:      host.ID,
			LastFetched: mockTime,
			Data:        json.RawMessage(`{"model": "Bulk Mouse", "vendor": "BulkTech"}`),
		}
	}

	err := ds.OverwriteQueryResultRows(context.Background(), largeBatchRows)
	require.NoError(t, err)

	// Confirm only max rows are stored for the queryID
	allResults, err := ds.QueryResultRowsForHost(context.Background(), query.ID, host.ID)
	require.NoError(t, err)
	require.Len(t, allResults, fleet.MaxQueryReportRows)

	// Confirm that new rows are not added when the max is reached
	host2 := test.NewHost(t, ds, "hostname2", "192.168.1.102", "678910", "UI8XB1225", time.Now())
	newMockTime := mockTime.Add(2 * time.Minute)
	overwriteRows := []*fleet.ScheduledQueryResultRow{
		{
			QueryID:     query.ID,
			HostID:      host2.ID,
			LastFetched: newMockTime,
			Data: json.RawMessage(
				`{"model": "USB Mouse", "vendor": "Logitech"}`,
			),
		},
	}

	err = ds.OverwriteQueryResultRows(context.Background(), overwriteRows)
	require.NoError(t, err)

	host2Results, err := ds.QueryResultRowsForHost(context.Background(), query.ID, host2.ID)
	require.NoError(t, err)
	require.Len(t, host2Results, 0)
}

func (ds *Datastore) SaveQueryResultRows(ctx context.Context, rows []*fleet.ScheduledQueryResultRow) error {
	if len(rows) == 0 {
		return nil // Nothing to insert
	}

	valueStrings := make([]string, 0, len(rows))
	valueArgs := make([]interface{}, 0, len(rows)*4)

	for _, row := range rows {
		valueStrings = append(valueStrings, "(?, ?, ?, ?)")
		valueArgs = append(valueArgs, row.QueryID, row.HostID, row.LastFetched, row.Data)
	}

	insertStmt := fmt.Sprintf(`
        INSERT INTO query_results (query_id, host_id, last_fetched, data)
            VALUES %s
    `, strings.Join(valueStrings, ","))

	_, err := ds.writer(ctx).ExecContext(ctx, insertStmt, valueArgs...)
	if err != nil {
		return err
	}

	return nil
}

func (ds *Datastore) QueryResultRowsForHost(ctx context.Context, queryID, hostID uint) ([]*fleet.ScheduledQueryResultRow, error) {
	selectStmt := `
               SELECT query_id, host_id, last_fetched, data FROM query_results
                       WHERE query_id = ? AND host_id = ?
               `
	results := []*fleet.ScheduledQueryResultRow{}
	err := sqlx.SelectContext(ctx, ds.reader(ctx), &results, selectStmt, queryID, hostID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "selecting query result rows for host")
	}

	return results, nil
}
