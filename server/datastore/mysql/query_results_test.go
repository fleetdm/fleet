package mysql

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/require"
)

func TestQueryResults(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"Get", testGetQueryResultRows},
		{"GetForHost", testGetQueryResultRowsForHost},
		{"CountForQuery", testCountResultsForQuery},
		{"CountForQueryAndHost", testCountResultsForQueryAndHost},
		{"Overwrite", testOverwriteQueryResultRows},
		{"MaxRows", testQueryResultRowsDoNotExceedMaxRows},
		{"QueryResultRows", testQueryResultRows},
		{"QueryResultRowsFilter", testQueryResultRowsTeamFilter},
		{"CleanupQueryResultRows", testCleanupQueryResultRows},
		{"CleanupExcessQueryResultRows", testCleanupExcessQueryResultRows},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)
			c.fn(t, ds)
		})
	}
}

func testGetQueryResultRows(t *testing.T, ds *Datastore) {
	user := test.NewUser(t, ds, "Test User", "test@example.com", true)
	query := test.NewQuery(t, ds, nil, "New Query", "SELECT 1", user.ID, true)
	host := test.NewHost(t, ds, "hostname123", "192.168.1.100", "1234", "UI8XB1223", time.Now())

	mockTime := time.Now().UTC().Truncate(time.Second)

	// Insert Result Rows for Query1
	query1Rows := []*fleet.ScheduledQueryResultRow{
		{
			QueryID:     query.ID,
			HostID:      host.ID,
			LastFetched: mockTime,
			Data:        nil,
		},
		{
			QueryID:     query.ID,
			HostID:      host.ID,
			LastFetched: mockTime,
			Data: ptr.RawMessage([]byte(`{
				"model": "USB Keyboard",
				"vendor": "Apple Inc."
			}`)),
		},
	}
	err := ds.OverwriteQueryResultRows(context.Background(), query1Rows, fleet.DefaultMaxQueryReportRows)
	require.NoError(t, err)

	// Insert Result Row for different Scheduled Query
	query2 := test.NewQuery(t, ds, nil, "New Query 2", "SELECT 1", user.ID, true)
	query2Rows := []*fleet.ScheduledQueryResultRow{
		{
			QueryID:     query2.ID,
			HostID:      host.ID,
			LastFetched: mockTime,
			Data:        ptr.RawMessage([]byte(`{"model": "USB Hub","vendor": "Logitech"}`)),
		},
	}

	err = ds.OverwriteQueryResultRows(context.Background(), query2Rows, fleet.DefaultMaxQueryReportRows)
	require.NoError(t, err)

	results, err := ds.QueryResultRows(context.Background(), query.ID, fleet.TeamFilter{User: test.UserAdmin})
	require.NoError(t, err)
	require.Len(t, results, 1) // Should not return rows with nil data
	require.Equal(t, query1Rows[1].QueryID, results[0].QueryID)
	require.Equal(t, query1Rows[1].HostID, results[0].HostID)
	require.Equal(t, query1Rows[1].LastFetched.Unix(), results[0].LastFetched.Unix())
	require.JSONEq(t, string(*query1Rows[1].Data), string(*results[0].Data))

	// Assert that Query2 returns 1 result
	results, err = ds.QueryResultRows(context.Background(), query2.ID, fleet.TeamFilter{User: test.UserAdmin})
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Equal(t, query2Rows[0].QueryID, results[0].QueryID)
	require.Equal(t, query2Rows[0].HostID, results[0].HostID)
	require.Equal(t, query2Rows[0].LastFetched.Unix(), results[0].LastFetched.Unix())
	require.JSONEq(t, string(*query2Rows[0].Data), string(*results[0].Data))

	// Assert that QueryResultRowsForHost returns empty slice when no results are found
	results, err = ds.QueryResultRowsForHost(context.Background(), 999, 999)
	require.NoError(t, err)
	require.Len(t, results, 0)
}

