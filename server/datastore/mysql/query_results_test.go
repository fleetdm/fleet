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
		{"Save", saveQueryResultRow},
		{"Get", getQueryResultRows},
		// {"Apply", testQueriesApply},
		// {"Delete", testQueriesDelete},
		// {"GetByName", testQueriesGetByName},
		// {"DeleteMany", testQueriesDeleteMany},
		// {"Save", testQueriesSave},
		// {"List", testQueriesList},
		// {"LoadPacksForQueries", testQueriesLoadPacksForQueries},
		// {"DuplicateNew", testQueriesDuplicateNew},
		// {"ListFiltersObservers", testQueriesListFiltersObservers},
		// {"ObserverCanRunQuery", testObserverCanRunQuery},
		// {"ListQueriesFiltersByTeamID", testListQueriesFiltersByTeamID},
		// {"ListQueriesFiltersByIsScheduled", testListQueriesFiltersByIsScheduled},
		// {"ListScheduledQueriesForAgents", testListScheduledQueriesForAgents},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)
			c.fn(t, ds)
		})
	}
}

func saveQueryResultRow(t *testing.T, ds *Datastore) {
	user := test.NewUser(t, ds, "Test User", "test@example.com", true)
	query := test.NewQuery(t, ds, nil, "New Query", "SELECT 1", user.ID, true)
	host := test.NewHost(t, ds, "hostname123", "192.168.1.100", "1234", "UI8XB1223", time.Now())

	resultRow := &fleet.ScheduledQueryResultRow{
		QueryID:     query.ID,
		HostID:      host.ID,
		LastFetched: time.Now(),
		Data: json.RawMessage(`{
			"model": "USB Keyboard",
			"vendor": "Apple Inc."
		}`),
	}

	result, err := ds.SaveQueryResultRow(context.Background(), resultRow)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, resultRow.QueryID, result.QueryID)
	require.Equal(t, resultRow.HostID, result.HostID)
	require.Equal(t, resultRow.LastFetched.Unix(), result.LastFetched.Unix())
	require.Equal(t, resultRow.Data, result.Data)
}

func getQueryResultRows(t *testing.T, ds *Datastore) {
	user := test.NewUser(t, ds, "Test User", "test@example.com", true)
	query := test.NewQuery(t, ds, nil, "New Query", "SELECT 1", user.ID, true)
	host := test.NewHost(t, ds, "hostname123", "192.168.1.100", "1234", "UI8XB1223", time.Now())

	mockTime := time.Now().UTC().Truncate(time.Millisecond)

	resultRow := &fleet.ScheduledQueryResultRow{
		QueryID:     query.ID,
		HostID:      host.ID,
		LastFetched: mockTime,
		Data: json.RawMessage(`{
			"model": "USB Keyboard",
			"vendor": "Apple Inc."
		}`),
	}

	// Insert 1 Result Row
	_, err := ds.SaveQueryResultRow(context.Background(), resultRow)
	require.NoError(t, err)

	results, err := ds.QueryResultRows(context.Background(), resultRow.QueryID, resultRow.HostID)
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Equal(t, resultRow.QueryID, results[0].QueryID)
	require.Equal(t, resultRow.HostID, results[0].HostID)
	require.Equal(t, resultRow.LastFetched.Unix(), results[0].LastFetched.Unix())
	require.JSONEq(t, string(resultRow.Data), string(results[0].Data))

	// Insert 2nd Result Row
	resultRow2 := &fleet.ScheduledQueryResultRow{
		QueryID:     query.ID,
		HostID:      host.ID,
		LastFetched: mockTime,
		Data: json.RawMessage(`{
			"model": "USB Mouse",
			"vendor": "Apple Inc."
		}`),
	}
	_, err = ds.SaveQueryResultRow(context.Background(), resultRow2)
	require.NoError(t, err)

	results, err = ds.QueryResultRows(context.Background(), resultRow.QueryID, resultRow.HostID)
	require.NoError(t, err)
	require.Len(t, results, 2)
	require.Equal(t, resultRow.QueryID, results[0].QueryID)
	require.Equal(t, resultRow.HostID, results[0].HostID)
	require.Equal(t, resultRow.LastFetched.Unix(), results[0].LastFetched.Unix())
	require.JSONEq(t, string(resultRow.Data), string(results[0].Data))
	require.Equal(t, resultRow2.QueryID, results[1].QueryID)
	require.Equal(t, resultRow2.HostID, results[1].HostID)
	require.Equal(t, resultRow2.LastFetched.Unix(), results[1].LastFetched.Unix())
	require.JSONEq(t, string(resultRow2.Data), string(results[1].Data))

	// Assert that QueryResultRows returns empty slice when no results are found
	results, err = ds.QueryResultRows(context.Background(), 999, 999)
	require.NoError(t, err)
	require.Len(t, results, 0)
}
