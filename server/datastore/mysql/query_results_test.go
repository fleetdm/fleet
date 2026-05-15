package mysql

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
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
		{"CleanupExcessQueryResultRowsManyQueries", testCleanupExcessQueryResultRowsManyQueries},
		{"ListHostReports", testListHostReports},
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
	_, err := ds.OverwriteQueryResultRows(context.Background(), query1Rows, fleet.DefaultMaxQueryReportRows)
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

	_, err = ds.OverwriteQueryResultRows(context.Background(), query2Rows, fleet.DefaultMaxQueryReportRows)
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
	_, err := ds.OverwriteQueryResultRows(context.Background(), host1ResultRows, fleet.DefaultMaxQueryReportRows)
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
	_, err = ds.OverwriteQueryResultRows(context.Background(), host2ResultRows, fleet.DefaultMaxQueryReportRows)
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

	_, err = ds.OverwriteQueryResultRows(context.Background(), globalRow, fleet.DefaultMaxQueryReportRows)
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
	_, err = ds.OverwriteQueryResultRows(context.Background(), teamRow, fleet.DefaultMaxQueryReportRows)
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
	_, err = ds.OverwriteQueryResultRows(context.Background(), observerTeamRow, fleet.DefaultMaxQueryReportRows)
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
	_, err := ds.OverwriteQueryResultRows(context.Background(), host1ResultRow, fleet.DefaultMaxQueryReportRows)
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
	_, err = ds.OverwriteQueryResultRows(context.Background(), host2ResultRow, fleet.DefaultMaxQueryReportRows)
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

	_, err = ds.OverwriteQueryResultRows(context.Background(), resultRows, fleet.DefaultMaxQueryReportRows)
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
	_, err := ds.OverwriteQueryResultRows(context.Background(), host1ResultRows, fleet.DefaultMaxQueryReportRows)
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
	_, err = ds.OverwriteQueryResultRows(context.Background(), host1Query2, fleet.DefaultMaxQueryReportRows)
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
	_, err = ds.OverwriteQueryResultRows(context.Background(), host2ResultRow, fleet.DefaultMaxQueryReportRows)
	require.NoError(t, err)

	host3ResultRow := []*fleet.ScheduledQueryResultRow{
		{
			QueryID:     query2.ID,
			HostID:      host3.ID,
			LastFetched: mockTime,
			Data:        nil,
		},
	}
	_, err = ds.OverwriteQueryResultRows(context.Background(), host3ResultRow, fleet.DefaultMaxQueryReportRows)
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

	rowsAdded, err := ds.OverwriteQueryResultRows(context.Background(), initialRow, fleet.DefaultMaxQueryReportRows)
	require.NoError(t, err)
	require.Equal(t, 1, rowsAdded)

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

	rowsAdded, err = ds.OverwriteQueryResultRows(context.Background(), overwriteRows, fleet.DefaultMaxQueryReportRows)
	require.NoError(t, err)
	// rowsAdded is zero here because we deleted one row and inserted one.
	require.Equal(t, 0, rowsAdded)

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
	rowsAdded, err = ds.OverwriteQueryResultRows(context.Background(), overwriteRows, fleet.DefaultMaxQueryReportRows)
	require.NoError(t, err)
	require.Equal(t, 1, rowsAdded)

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
	_, err := ds.OverwriteQueryResultRows(context.Background(), maxRowsBatch, fleet.DefaultMaxQueryReportRows)
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
	_, err = ds.OverwriteQueryResultRows(context.Background(), largeBatchRows, fleet.DefaultMaxQueryReportRows)
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
	_, err = ds.OverwriteQueryResultRows(context.Background(), smallBatch, fleet.DefaultMaxQueryReportRows)
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
	_, err := ds.OverwriteQueryResultRows(context.Background(), overwriteRows, fleet.DefaultMaxQueryReportRows)
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
	_, err = ds.OverwriteQueryResultRows(context.Background(), rows, fleet.DefaultMaxQueryReportRows)
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
	_, err = ds.OverwriteQueryResultRows(context.Background(), overwriteRows, fleet.DefaultMaxQueryReportRows)
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
	query2 := test.NewQuery(t, ds, nil, "Query Without Results", "SELECT 1", user.ID, true)
	query3 := test.NewQuery(t, ds, nil, "Query With Some Results", "SELECT 1", user.ID, true)

	mockTime := time.Now().UTC().Truncate(time.Second)
	maxRows := 10

	// Create 15 hosts and insert 1 row per host (need different hosts since
	// OverwriteQueryResultRows deletes existing rows for the same host/query)
	for i := range 25 {
		host := test.NewHost(t, ds, "host"+string(rune('a'+i)), "192.168.1.100", "serial"+string(rune('a'+i)), "uuid"+string(rune('a'+i)), time.Now())
		rowsForQuery1 := []*fleet.ScheduledQueryResultRow{{
			QueryID:     query.ID,
			HostID:      host.ID,
			LastFetched: mockTime.Add(time.Duration(i) * time.Minute),
			Data:        ptr.RawMessage([]byte(`{"index": ` + string(rune('0'+i%10)) + `}`)),
		}}
		rowsForQuery2 := []*fleet.ScheduledQueryResultRow{{
			QueryID:     query2.ID,
			HostID:      host.ID,
			LastFetched: mockTime.Add(time.Duration(i) * time.Minute),
			Data:        nil,
		}}
		dataForQuery3 := ptr.RawMessage(nil)
		if i%2 == 0 {
			dataForQuery3 = ptr.RawMessage([]byte(`{"index": ` + string(rune('0'+i%10)) + `}`))
		}
		rowsForQuery3 := []*fleet.ScheduledQueryResultRow{{
			QueryID:     query3.ID,
			HostID:      host.ID,
			LastFetched: mockTime.Add(time.Duration(i) * time.Minute),
			Data:        dataForQuery3,
		}}
		_, err := ds.OverwriteQueryResultRows(context.Background(), rowsForQuery1, fleet.DefaultMaxQueryReportRows)
		require.NoError(t, err)
		_, err = ds.OverwriteQueryResultRows(context.Background(), rowsForQuery2, fleet.DefaultMaxQueryReportRows)
		require.NoError(t, err)
		_, err = ds.OverwriteQueryResultRows(context.Background(), rowsForQuery3, fleet.DefaultMaxQueryReportRows)
		require.NoError(t, err)
	}

	// Verify we have 25 rows for the data query
	count, err := ds.ResultCountForQuery(context.Background(), query.ID)
	require.NoError(t, err)
	require.Equal(t, 25, count)

	// Verify we have 0 rows for the non-data query
	count, err = ds.ResultCountForQuery(context.Background(), query2.ID)
	require.NoError(t, err)
	require.Equal(t, 0, count)

	// Verify we have 13 rows for the some-data query
	count, err = ds.ResultCountForQuery(context.Background(), query3.ID)
	require.NoError(t, err)
	require.Equal(t, 13, count)

	// Run cleanup with maxRows = 10, using a small batch size to test batching
	opts := fleet.CleanupExcessQueryResultRowsOptions{BatchSize: 2}
	queryCounts, err := ds.CleanupExcessQueryResultRows(context.Background(), maxRows, opts)
	require.NoError(t, err)
	require.Contains(t, queryCounts, query.ID)
	require.Equal(t, maxRows, queryCounts[query.ID])
	require.Contains(t, queryCounts, query2.ID)
	require.Equal(t, 0, queryCounts[query2.ID])
	require.Contains(t, queryCounts, query3.ID)
	require.Equal(t, maxRows, queryCounts[query3.ID])

	// Verify only 10 rows remain for query1
	count, err = ds.ResultCountForQuery(context.Background(), query.ID)
	require.NoError(t, err)
	require.Equal(t, maxRows, count)

	// Verify the most recent rows were kept
	results, err := ds.QueryResultRows(context.Background(), query.ID, fleet.TeamFilter{User: user})
	require.NoError(t, err)
	require.Len(t, results, maxRows)

	// Verify 0 rows remain for query2
	count, err = ds.ResultCountForQuery(context.Background(), query2.ID)
	require.NoError(t, err)
	require.Equal(t, 0, count)

	// Check that no rows were actually deleted for query2
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		stmt := "SELECT COUNT(*) FROM query_results WHERE query_id = ?"
		var result int
		require.NoError(t, sqlx.GetContext(context.Background(), q, &result, stmt, query2.ID))
		assert.Equal(t, 25, result)
		return nil
	})

	// Verify 10 rows remain for query3
	count, err = ds.ResultCountForQuery(context.Background(), query2.ID)
	require.NoError(t, err)
	require.Equal(t, 0, count)

	// Check that we actually have 22 rows (the 10 with data, and the 12 without data)
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		stmt := "SELECT COUNT(*) FROM query_results WHERE query_id = ?"
		var result int
		require.NoError(t, sqlx.GetContext(context.Background(), q, &result, stmt, query3.ID))
		assert.Equal(t, 22, result)
		return nil
	})
}