func testGetQueryResultRowsForHost(t *testing.T, ds *Datastore) {
	user := test.NewUser(t, ds, "Test User", "test@example.com", true)
	query := test.NewQuery(t, ds, nil, "New Query", "SELECT 1", user.ID, true)
	host1 := test.NewHost(t, ds, "hostname1", "192.168.1.100", "1111", "UI8XB1223", time.Now())
	host2 := test.NewHost(t, ds, "hostname2", "192.168.1.100", "2222", "UI8XB1223", time.Now())

	mockTime := time.Now().UTC().Truncate(time.Second)

	// Insert 2 Result Rows for Query1 Host1
	host1ResultRows := []*fleet.ScheduledQueryResultRow{
		{
			QueryID:     query.ID,
			HostID:      host1.ID,
			LastFetched: mockTime,
			Data:        nil,
		},
		{
			QueryID:     query.ID,
			HostID:      host1.ID,
			LastFetched: mockTime,
			Data:        ptr.RawMessage([]byte(`{"model": "USB Mouse", "vendor": "Logitech"}`)),
		},
	}
	err := ds.OverwriteQueryResultRows(context.Background(), host1ResultRows, fleet.DefaultMaxQueryReportRows)
	require.NoError(t, err)

	// Insert 1 Result Row for Query1 Host2
	host2ResultRows := []*fleet.ScheduledQueryResultRow{
		{
			QueryID:     query.ID,
			HostID:      host2.ID,
			LastFetched: mockTime,
			Data:        ptr.RawMessage([]byte(`{"model": "USB Mouse", "vendor": "Logitech"}`)),
		},
	}
	err = ds.OverwriteQueryResultRows(context.Background(), host2ResultRows, fleet.DefaultMaxQueryReportRows)
	require.NoError(t, err)

	// Assert that Query1 returns 2 results for Host1
	results, err := ds.QueryResultRowsForHost(context.Background(), query.ID, host1.ID)
	require.NoError(t, err)
	require.Len(t, results, 2) // should return rows with nil data
	require.Equal(t, host1ResultRows[0].QueryID, results[0].QueryID)
	require.Equal(t, host1ResultRows[0].HostID, results[0].HostID)
	require.Equal(t, host1ResultRows[0].LastFetched.Unix(), results[0].LastFetched.Unix())
	require.Nil(t, results[0].Data)
	require.Equal(t, host1ResultRows[1].QueryID, results[1].QueryID)
	require.Equal(t, host1ResultRows[1].HostID, results[1].HostID)
	require.Equal(t, host1ResultRows[1].LastFetched.Unix(), results[1].LastFetched.Unix())
	require.JSONEq(t, string(*host1ResultRows[1].Data), string(*results[1].Data))

	// Assert that Query1 returns 1 result for Host2
	results, err = ds.QueryResultRowsForHost(context.Background(), query.ID, host2.ID)
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Equal(t, host2ResultRows[0].QueryID, results[0].QueryID)
	require.Equal(t, host2ResultRows[0].HostID, results[0].HostID)
	require.Equal(t, host2ResultRows[0].LastFetched.Unix(), results[0].LastFetched.Unix())
	require.JSONEq(t, string(*host2ResultRows[0].Data), string(*results[0].Data))
}

