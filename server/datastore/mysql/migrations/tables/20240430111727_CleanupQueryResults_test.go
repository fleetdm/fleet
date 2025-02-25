package tables

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20240430111727(t *testing.T) {
	db := applyUpToPrev(t)

	hostID := 1
	newTeam := func(name string) uint {
		return uint(execNoErrLastID(t, db, //nolint:gosec // dismiss G115
			`INSERT INTO teams (name) VALUES (?);`,
			name,
		))
	}
	newHost := func(teamID *uint) uint {
		id := fmt.Sprintf("%d", hostID)
		hostID++
		return uint(execNoErrLastID(t, db, //nolint:gosec // dismiss G115
			`INSERT INTO hosts (osquery_host_id, node_key, team_id) VALUES (?, ?, ?);`,
			id, id, teamID,
		))
	}
	newQuery := func(name string, teamID *uint) uint {
		return uint(execNoErrLastID(t, db, //nolint:gosec // dismiss G115
			`INSERT INTO queries (name, description, logging_type, team_id, query, saved) VALUES (?, '', 'snapshot', ?, 'SELECT 1;', 1);`,
			name, teamID,
		))
	}
	newQueryResults := func(queryID, hostID uint, resultCount int) {
		var args []interface{}
		for i := 0; i < resultCount; i++ {
			args = append(args, queryID, hostID, fmt.Sprintf(`{"foo": "bar%d"}`, i))
		}
		values := strings.TrimSuffix(strings.Repeat("(?, ?, ?, NOW()),", resultCount), ",")
		_, err := db.Exec(fmt.Sprintf(`INSERT INTO query_results (query_id, host_id, data, last_fetched) VALUES %s;`, values),
			args...,
		)
		require.NoError(t, err)
	}

	team1ID := newTeam("team1")
	team2ID := newTeam("team2")
	host1GlobalID := newHost(nil)
	host2Team1ID := newHost(&team1ID)
	host3Team2ID := newHost(&team2ID)
	query1GlobalID := newQuery("query1Global", nil)
	query2Team1ID := newQuery("query2Team1", &team1ID)
	query3Team2ID := newQuery("query3Team2", &team2ID)

	newQueryResults(query1GlobalID, host1GlobalID, 1)
	newQueryResults(query1GlobalID, host2Team1ID, 2)
	newQueryResults(query1GlobalID, host3Team2ID, 3)

	newQueryResults(query2Team1ID, host1GlobalID, 4)
	newQueryResults(query2Team1ID, host2Team1ID, 5)
	newQueryResults(query2Team1ID, host3Team2ID, 6)

	newQueryResults(query3Team2ID, host1GlobalID, 7)
	newQueryResults(query3Team2ID, host2Team1ID, 8)
	newQueryResults(query3Team2ID, host3Team2ID, 9)

	// Apply current migration.
	applyNext(t, db)

	getQueryResultsCount := func(queryID, hostID uint) int {
		var count int
		err := db.Get(&count, `SELECT COUNT(*) FROM query_results WHERE query_id = ? AND host_id = ?`, queryID, hostID)
		require.NoError(t, err)
		return count
	}

	count := getQueryResultsCount(query1GlobalID, host1GlobalID)
	require.Equal(t, 1, count) // result for global queries are not deleted.
	count = getQueryResultsCount(query1GlobalID, host2Team1ID)
	require.Equal(t, 2, count) // result for global queries are not deleted.
	count = getQueryResultsCount(query1GlobalID, host3Team2ID)
	require.Equal(t, 3, count) // result for global queries are not deleted.

	count = getQueryResultsCount(query2Team1ID, host1GlobalID)
	require.Equal(t, 0, count) // query results of a team query different than the host's team are deleted.
	count = getQueryResultsCount(query2Team1ID, host2Team1ID)
	require.Equal(t, 5, count) // team query results of the host's team are not deleted.
	count = getQueryResultsCount(query2Team1ID, host3Team2ID)
	require.Equal(t, 0, count) // query results of a team query different than the host's team are deleted.

	count = getQueryResultsCount(query3Team2ID, host1GlobalID)
	require.Equal(t, 0, count) // query results of a team query different than the host's team are deleted.
	count = getQueryResultsCount(query3Team2ID, host2Team1ID)
	require.Equal(t, 0, count) // query results of a team query different than the host's team are deleted.
	count = getQueryResultsCount(query3Team2ID, host3Team2ID)
	require.Equal(t, 9, count) // team query results of the host's team are not deleted.
}