// testCleanupExcessQueryResultRowsManyQueries verifies that CleanupExcessQueryResultRows
// works when there are more queries than MySQL's prepared statement placeholder limit (65,535).
func testCleanupExcessQueryResultRowsManyQueries(t *testing.T, ds *Datastore) {
	const numQueries = 70000

	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(t.Context(),
			`INSERT INTO users (name, email, password, salt) VALUES ('test', 'bulk@test.com', 'x', 'x')`)
		if err != nil {
			return err
		}

		_, err = q.ExecContext(t.Context(), `
			INSERT INTO queries (name, description, query, author_id, logging_type, discard_data, saved)
			SELECT
				CONCAT('bulk_query_', seq),
				'',
				'SELECT 1',
				(SELECT id FROM users LIMIT 1),
				'snapshot',
				false,
				1
			FROM (
				SELECT a.N + b.N*10 + c.N*100 + d.N*1000 + e.N*10000 as seq
				FROM
					(SELECT 0 AS N UNION SELECT 1 UNION SELECT 2 UNION SELECT 3 UNION SELECT 4 UNION SELECT 5 UNION SELECT 6 UNION SELECT 7 UNION SELECT 8 UNION SELECT 9) a,
					(SELECT 0 AS N UNION SELECT 1 UNION SELECT 2 UNION SELECT 3 UNION SELECT 4 UNION SELECT 5 UNION SELECT 6 UNION SELECT 7 UNION SELECT 8 UNION SELECT 9) b,
					(SELECT 0 AS N UNION SELECT 1 UNION SELECT 2 UNION SELECT 3 UNION SELECT 4 UNION SELECT 5 UNION SELECT 6 UNION SELECT 7 UNION SELECT 8 UNION SELECT 9) c,
					(SELECT 0 AS N UNION SELECT 1 UNION SELECT 2 UNION SELECT 3 UNION SELECT 4 UNION SELECT 5 UNION SELECT 6 UNION SELECT 7 UNION SELECT 8 UNION SELECT 9) d,
					(SELECT 0 AS N UNION SELECT 1 UNION SELECT 2 UNION SELECT 3 UNION SELECT 4 UNION SELECT 5 UNION SELECT 6 UNION SELECT 7 UNION SELECT 8 UNION SELECT 9) e
			) numbers
			WHERE seq < ?
		`, numQueries)
		return err
	})

	var count int
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(t.Context(), q, &count,
			`SELECT COUNT(*) FROM queries WHERE discard_data = false AND logging_type = 'snapshot'`)
	})
	require.Equal(t, numQueries, count)

	queryCounts, err := ds.CleanupExcessQueryResultRows(t.Context(), fleet.DefaultMaxQueryReportRows)
	require.NoError(t, err)
	require.Len(t, queryCounts, numQueries)
}