func testQueryResultRowsTeamFilter(t *testing.T, ds *Datastore) {
	team, err := ds.NewTeam(context.Background(), &fleet.Team{
		Name: "teamFoo",
	})
	require.NoError(t, err)
	observerTeam, err := ds.NewTeam(context.Background(), &fleet.Team{
		Name: "observerTeam",
	})
	require.NoError(t, err)

	teamUser, err := ds.NewUser(context.Background(), &fleet.User{
		Password:   []byte("foo"),
		Salt:       "bar",
		Name:       "teamUser",
		Email:      "teamUser@example.com",
		GlobalRole: nil,
		Teams: []fleet.UserTeam{
			{
				Team: *team,
				Role: fleet.RoleAdmin,
			},
			{
				Team: *observerTeam,
				Role: fleet.RoleObserver,
			},
		},
	})
	require.NoError(t, err)

	query := test.NewQuery(t, ds, nil, "New Query", "SELECT 1", teamUser.ID, true)
	globalHost := test.NewHost(t, ds, "globalHost", "192.168.1.100", "1111", "UI8XB1223", time.Now())
	teamHost := test.NewHost(t, ds, "teamHost", "192.168.1.100", "2222", "UI8XB1223", time.Now())
	err = ds.AddHostsToTeam(context.Background(), fleet.NewAddHostsToTeamParams(&team.ID, []uint{teamHost.ID}))
	require.NoError(t, err)
	observerTeamHost := test.NewHost(t, ds, "teamHost", "192.168.1.100", "3333", "UI8XB1223", time.Now())
	err = ds.AddHostsToTeam(context.Background(), fleet.NewAddHostsToTeamParams(&observerTeam.ID, []uint{observerTeamHost.ID}))
	require.NoError(t, err)

	mockTime := time.Now().UTC().Truncate(time.Second)

	globalRow := []*fleet.ScheduledQueryResultRow{
		{
			QueryID:     query.ID,
			HostID:      globalHost.ID,
			LastFetched: mockTime,
			Data: ptr.RawMessage(json.RawMessage(`{
				"model": "Global USB Keyboard",
				"vendor": "Global Inc."
			}`)),
		},
	}

	err = ds.OverwriteQueryResultRows(context.Background(), globalRow, fleet.DefaultMaxQueryReportRows)
	require.NoError(t, err)

	teamRow := []*fleet.ScheduledQueryResultRow{
		{
			QueryID:     query.ID,
			HostID:      teamHost.ID,
			LastFetched: mockTime,
			Data: ptr.RawMessage(json.RawMessage(`{
				"model": "Team USB Keyboard",
				"vendor": "Team Inc."
			}`)),
		},
	}
	err = ds.OverwriteQueryResultRows(context.Background(), teamRow, fleet.DefaultMaxQueryReportRows)
	require.NoError(t, err)

	observerTeamRow := []*fleet.ScheduledQueryResultRow{
		{
			QueryID:     query.ID,
			HostID:      observerTeamHost.ID,
			LastFetched: mockTime,
			Data: ptr.RawMessage(json.RawMessage(`{
				"model": "Team USB Keyboard",
				"vendor": "Team Inc."
			}`)),
		},
	}
	err = ds.OverwriteQueryResultRows(context.Background(), observerTeamRow, fleet.DefaultMaxQueryReportRows)
	require.NoError(t, err)

	filter := fleet.TeamFilter{
		User:            teamUser,
		IncludeObserver: true,
	}

	results, err := ds.QueryResultRows(context.Background(), query.ID, filter)
	require.NoError(t, err)

	require.Len(t, results, 2)
	require.Equal(t, teamRow[0].HostID, results[0].HostID)
	require.Equal(t, teamRow[0].QueryID, results[0].QueryID)
	require.Equal(t, teamRow[0].LastFetched, results[0].LastFetched)
	require.JSONEq(t, string(*teamRow[0].Data), string(*results[0].Data))
	require.Equal(t, observerTeamRow[0].HostID, results[1].HostID)
	require.Equal(t, observerTeamRow[0].QueryID, results[1].QueryID)
	require.Equal(t, observerTeamRow[0].LastFetched, results[1].LastFetched)
	require.JSONEq(t, string(*observerTeamRow[0].Data), string(*results[1].Data))
}

func testCountResultsForQuery(t *testing.T, ds *Datastore) {
	user := test.NewUser(t, ds, "Test User", "test@example.com", true)
	query1 := test.NewQuery(t, ds, nil, "New Query", "SELECT 1", user.ID, true)
	query2 := test.NewQuery(t, ds, nil, "New Query 2", "SELECT 1", user.ID, true)
	host := test.NewHost(t, ds, "hostname1", "192.168.1.101", "1111", "UI8XB1223", time.Now())
	host2 := test.NewHost(t, ds, "hostname1", "192.168.1.102", "2222", "UI8XB1224", time.Now())

	mockTime := time.Now().UTC().Truncate(time.Second)

	// Insert 1 Result Row for Query1
	host1ResultRow := []*fleet.ScheduledQueryResultRow{
		{
			QueryID:     query1.ID,
			HostID:      host.ID,
			LastFetched: mockTime,
			Data: ptr.RawMessage([]byte(`{
				"model": "USB Keyboard",
				"vendor": "Apple Inc."
			}`)),
		},
	}
	err := ds.OverwriteQueryResultRows(context.Background(), host1ResultRow, fleet.DefaultMaxQueryReportRows)
	require.NoError(t, err)

	// Insert Nil Result Row for Query1, nil data rows are not counted
	host2ResultRow := []*fleet.ScheduledQueryResultRow{
		{
			QueryID:     query1.ID,
			HostID:      host2.ID,
			LastFetched: mockTime,
			Data:        nil,
		},
	}
	err = ds.OverwriteQueryResultRows(context.Background(), host2ResultRow, fleet.DefaultMaxQueryReportRows)
	require.NoError(t, err)

	// Insert 5 Result Rows for Query2
	resultRow2 := &fleet.ScheduledQueryResultRow{
		QueryID:     query2.ID,
		HostID:      host.ID,
		LastFetched: mockTime,
		Data: ptr.RawMessage([]byte(`{
				"model": "USB Mouse",
				"vendor": "Apple Inc."
			}`)),
	}

	var resultRows []*fleet.ScheduledQueryResultRow
	for i := 0; i < 5; i++ {
		resultRows = append(resultRows, resultRow2)
	}

	err = ds.OverwriteQueryResultRows(context.Background(), resultRows, fleet.DefaultMaxQueryReportRows)
	require.NoError(t, err)

	// Assert that ResultCountForQuery returns 1
	count, err := ds.ResultCountForQuery(context.Background(), query1.ID)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	// Assert that ResultCountForQuery returns 5
	count, err = ds.ResultCountForQuery(context.Background(), query2.ID)
	require.NoError(t, err)
	require.Equal(t, 5, count)

	// Returns 0 when no results are found
	count, err = ds.ResultCountForQuery(context.Background(), 999)
	require.NoError(t, err)
	require.Equal(t, 0, count)
}

