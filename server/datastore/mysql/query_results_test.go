package mysql

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/test"
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
		{"DeleteForHost", testDeleteQueryResultsForHost},
		{"Count", testCountResultsForQuery},
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
	results, err := ds.QueryResultRows(context.Background(), resultRows[0].QueryID, resultRows[0].HostID)
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
	results, err = ds.QueryResultRows(context.Background(), resultRow3[0].QueryID, resultRow3[0].HostID)
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Equal(t, resultRow3[0].QueryID, results[0].QueryID)
	require.Equal(t, resultRow3[0].HostID, results[0].HostID)
	require.Equal(t, resultRow3[0].LastFetched.Unix(), results[0].LastFetched.Unix())
	require.JSONEq(t, string(resultRow3[0].Data), string(results[0].Data))

	// Assert that QueryResultRows returns empty slice when no results are found
	results, err = ds.QueryResultRows(context.Background(), 999, 999)
	require.NoError(t, err)
	require.Len(t, results, 0)
}

func testDeleteQueryResultsForHost(t *testing.T, ds *Datastore) {
	user := test.NewUser(t, ds, "Test User", "test@example.com", true)
	query := test.NewQuery(t, ds, nil, "New Query", "SELECT 1", user.ID, true)
	query2 := test.NewQuery(t, ds, nil, "New Query 2", "SELECT 1", user.ID, true)
	host := test.NewHost(t, ds, "hostname123", "192.168.1.100", "1234", "UI8XB1223", time.Now())

	mockTime := time.Now().UTC().Truncate(time.Second)

	// Insert 2 Result Rows
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

	// Delete Query Results for Host
	err = ds.DeleteQueryResultsForHost(context.Background(), host.ID, query.ID)
	require.NoError(t, err)

	// Assert that Query1 returns 0 results
	results, err := ds.QueryResultRows(context.Background(), query.ID, host.ID)
	require.NoError(t, err)
	require.Len(t, results, 0)

	// Assert that Query2 returns 1 result
	results, err = ds.QueryResultRows(context.Background(), query2.ID, host.ID)
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Equal(t, resultRow3[0].QueryID, results[0].QueryID)
	require.Equal(t, resultRow3[0].HostID, results[0].HostID)
	require.Equal(t, resultRow3[0].LastFetched.Unix(), results[0].LastFetched.Unix())
	require.JSONEq(t, string(resultRow3[0].Data), string(results[0].Data))
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