func testListHostReports(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	user := test.NewUser(t, ds, "Test User", "list@example.com", true)
	host := test.NewHost(t, ds, "host1", "192.168.1.1", "key1", "serial1", time.Now())

	now := time.Now().UTC().Truncate(time.Second)
	earlier := now.Add(-time.Hour)

	// Create queries: two that save results, one that discards them.
	qSave1 := test.NewQuery(t, ds, nil, "Save Query Alpha", "SELECT 1", user.ID, true)
	// Override DiscardData so it saves results (default is false, which is correct already).

	_ = test.NewQuery(t, ds, nil, "Save Query Beta", "SELECT 2", user.ID, true)

	// Create a query that discards results; excluded by default since it doesn't
	// satisfy discard_data=0 AND logging_type='snapshot'.
	qDiscard, err := ds.NewQuery(ctx, &fleet.Query{
		Name:        "Discard Query Gamma",
		Query:       "SELECT 3",
		AuthorID:    &user.ID,
		Saved:       true,
		DiscardData: true,
		Logging:     fleet.LoggingDifferential,
	})
	require.NoError(t, err)

	// Create a query with discard_data=false but logging_type='differential'.
	// This is the edge case fixed by the StoreResults check: even though
	// discard_data=0, it does not store snapshot reports, so StoreResults must
	// be false and it must be excluded by the default filter.
	qDifferentialNoDiscard, err := ds.NewQuery(ctx, &fleet.Query{
		Name:        "Differential No Discard Delta",
		Query:       "SELECT 4",
		AuthorID:    &user.ID,
		Saved:       true,
		DiscardData: false,
		Logging:     fleet.LoggingDifferential,
	})
	require.NoError(t, err)

	// Insert results for qSave1 on our host: two rows.
	rows1 := []*fleet.ScheduledQueryResultRow{
		{
			QueryID:     qSave1.ID,
			HostID:      host.ID,
			LastFetched: earlier,
			Data:        ptr.RawMessage([]byte(`{"col":"row1"}`)),
		},
		{
			QueryID:     qSave1.ID,
			HostID:      host.ID,
			LastFetched: now,
			Data:        ptr.RawMessage([]byte(`{"col":"row2"}`)),
		},
	}
	_, err = ds.OverwriteQueryResultRows(ctx, rows1, fleet.DefaultMaxQueryReportRows)
	require.NoError(t, err)

	// qSave2 has no results yet (should still be returned when SaveResults=true).

	// Insert a result for qDiscard (to confirm it's excluded by default).
	rowsDiscard := []*fleet.ScheduledQueryResultRow{
		{
			QueryID:     qDiscard.ID,
			HostID:      host.ID,
			LastFetched: now,
			Data:        ptr.RawMessage([]byte(`{"col":"discarded"}`)),
		},
	}
	_, err = ds.OverwriteQueryResultRows(ctx, rowsDiscard, fleet.DefaultMaxQueryReportRows)
	require.NoError(t, err)

	t.Run("default_excludes_dont_store_results_queries", func(t *testing.T) {
		opts := fleet.ListHostReportsOptions{
			// IncludeReportsDontStoreResults defaults to false: only include discard_data=0 AND logging_type='snapshot'.
			ListOptions: fleet.ListOptions{OrderKey: "name", IncludeMetadata: true},
		}
		reports, total, meta, err := ds.ListHostReports(ctx, host.ID, nil, "", opts, fleet.DefaultMaxQueryReportRows)
		require.NoError(t, err)
		// Should get qSave1 and qSave2 but NOT qDiscard (doesn't satisfy discard_data=0 AND logging_type='snapshot').
		assert.Equal(t, 2, total)
		require.Len(t, reports, 2)
		assert.NotNil(t, meta)
		// Sorted by name ASC: "Save Query Alpha", "Save Query Beta".
		assert.Equal(t, "Save Query Alpha", reports[0].Name)
		assert.Equal(t, "Save Query Beta", reports[1].Name)
	})

	t.Run("include_reports_dont_store_results_returns_all_queries", func(t *testing.T) {
		opts := fleet.ListHostReportsOptions{
			IncludeReportsDontStoreResults: true,
			ListOptions:                    fleet.ListOptions{OrderKey: "name", IncludeMetadata: true},
		}
		reports, total, _, err := ds.ListHostReports(ctx, host.ID, nil, "", opts, fleet.DefaultMaxQueryReportRows)
		require.NoError(t, err)
		// All 4 queries are returned including both don't-store-results variants.
		assert.Equal(t, 4, total)
		require.Len(t, reports, 4)
		assert.Equal(t, "Differential No Discard Delta", reports[0].Name)
		assert.Equal(t, "Discard Query Gamma", reports[1].Name)
		assert.Equal(t, "Save Query Alpha", reports[2].Name)
		assert.Equal(t, "Save Query Beta", reports[3].Name)
	})

	t.Run("store_results_field_reflects_both_discard_data_and_logging_type", func(t *testing.T) {
		// This subtest validates the fix: StoreResults must be true only when
		// discard_data=0 AND logging_type='snapshot'. A query with discard_data=0
		// but logging_type='differential' must have StoreResults=false.
		opts := fleet.ListHostReportsOptions{
			IncludeReportsDontStoreResults: true,
			ListOptions:                    fleet.ListOptions{OrderKey: "name"},
		}
		reports, _, _, err := ds.ListHostReports(ctx, host.ID, nil, "", opts, fleet.DefaultMaxQueryReportRows)
		require.NoError(t, err)
		require.Len(t, reports, 4)

		byName := make(map[string]*fleet.HostReport, len(reports))
		for _, r := range reports {
			byName[r.Name] = r
		}

		// discard_data=false, logging_type='snapshot' → StoreResults=true
		require.Contains(t, byName, qSave1.Name)
		assert.True(t, byName[qSave1.Name].StoreResults, "snapshot query should have StoreResults=true")

		// discard_data=true, logging_type='differential' → StoreResults=false
		require.Contains(t, byName, qDiscard.Name)
		assert.False(t, byName[qDiscard.Name].StoreResults, "discard query should have StoreResults=false")

		// discard_data=false, logging_type='differential' → StoreResults=false (the fixed edge case)
		require.Contains(t, byName, qDifferentialNoDiscard.Name)
		assert.False(t, byName[qDifferentialNoDiscard.Name].StoreResults, "differential query with discard_data=false should have StoreResults=false")
	})

	t.Run("first_result_is_most_recent_non_null_row", func(t *testing.T) {
		opts := fleet.ListHostReportsOptions{
			ListOptions: fleet.ListOptions{OrderKey: "name"},
		}
		reports, _, _, err := ds.ListHostReports(ctx, host.ID, nil, "", opts, fleet.DefaultMaxQueryReportRows)
		require.NoError(t, err)
		require.Len(t, reports, 2)

		// qSave1 has 2 results; first result should be the most recent one.
		alpha := reports[0]
		assert.Equal(t, "Save Query Alpha", alpha.Name)
		require.NotNil(t, alpha.FirstResult)
		assert.Equal(t, "row2", alpha.FirstResult["col"])
		// LastFetched should be MAX(last_fetched) = now.
		require.NotNil(t, alpha.LastFetched)
		assert.Equal(t, now.Unix(), alpha.LastFetched.Unix())
		assert.Equal(t, 2, alpha.NHostResults)
	})

	t.Run("query_with_no_results_has_nil_first_result", func(t *testing.T) {
		opts := fleet.ListHostReportsOptions{
			ListOptions: fleet.ListOptions{OrderKey: "name"},
		}
		reports, _, _, err := ds.ListHostReports(ctx, host.ID, nil, "", opts, fleet.DefaultMaxQueryReportRows)
		require.NoError(t, err)
		require.Len(t, reports, 2)

		// qSave2 has no results.
		beta := reports[1]
		assert.Equal(t, "Save Query Beta", beta.Name)
		assert.Nil(t, beta.FirstResult)
		assert.Nil(t, beta.LastFetched)
		assert.Equal(t, 0, beta.NHostResults)
	})

	t.Run("name_filter", func(t *testing.T) {
		opts := fleet.ListHostReportsOptions{
			ListOptions: fleet.ListOptions{
				OrderKey:   "name",
				MatchQuery: "Alpha",
			},
		}
		reports, total, _, err := ds.ListHostReports(ctx, host.ID, nil, "", opts, fleet.DefaultMaxQueryReportRows)
		require.NoError(t, err)
		assert.Equal(t, 1, total)
		require.Len(t, reports, 1)
		assert.Equal(t, "Save Query Alpha", reports[0].Name)
	})

	t.Run("order_by_last_fetched_nulls_last", func(t *testing.T) {
		// qSave1 has results (non-null last_fetched), qSave2 has none (null).
		// NULLs should sort last regardless of ASC/DESC direction.

		// ASC: non-null values first (oldest to newest), NULLs at bottom.
		opts := fleet.ListHostReportsOptions{
			ListOptions: fleet.ListOptions{
				OrderKey:       "last_fetched",
				OrderDirection: fleet.OrderAscending,
			},
		}
		reports, _, _, err := ds.ListHostReports(ctx, host.ID, nil, "", opts, fleet.DefaultMaxQueryReportRows)
		require.NoError(t, err)
		require.Len(t, reports, 2)
		assert.Equal(t, "Save Query Alpha", reports[0].Name) // has results
		assert.Equal(t, "Save Query Beta", reports[1].Name)  // no results → NULL last

		// DESC: non-null values first (newest to oldest), NULLs at bottom.
		opts.ListOptions.OrderDirection = fleet.OrderDescending
		reports, _, _, err = ds.ListHostReports(ctx, host.ID, nil, "", opts, fleet.DefaultMaxQueryReportRows)
		require.NoError(t, err)
		require.Len(t, reports, 2)
		assert.Equal(t, "Save Query Alpha", reports[0].Name) // has results
		assert.Equal(t, "Save Query Beta", reports[1].Name)  // no results → NULL last
	})

	t.Run("pagination", func(t *testing.T) {
		// Use PerPage:1 over the 2 save queries to exercise both pages.
		opts := fleet.ListHostReportsOptions{
			ListOptions: fleet.ListOptions{
				OrderKey:        "name",
				PerPage:         1,
				Page:            0,
				IncludeMetadata: true,
			},
		}
		reports, total, meta, err := ds.ListHostReports(ctx, host.ID, nil, "", opts, fleet.DefaultMaxQueryReportRows)
		require.NoError(t, err)
		assert.Equal(t, 2, total)
		require.Len(t, reports, 1)
		require.NotNil(t, meta)
		assert.True(t, meta.HasNextResults)
		assert.False(t, meta.HasPreviousResults)

		// Second page.
		opts.ListOptions.Page = 1
		reports2, total2, meta2, err := ds.ListHostReports(ctx, host.ID, nil, "", opts, fleet.DefaultMaxQueryReportRows)
		require.NoError(t, err)
		assert.Equal(t, 2, total2)
		require.Len(t, reports2, 1)
		require.NotNil(t, meta2)
		assert.False(t, meta2.HasNextResults)
		assert.True(t, meta2.HasPreviousResults)
	})

	t.Run("team_scoping_excludes_other_team_queries", func(t *testing.T) {
		// Create a team first, then a query for that team.
		// Since host has no team, only global queries (team_id IS NULL) should be shown.
		team, err := ds.NewTeam(ctx, &fleet.Team{Name: "scoping-test-team"})
		require.NoError(t, err)
		_, err = ds.NewQuery(ctx, &fleet.Query{
			Name:     "Team-Only Query",
			Query:    "SELECT 4",
			AuthorID: &user.ID,
			Saved:    true,
			TeamID:   &team.ID,
			Logging:  fleet.LoggingSnapshot,
		})
		require.NoError(t, err)

		opts := fleet.ListHostReportsOptions{
			ListOptions: fleet.ListOptions{OrderKey: "name"},
		}
		reports, total, _, err := ds.ListHostReports(ctx, host.ID, nil, "", opts, fleet.DefaultMaxQueryReportRows)
		require.NoError(t, err)
		// Team-Only query should not appear because host has no team.
		assert.Equal(t, 2, total)
		for _, r := range reports {
			assert.NotEqual(t, "Team-Only Query", r.Name)
		}
	})

	t.Run("team_host_sees_global_and_team_queries", func(t *testing.T) {
		// Create a team, a team host, and a team-scoped query.
		// The host should see both global queries and its own team's queries.
		team, err := ds.NewTeam(ctx, &fleet.Team{Name: "team-host-test-team"})
		require.NoError(t, err)
		teamHost := test.NewHost(t, ds, "team-host1", "10.0.0.1", "key-team1", "serial-team1", time.Now())
		err = ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&team.ID, []uint{teamHost.ID}))
		require.NoError(t, err)

		_ = test.NewQuery(t, ds, &team.ID, "Team Query Zeta", "SELECT 5", user.ID, true)

		opts := fleet.ListHostReportsOptions{
			ListOptions: fleet.ListOptions{OrderKey: "name"},
		}
		reports, total, _, err := ds.ListHostReports(ctx, teamHost.ID, &team.ID, "", opts, fleet.DefaultMaxQueryReportRows)
		require.NoError(t, err)
		// Should see the 2 global save queries plus the team-scoped query.
		assert.Equal(t, 3, total)
		names := make([]string, 0, len(reports))
		for _, r := range reports {
			names = append(names, r.Name)
		}
		assert.Contains(t, names, "Save Query Alpha")
		assert.Contains(t, names, "Save Query Beta")
		assert.Contains(t, names, "Team Query Zeta")
	})

	t.Run("report_clipped_when_total_results_reach_cap", func(t *testing.T) {
		// Insert exactly maxQueryReportRows results for qSave1 across multiple hosts
		// so that n_query_results >= capacity, making report_clipped=true.
		capacity := 3
		extraHost := test.NewHost(t, ds, "extra-host", "192.168.2.1", "key2", "serial2", time.Now())
		t.Cleanup(func() {
			ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
				_, err := q.ExecContext(ctx, `DELETE FROM query_results WHERE host_id = ?`, extraHost.ID)
				return err
			})
		})
		_, err := ds.OverwriteQueryResultRows(ctx, []*fleet.ScheduledQueryResultRow{
			{QueryID: qSave1.ID, HostID: extraHost.ID, LastFetched: now, Data: ptr.RawMessage([]byte(`{"col":"extra"}`))},
		}, fleet.DefaultMaxQueryReportRows)
		require.NoError(t, err)
		// At this point qSave1 has 2 rows on host + 1 on extraHost = 3 total, which equals cap.

		opts := fleet.ListHostReportsOptions{
			ListOptions: fleet.ListOptions{OrderKey: "name"},
		}
		reports, _, _, err := ds.ListHostReports(ctx, host.ID, nil, "", opts, capacity)
		require.NoError(t, err)
		require.Len(t, reports, 2)

		alpha := reports[0]
		assert.Equal(t, "Save Query Alpha", alpha.Name)
		assert.True(t, alpha.ReportClipped)

		// qSave2 has no results, so it should not be clipped.
		beta := reports[1]
		assert.Equal(t, "Save Query Beta", beta.Name)
		assert.False(t, beta.ReportClipped)
	})

	t.Run("platform_filtering", func(t *testing.T) {
		// Create a darwin-only query and a linux-only query.
		qDarwin, err := ds.NewQuery(ctx, &fleet.Query{
			Name:     "Darwin Only Query",
			Query:    "SELECT 7",
			AuthorID: &user.ID,
			Saved:    true,
			Logging:  fleet.LoggingSnapshot,
			Platform: "darwin",
		})
		require.NoError(t, err)

		_, err = ds.NewQuery(ctx, &fleet.Query{
			Name:     "Linux Only Query",
			Query:    "SELECT 8",
			AuthorID: &user.ID,
			Saved:    true,
			Logging:  fleet.LoggingSnapshot,
			Platform: "linux",
		})
		require.NoError(t, err)

		// host has platform "darwin" (set by test.NewHost via osqueryID heuristic;
		// we'll use a darwin host for clarity).
		darwinHost := test.NewHost(t, ds, "darwin-host", "10.1.1.1", "darwin-key", "darwin-serial", time.Now())
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(ctx, `UPDATE hosts SET platform = 'darwin' WHERE id = ?`, darwinHost.ID)
			return err
		})

		opts := fleet.ListHostReportsOptions{
			IncludeReportsDontStoreResults: true,
			ListOptions:                    fleet.ListOptions{OrderKey: "name"},
		}

		reports, _, _, err := ds.ListHostReports(ctx, darwinHost.ID, nil, "darwin", opts, fleet.DefaultMaxQueryReportRows)
		require.NoError(t, err)
		names := make([]string, 0, len(reports))
		for _, r := range reports {
			names = append(names, r.Name)
		}
		assert.Contains(t, names, qDarwin.Name, "darwin host must see darwin-platform query")
		assert.NotContains(t, names, "Linux Only Query", "darwin host must not see linux-only query")

		// Queries with no platform restriction must always be visible.
		assert.Contains(t, names, qSave1.Name, "darwin host must see platform-unrestricted query")
	})

	t.Run("label_filtering", func(t *testing.T) {
		// Create a label and a query scoped to that label.
		label, err := ds.NewLabel(ctx, &fleet.Label{Name: "label-filter-test", Query: "SELECT 1"})
		require.NoError(t, err)

		qLabeled, err := ds.NewQuery(ctx, &fleet.Query{
			Name:             "Labeled Query Eta",
			Query:            "SELECT 6",
			AuthorID:         &user.ID,
			Saved:            true,
			Logging:          fleet.LoggingSnapshot,
			LabelsIncludeAny: []fleet.LabelIdent{{LabelName: label.Name}},
		})
		require.NoError(t, err)

		opts := fleet.ListHostReportsOptions{
			IncludeReportsDontStoreResults: true,
			ListOptions:                    fleet.ListOptions{OrderKey: "name"},
		}

		// host is NOT a member of the label — labeled query must be excluded.
		reports, _, _, err := ds.ListHostReports(ctx, host.ID, nil, "", opts, fleet.DefaultMaxQueryReportRows)
		require.NoError(t, err)
		names := make([]string, 0, len(reports))
		for _, r := range reports {
			names = append(names, r.Name)
		}
		assert.NotContains(t, names, qLabeled.Name, "host without label must not see label-scoped query")

		// Add host to the label.
		err = ds.RecordLabelQueryExecutions(ctx, host, map[uint]*bool{label.ID: new(true)}, time.Now(), false)
		require.NoError(t, err)

		// host IS now a member of the label — labeled query must be included.
		reports, _, _, err = ds.ListHostReports(ctx, host.ID, nil, "", opts, fleet.DefaultMaxQueryReportRows)
		require.NoError(t, err)
		names = names[:0]
		for _, r := range reports {
			names = append(names, r.Name)
		}
		assert.Contains(t, names, qLabeled.Name, "host with matching label must see label-scoped query")

		// Queries with NO labels are always visible regardless of host membership.
		assert.Contains(t, names, qSave1.Name, "unlabeled query must always be visible")
	})

	t.Run("label_filtering_include_all", func(t *testing.T) {
		labelA, err := ds.NewLabel(ctx, &fleet.Label{Name: "include-all-A", Query: "SELECT 1"})
		require.NoError(t, err)
		labelB, err := ds.NewLabel(ctx, &fleet.Label{Name: "include-all-B", Query: "SELECT 1"})
		require.NoError(t, err)

		qIncludeAll, err := ds.NewQuery(ctx, &fleet.Query{
			Name:     "Include All Query",
			Query:    "SELECT 1",
			AuthorID: &user.ID,
			Saved:    true,
			Logging:  fleet.LoggingSnapshot,
			LabelsIncludeAll: []fleet.LabelIdent{
				{LabelName: labelA.Name},
				{LabelName: labelB.Name},
			},
		})
		require.NoError(t, err)

		newHost := func(name string) *fleet.Host {
			return test.NewHost(t, ds, name, "10.0.0.1", "k-"+name, "u-"+name, time.Now())
		}
		hostNone := newHost("include-all-host-none")
		hostOnlyA := newHost("include-all-host-onlyA")
		hostBoth := newHost("include-all-host-both")

		require.NoError(t, ds.RecordLabelQueryExecutions(ctx, hostOnlyA, map[uint]*bool{labelA.ID: new(true)}, time.Now(), false))
		require.NoError(t, ds.RecordLabelQueryExecutions(ctx, hostBoth, map[uint]*bool{labelA.ID: new(true), labelB.ID: new(true)}, time.Now(), false))

		opts := fleet.ListHostReportsOptions{
			IncludeReportsDontStoreResults: true,
			ListOptions:                    fleet.ListOptions{OrderKey: "name"},
		}

		hasReport := func(hostID uint) bool {
			t.Helper()
			reports, _, _, err := ds.ListHostReports(ctx, hostID, nil, "", opts, fleet.DefaultMaxQueryReportRows)
			require.NoError(t, err)
			for _, r := range reports {
				if r.Name == qIncludeAll.Name {
					return true
				}
			}
			return false
		}

		assert.False(t, hasReport(hostNone.ID), "host with NO required labels must not see include_all query")
		assert.False(t, hasReport(hostOnlyA.ID), "host with SUBSET of required labels must not see include_all query")
		assert.True(t, hasReport(hostBoth.ID), "host with ALL required labels must see include_all query")

		// ExcludeIncludeAllQueries hides include_all queries entirely from
		// the result set, regardless of host membership.
		excludeOpts := opts
		excludeOpts.ExcludeIncludeAllQueries = true
		hasReportExcluded := func(hostID uint) bool {
			t.Helper()
			reports, _, _, err := ds.ListHostReports(ctx, hostID, nil, "", excludeOpts, fleet.DefaultMaxQueryReportRows)
			require.NoError(t, err)
			for _, r := range reports {
				if r.Name == qIncludeAll.Name {
					return true
				}
			}
			return false
		}
		assert.False(t, hasReportExcluded(hostNone.ID), "ExcludeIncludeAllQueries must hide include_all from host with no labels")
		assert.False(t, hasReportExcluded(hostOnlyA.ID), "ExcludeIncludeAllQueries must hide include_all from host with subset of labels")
		assert.False(t, hasReportExcluded(hostBoth.ID), "ExcludeIncludeAllQueries must hide include_all from host with all labels")
	})

	t.Run("combined_platform_label_team_filters", func(t *testing.T) {
		// This test verifies that all three filters (platform, label, team) are
		// applied simultaneously and are each independently capable of excluding
		// a query.
		//
		// Setup:
		//   host: linux platform, member of labelA, no team (global)
		//   team: teamB
		//
		// Queries (all saved, IncludeReportsDontStoreResults=true):
		//   qAll         – no platform, no label, global → VISIBLE
		//   qLinux       – linux, no label, global       → VISIBLE (platform matches)
		//   qLabelA      – no platform, labelA, global   → VISIBLE (label matches)
		//   qLinuxLabelA – linux, labelA, global         → VISIBLE (both match)
		//   qDarwin      – darwin, no label, global      → EXCLUDED by platform
		//   qLabelB      – no platform, labelB, global   → EXCLUDED by label
		//   qTeamB       – no platform, no label, teamB  → EXCLUDED by team
		//   qDarwinLabelA– darwin, labelA, global        → EXCLUDED by platform (despite label)
		//   qLinuxLabelB – linux, labelB, global         → EXCLUDED by label (despite platform)

		team, err := ds.NewTeam(ctx, &fleet.Team{Name: "combined-filter-team"})
		require.NoError(t, err)

		labelA, err := ds.NewLabel(ctx, &fleet.Label{Name: "combined-labelA", Query: "SELECT 1"})
		require.NoError(t, err)
		labelB, err := ds.NewLabel(ctx, &fleet.Label{Name: "combined-labelB", Query: "SELECT 1"})
		require.NoError(t, err)

		newQ := func(name, platform string, labels []fleet.LabelIdent, teamID *uint) {
			t.Helper()
			_, err := ds.NewQuery(ctx, &fleet.Query{
				Name:             name,
				Query:            "SELECT 1",
				AuthorID:         &user.ID,
				Saved:            true,
				Logging:          fleet.LoggingSnapshot,
				Platform:         platform,
				LabelsIncludeAny: labels,
				TeamID:           teamID,
			})
			require.NoError(t, err)
		}

		newQ("combined-qAll", "", nil, nil)
		newQ("combined-qLinux", "linux", nil, nil)
		newQ("combined-qLabelA", "", []fleet.LabelIdent{{LabelName: labelA.Name}}, nil)
		newQ("combined-qLinuxLabelA", "linux", []fleet.LabelIdent{{LabelName: labelA.Name}}, nil)
		newQ("combined-qDarwin", "darwin", nil, nil)
		newQ("combined-qLabelB", "", []fleet.LabelIdent{{LabelName: labelB.Name}}, nil)
		newQ("combined-qTeamB", "", nil, &team.ID)
		newQ("combined-qDarwinLabelA", "darwin", []fleet.LabelIdent{{LabelName: labelA.Name}}, nil)
		newQ("combined-qLinuxLabelB", "linux", []fleet.LabelIdent{{LabelName: labelB.Name}}, nil)

		linuxHost := test.NewHost(t, ds, "combined-linux-host", "10.2.2.2", "combined-key", "combined-serial", time.Now())
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(ctx, `UPDATE hosts SET platform = 'ubuntu' WHERE id = ?`, linuxHost.ID)
			return err
		})
		// Add the host to labelA only.
		err = ds.RecordLabelQueryExecutions(ctx, linuxHost, map[uint]*bool{labelA.ID: new(true)}, time.Now(), false)
		require.NoError(t, err)

		opts := fleet.ListHostReportsOptions{
			IncludeReportsDontStoreResults: true,
			ListOptions:                    fleet.ListOptions{OrderKey: "name"},
		}
		// host has no team → pass nil teamID; PlatformFromHost("ubuntu") = "linux"
		reports, _, _, err := ds.ListHostReports(ctx, linuxHost.ID, nil, "linux", opts, fleet.DefaultMaxQueryReportRows)
		require.NoError(t, err)

		names := make(map[string]bool, len(reports))
		for _, r := range reports {
			names[r.Name] = true
		}

		assert.True(t, names["combined-qAll"], "unrestricted query must be visible")
		assert.True(t, names["combined-qLinux"], "matching-platform query must be visible")
		assert.True(t, names["combined-qLabelA"], "matching-label query must be visible")
		assert.True(t, names["combined-qLinuxLabelA"], "matching platform+label query must be visible")

		assert.False(t, names["combined-qDarwin"], "wrong-platform query must be excluded")
		assert.False(t, names["combined-qLabelB"], "non-member label query must be excluded")
		assert.False(t, names["combined-qTeamB"], "other-team query must be excluded")
		assert.False(t, names["combined-qDarwinLabelA"], "wrong platform must exclude even if label matches")
		assert.False(t, names["combined-qLinuxLabelB"], "non-member label must exclude even if platform matches")
	})
}