func testCountResultsForQueryAndHost(t *testing.T, ds *Datastore) {
	user := test.NewUser(t, ds, "Test User", "test@example.com", true)
	query1 := test.NewQuery(t, ds, nil, "New Query", "SELECT 1", user.ID, true)
	query2 := test.NewQuery(t, ds, nil, "New Query 2", "SELECT 1", user.ID, true)
	host1 := test.NewHost(t, ds, "host1", "192.168.1.100", "1234", "UI8XB1223", time.Now())
	host2 := test.NewHost(t, ds, "host2", "192.168.1.101", "4567", "UI8XB1224", time.Now())
	host3 := test.NewHost(t, ds, "host3", "192.168.1.102", "8910", "UI8XB1225", time.Now())

	mockTime := time.Now().UTC().Truncate(time.Second)

	host1ResultRows := []*fleet.ScheduledQueryResultRow{
		{
			QueryID:     query1.ID,
			HostID:      host1.ID,
			LastFetched: mockTime,
			Data: ptr.RawMessage([]byte(`{
				"model": "USB Keyboard",
				"vendor": "Apple Inc."
			}`)),
		},
		{
			QueryID:     query1.ID,
			HostID:      host1.ID,
			LastFetched: mockTime,
			Data: ptr.RawMessage([]byte(`{
				"model": "USB Mouse",
				"vendor": "Logitech"
			}`)),
		},
	}
	err := ds.OverwriteQueryResultRows(context.Background(), host1ResultRows, fleet.DefaultMaxQueryReportRows)
	require.NoError(t, err)

	host1Query2 := []*fleet.ScheduledQueryResultRow{
		{
			QueryID:     query2.ID,
			HostID:      host1.ID,
			LastFetched: mockTime,
			Data: ptr.RawMessage([]byte(`{
				"model": "USB Mouse",
				"vendor": "Logitech"
			}`)),
		},
	}
	err = ds.OverwriteQueryResultRows(context.Background(), host1Query2, fleet.DefaultMaxQueryReportRows)
	require.NoError(t, err)

	host2ResultRow := []*fleet.ScheduledQueryResultRow{
		{
			QueryID:     query1.ID,
			HostID:      host2.ID,
			LastFetched: mockTime,
			Data: ptr.RawMessage([]byte(`{
				"model": "USB Mouse",
				"vendor": "Logitech"
			}`)),
		},
	}
	err = ds.OverwriteQueryResultRows(context.Background(), host2ResultRow, fleet.DefaultMaxQueryReportRows)
	require.NoError(t, err)

	host3ResultRow := []*fleet.ScheduledQueryResultRow{
		{
			QueryID:     query2.ID,
			HostID:      host3.ID,
			LastFetched: mockTime,
			Data:        nil,
		},
	}
	err = ds.OverwriteQueryResultRows(context.Background(), host3ResultRow, fleet.DefaultMaxQueryReportRows)
	require.NoError(t, err)

	// Assert that Query1 returns 2
	count, err := ds.ResultCountForQueryAndHost(context.Background(), query1.ID, host1.ID)
	require.NoError(t, err)
	require.Equal(t, 2, count)

	// Assert that ResultCountForQuery returns 1
	count, err = ds.ResultCountForQueryAndHost(context.Background(), query2.ID, host1.ID)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	// Assert that host2 returns 1 row
	count, err = ds.ResultCountForQueryAndHost(context.Background(), query1.ID, host2.ID)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	// Assert Nil Data rows are not counted
	count, err = ds.ResultCountForQueryAndHost(context.Background(), query2.ID, host3.ID)
	require.NoError(t, err)
	require.Zero(t, count)

	// Returns empty result when no results are found
	count, err = ds.ResultCountForQueryAndHost(context.Background(), 999, host1.ID)
	require.NoError(t, err)
	require.Zero(t, count)
}

func testOverwriteQueryResultRows(t *testing.T, ds *Datastore) {
	user := test.NewUser(t, ds, "Test User", "test@example.com", true)
	query := test.NewQuery(t, ds, nil, "Overwrite Test Query", "SELECT 1", user.ID, true)
	host := test.NewHost(t, ds, "hostname1234", "192.168.1.101", "12345", "UI8XB1224", time.Now())

	mockTime := time.Now().UTC().Truncate(time.Second)

	// Insert initial Result Rows
	initialRow := []*fleet.ScheduledQueryResultRow{
		{
			QueryID:     query.ID,
			HostID:      host.ID,
			LastFetched: mockTime,
			Data:        ptr.RawMessage([]byte(`{"model": "USB Keyboard", "vendor": "Apple Inc."}`)),
		},
	}

	err := ds.OverwriteQueryResultRows(context.Background(), initialRow, fleet.DefaultMaxQueryReportRows)
	require.NoError(t, err)

	// Overwrite Result Rows with new data
	newMockTime := mockTime.Add(2 * time.Minute)
	overwriteRows := []*fleet.ScheduledQueryResultRow{
		{
			QueryID:     query.ID,
			HostID:      host.ID,
			LastFetched: newMockTime,
			Data:        ptr.RawMessage([]byte(`{"model": "USB Mouse", "vendor": "Logitech"}`)),
		},
	}

	err = ds.OverwriteQueryResultRows(context.Background(), overwriteRows, fleet.DefaultMaxQueryReportRows)
	require.NoError(t, err)

	// Assert that we get the overwritten data (1 result with USB Mouse data)
	results, err := ds.QueryResultRowsForHost(context.Background(), overwriteRows[0].QueryID, overwriteRows[0].HostID)
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Equal(t, overwriteRows[0].QueryID, results[0].QueryID)
	require.Equal(t, overwriteRows[0].HostID, results[0].HostID)
	require.Equal(t, overwriteRows[0].LastFetched.Unix(), results[0].LastFetched.Unix())
	require.JSONEq(t, string(*overwriteRows[0].Data), string(*results[0].Data))

	// Test calling OverwriteQueryResultRows with a query that doesn't exist (e.g. a deleted query).
	overwriteRows = []*fleet.ScheduledQueryResultRow{
		{
			QueryID:     9999,
			HostID:      host.ID,
			LastFetched: newMockTime,
			Data:        ptr.RawMessage([]byte(`{"model": "USB Mouse", "vendor": "Logitech"}`)),
		},
	}
	err = ds.OverwriteQueryResultRows(context.Background(), overwriteRows, fleet.DefaultMaxQueryReportRows)
	require.NoError(t, err)

	// Assert that the data has not changed
	results, err = ds.QueryResultRowsForHost(context.Background(), overwriteRows[0].QueryID, overwriteRows[0].HostID)
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Equal(t, overwriteRows[0].QueryID, results[0].QueryID)
	require.Equal(t, overwriteRows[0].HostID, results[0].HostID)
	require.Equal(t, overwriteRows[0].LastFetched.Unix(), results[0].LastFetched.Unix())
	require.JSONEq(t, string(*overwriteRows[0].Data), string(*results[0].Data))
}

func testQueryResultRowsDoNotExceedMaxRows(t *testing.T, ds *Datastore) {
	// This test verifies that when a single host sends more than 1000 rows in one submission,
	// the rows are not stored (we bail early). The actual enforcement of the max rows limit
	// is done by the CleanupExcessQueryResultRows cron job.
	user := test.NewUser(t, ds, "Test User", "test@example.com", true)
	query := test.NewQuery(t, ds, nil, "Overwrite Test Query", "SELECT 1", user.ID, true)
	query2 := test.NewQuery(t, ds, nil, "Overwrite Test Query 2", "SELECT 1", user.ID, true)
	host1 := test.NewHost(t, ds, "hostname1", "192.168.1.101", "11111", "UI8XB1221", time.Now())
	host2 := test.NewHost(t, ds, "hostname2", "192.168.1.101", "22222", "UI8XB1222", time.Now())

	mockTime := time.Now().UTC().Truncate(time.Second)

	// Generate max rows (exactly 1000)
	maxRows := fleet.DefaultMaxQueryReportRows
	maxRowsBatch := make([]*fleet.ScheduledQueryResultRow, maxRows)
	for i := 0; i < maxRows; i++ {
		maxRowsBatch[i] = &fleet.ScheduledQueryResultRow{
			QueryID:     query.ID,
			HostID:      host1.ID,
			LastFetched: mockTime,
			Data:        ptr.RawMessage([]byte(`{"model": "USB Mouse", "vendor": "Logitech"}`)),
		}
	}
	err := ds.OverwriteQueryResultRows(context.Background(), maxRowsBatch, fleet.DefaultMaxQueryReportRows)
	require.NoError(t, err)

	// Verify that exactly 1000 rows were stored (1000 is the limit for a single submission)
	count, err := ds.ResultCountForQuery(context.Background(), query.ID)
	require.NoError(t, err)
	require.Equal(t, fleet.DefaultMaxQueryReportRows, count)

	// Generate more than max rows (1001+) for a single host submission - should bail early
	rows := fleet.DefaultMaxQueryReportRows + 50
	largeBatchRows := make([]*fleet.ScheduledQueryResultRow, rows)
	for i := 0; i < rows; i++ {
		largeBatchRows[i] = &fleet.ScheduledQueryResultRow{
			QueryID:     query2.ID,
			HostID:      host1.ID,
			LastFetched: mockTime,
			Data:        ptr.RawMessage([]byte(`{"model": "USB Mouse", "vendor": "Logitech"}`)),
		}
	}
	err = ds.OverwriteQueryResultRows(context.Background(), largeBatchRows, fleet.DefaultMaxQueryReportRows)
	require.NoError(t, err)

	// Confirm NO rows are stored when > 1000 rows in a single submission (we bail early)
	allResults, err := ds.QueryResultRowsForHost(context.Background(), query2.ID, host1.ID)
	require.NoError(t, err)
	require.Len(t, allResults, 0)

	// Add a small batch to query2 - should work fine
	smallBatch := []*fleet.ScheduledQueryResultRow{
		{
			QueryID:     query2.ID,
			HostID:      host2.ID,
			LastFetched: mockTime,
			Data:        ptr.RawMessage([]byte(`{"model": "USB Mouse", "vendor": "Logitech"}`)),
		},
	}
	err = ds.OverwriteQueryResultRows(context.Background(), smallBatch, fleet.DefaultMaxQueryReportRows)
	require.NoError(t, err)

	// Verify the small batch was stored
	host2Results, err := ds.QueryResultRowsForHost(context.Background(), query2.ID, host2.ID)
	require.NoError(t, err)
	require.Len(t, host2Results, 1)
}

func testQueryResultRows(t *testing.T, ds *Datastore) {
	user := test.NewUser(t, ds, "Test User", "test@example.com", true)
	query := test.NewQuery(t, ds, nil, "Overwrite Test Query", "SELECT 1", user.ID, true)

	mockTime := time.Now().UTC().Truncate(time.Second)

	overwriteRows := []*fleet.ScheduledQueryResultRow{
		{
			QueryID:     query.ID,
			HostID:      9999,
			LastFetched: mockTime,
			Data:        ptr.RawMessage([]byte(`{"model": "USB Mouse", "vendor": "Logitech"}`)),
		},
	}
	err := ds.OverwriteQueryResultRows(context.Background(), overwriteRows, fleet.DefaultMaxQueryReportRows)
	require.NoError(t, err)

	filter := fleet.TeamFilter{User: user, IncludeObserver: true}

	// Test calling QueryResultRows with a query that has an entry with a host that doesn't exist anymore.
	results, err := ds.QueryResultRows(context.Background(), query.ID, filter)
	require.NoError(t, err)
	require.Len(t, results, 1)
}

func testCleanupQueryResultRows(t *testing.T, ds *Datastore) {
	user := test.NewUser(t, ds, "Test User", "test@example.com", true)
	queryNoDiscard := test.NewQuery(t, ds, nil, "Query No Discard", "SELECT 1", user.ID, true)
	queryDiscardTrue := test.NewQuery(t, ds, nil, "Query Discard True", "SELECT 1", user.ID, true)
	queryDiscardTrue.DiscardData = true
	err := ds.SaveQuery(context.Background(), queryDiscardTrue, false, false)
	require.NoError(t, err)

	mockTime := time.Now().UTC().Truncate(time.Second)

	// Insert query result rows
	rows := []*fleet.ScheduledQueryResultRow{
		{
			QueryID:     queryNoDiscard.ID,
			HostID:      1,
			LastFetched: mockTime,
			Data:        ptr.RawMessage([]byte(`{"model": "USB Mouse", "vendor": "Logitech"}`)),
		},
		{
			QueryID:     queryNoDiscard.ID,
			HostID:      1,
			LastFetched: mockTime,
			Data:        ptr.RawMessage([]byte(`{"model": "Keyboard", "vendor": "Microsoft"}`)),
		},
	}
	err = ds.OverwriteQueryResultRows(context.Background(), rows, fleet.DefaultMaxQueryReportRows)
	require.NoError(t, err)

	// Call OverwriteQueryResultRows again with different rows
	overwriteRows := []*fleet.ScheduledQueryResultRow{
		{
			QueryID:     queryDiscardTrue.ID,
			HostID:      1,
			LastFetched: mockTime,
			Data:        ptr.RawMessage([]byte(`{"model": "Headphones", "vendor": "Sony"}`)),
		},
		{
			QueryID:     queryDiscardTrue.ID,
			HostID:      1,
			LastFetched: mockTime,
			Data:        ptr.RawMessage([]byte(`{"model": "Speakers", "vendor": "Bose"}`)),
		},
	}
	err = ds.OverwriteQueryResultRows(context.Background(), overwriteRows, fleet.DefaultMaxQueryReportRows)
	require.NoError(t, err)

	// Cleanup query result rows
	err = ds.CleanupDiscardedQueryResults(context.Background())
	require.NoError(t, err)

	// Verify that the rows with discard data set to false are not removed
	results, err := ds.QueryResultRows(context.Background(), queryNoDiscard.ID, fleet.TeamFilter{User: user})
	require.NoError(t, err)
	require.Len(t, results, 2)

	// Verify that the rows with discard data set to true are removed
	results, err = ds.QueryResultRows(context.Background(), queryDiscardTrue.ID, fleet.TeamFilter{User: user})
	require.NoError(t, err)
	require.Len(t, results, 0)
}

func testCleanupExcessQueryResultRows(t *testing.T, ds *Datastore) {
	user := test.NewUser(t, ds, "Test User", "test@example.com", true)
	query := test.NewQuery(t, ds, nil, "Query With Results", "SELECT 1", user.ID, true)
	mockTime := time.Now().UTC().Truncate(time.Second)
	maxRows := 10

	// Create 15 hosts and insert 1 row per host (need different hosts since
	// OverwriteQueryResultRows deletes existing rows for the same host/query)
	for i := 0; i < 15; i++ {
		host := test.NewHost(t, ds, "host"+string(rune('a'+i)), "192.168.1.100", "serial"+string(rune('a'+i)), "uuid"+string(rune('a'+i)), time.Now())
		rows := []*fleet.ScheduledQueryResultRow{{
			QueryID:     query.ID,
			HostID:      host.ID,
			LastFetched: mockTime.Add(time.Duration(i) * time.Minute),
			Data:        ptr.RawMessage([]byte(`{"index": ` + string(rune('0'+i%10)) + `}`)),
		}}
		require.NoError(t, ds.OverwriteQueryResultRows(context.Background(), rows, fleet.DefaultMaxQueryReportRows))
	}

	// Verify we have 15 rows
	count, err := ds.ResultCountForQuery(context.Background(), query.ID)
	require.NoError(t, err)
	require.Equal(t, 15, count)

	// Run cleanup with maxRows = 10, using a small batch size to test batching
	opts := fleet.CleanupExcessQueryResultRowsOptions{BatchSize: 2}
	require.NoError(t, ds.CleanupExcessQueryResultRows(context.Background(), maxRows, opts))

	// Verify only 10 rows remain
	count, err = ds.ResultCountForQuery(context.Background(), query.ID)
	require.NoError(t, err)
	require.Equal(t, maxRows, count)

	// Verify the most recent rows were kept
	results, err := ds.QueryResultRows(context.Background(), query.ID, fleet.TeamFilter{User: user})
	require.NoError(t, err)
	require.Len(t, results, maxRows)
}
