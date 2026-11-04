package mysql

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestScheduledQueries(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"AsyncBatchSaveHostsScheduledQueryStats", testScheduledQueriesAsyncBatchSaveStats},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)
			c.fn(t, ds)
		})
	}
}

func testScheduledQueriesAsyncBatchSaveStats(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	lastExec := time.Now() // don't care about that value in the test, it just needs to be set

	user := test.NewUser(t, ds, "user", "user@example.com", true)

	h1 := test.NewHost(t, ds, "foo1.local", "192.168.1.1", "1", "1", time.Now())
	h2 := test.NewHost(t, ds, "foo2.local", "192.168.1.2", "2", "2", time.Now())
	h3 := test.NewHost(t, ds, "foo3.local", "192.168.1.3", "3", "3", time.Now())

	team, err := ds.NewTeam(ctx, &fleet.Team{Name: "team1"})
	require.NoError(t, err)
	teamPackName := fmt.Sprintf("team-%d", team.ID)

	q1 := test.NewQuery(t, ds, nil, "q1", "select 1", user.ID, true)
	q2 := test.NewQuery(t, ds, nil, "q2", "select 2", user.ID, true)
	q3 := test.NewQuery(t, ds, nil, "q3", "select 3", user.ID, true)
	q4 := test.NewQuery(t, ds, &team.ID, "q4", "select 4", user.ID, true)

	assertStats := func(m map[uint][]fleet.ScheduledQueryStats) {
		// checks that the stats are as expected (only the Executions field is
		// checked for the provided host ID/scheduled query ID).
		for hid, stats := range m {
			for _, st := range stats {
				ExecAdhocSQL(t, ds, func(tx sqlx.ExtContext) error {
					var got uint64
					err := sqlx.GetContext(ctx, tx, &got, `SELECT executions FROM scheduled_query_stats WHERE host_id = ? AND scheduled_query_id = ?`, hid, st.ScheduledQueryID)
					if err != nil {
						return err
					}
					require.Equal(t, st.Executions, got)
					return nil
				})
			}
		}
	}

	const batchSize = 2
	// save without any stats
	execs, err := ds.AsyncBatchSaveHostsScheduledQueryStats(ctx, nil, batchSize)
	require.NoError(t, err)
	require.Equal(t, 0, execs)

	// single host, single stat
	m := map[uint][]fleet.ScheduledQueryStats{
		h1.ID: {
			{
				ScheduledQueryID:   q1.ID,
				Executions:         1,
				LastExecuted:       lastExec,
				PackName:           "Global",
				ScheduledQueryName: q1.Name,
			},
		},
	}
	execs, err = ds.AsyncBatchSaveHostsScheduledQueryStats(ctx, m, batchSize)
	require.NoError(t, err)
	require.Equal(t, 1, execs)
	assertStats(m)

	// single host, stats == batch size
	m = map[uint][]fleet.ScheduledQueryStats{
		h1.ID: {
			{
				ScheduledQueryID:   q1.ID,
				Executions:         2,
				LastExecuted:       lastExec,
				PackName:           "Global",
				ScheduledQueryName: q1.Name,
			},
			{
				ScheduledQueryID:   q2.ID,
				Executions:         3,
				LastExecuted:       lastExec,
				PackName:           "Global",
				ScheduledQueryName: q2.Name,
			},
		},
	}
	execs, err = ds.AsyncBatchSaveHostsScheduledQueryStats(ctx, m, batchSize)
	require.NoError(t, err)
	require.Equal(t, 1, execs)
	assertStats(m)

	// single host, stats > batch size
	m = map[uint][]fleet.ScheduledQueryStats{
		h1.ID: {
			{
				ScheduledQueryID:   q1.ID,
				Executions:         4,
				LastExecuted:       lastExec,
				PackName:           "Global",
				ScheduledQueryName: q1.Name,
			},
			{
				ScheduledQueryID:   q2.ID,
				Executions:         5,
				LastExecuted:       lastExec,
				PackName:           "Global",
				ScheduledQueryName: q2.Name,
			},
			{
				ScheduledQueryID:   q3.ID,
				Executions:         6,
				LastExecuted:       lastExec,
				PackName:           "Global",
				ScheduledQueryName: q3.Name,
			},
		},
	}
	execs, err = ds.AsyncBatchSaveHostsScheduledQueryStats(ctx, m, batchSize)
	require.NoError(t, err)
	require.Equal(t, 2, execs)
	assertStats(m)

	// multi host, stats == batch size
	m = map[uint][]fleet.ScheduledQueryStats{
		h1.ID: {
			{
				ScheduledQueryID:   q1.ID,
				Executions:         7,
				LastExecuted:       lastExec,
				PackName:           "Global",
				ScheduledQueryName: q1.Name,
			},
		},
		h2.ID: {
			{
				ScheduledQueryID:   q2.ID,
				Executions:         8,
				LastExecuted:       lastExec,
				PackName:           "Global",
				ScheduledQueryName: q2.Name,
			},
		},
	}
	execs, err = ds.AsyncBatchSaveHostsScheduledQueryStats(ctx, m, batchSize)
	require.NoError(t, err)
	require.Equal(t, 1, execs)
	assertStats(m)

	// multi host, stats > batch size
	m = map[uint][]fleet.ScheduledQueryStats{
		h1.ID: {
			{
				ScheduledQueryID:   q1.ID,
				Executions:         9,
				LastExecuted:       lastExec,
				PackName:           "Global",
				ScheduledQueryName: q1.Name,
			},
		},
		h2.ID: {
			{
				ScheduledQueryID:   q2.ID,
				Executions:         10,
				LastExecuted:       lastExec,
				PackName:           "Global",
				ScheduledQueryName: q2.Name,
			},
			{
				ScheduledQueryID:   q3.ID,
				Executions:         11,
				LastExecuted:       lastExec,
				PackName:           "Global",
				ScheduledQueryName: q3.Name,
			},
		},
	}
	execs, err = ds.AsyncBatchSaveHostsScheduledQueryStats(ctx, m, batchSize)
	require.NoError(t, err)
	require.Equal(t, 2, execs)
	assertStats(m)

	// multi host, stats > (N * batch size)
	m = map[uint][]fleet.ScheduledQueryStats{
		h1.ID: {
			{
				ScheduledQueryID:   q1.ID,
				Executions:         12,
				LastExecuted:       lastExec,
				PackName:           "Global",
				ScheduledQueryName: q1.Name,
			},
			{
				ScheduledQueryID:   q2.ID,
				Executions:         13,
				LastExecuted:       lastExec,
				PackName:           "Global",
				ScheduledQueryName: q2.Name,
			},
		},
		h2.ID: {
			{
				ScheduledQueryID:   q2.ID,
				Executions:         14,
				LastExecuted:       lastExec,
				PackName:           "Global",
				ScheduledQueryName: q2.Name,
			},
			{
				ScheduledQueryID:   q4.ID,
				Executions:         15,
				LastExecuted:       lastExec,
				PackName:           teamPackName,
				ScheduledQueryName: q4.Name,
			},
		},
		h3.ID: {
			{
				ScheduledQueryID:   q1.ID,
				Executions:         16,
				LastExecuted:       lastExec,
				PackName:           "Global",
				ScheduledQueryName: q1.Name,
			},
			{
				ScheduledQueryID:   q2.ID,
				Executions:         17,
				LastExecuted:       lastExec,
				PackName:           "Global",
				ScheduledQueryName: q2.Name,
			},
			{
				ScheduledQueryID:   q3.ID,
				Executions:         18,
				LastExecuted:       lastExec,
				PackName:           "Global",
				ScheduledQueryName: q3.Name,
			},
		},
	}
	execs, err = ds.AsyncBatchSaveHostsScheduledQueryStats(ctx, m, batchSize)
	require.NoError(t, err)
	require.Equal(t, 4, execs)
	assertStats(m)

	// stats with no resolved report ID are skipped, don't count toward the
	// batch size, and don't write a row with a zero ID
	h1Stats := []fleet.ScheduledQueryStats{
		{
			ScheduledQueryID:   q1.ID,
			Executions:         19,
			LastExecuted:       lastExec,
			PackName:           "Global",
			ScheduledQueryName: q1.Name,
		},
		{
			Executions:         20,
			LastExecuted:       lastExec,
			PackName:           "Global",
			ScheduledQueryName: "deleted-report",
		},
		{
			ScheduledQueryID:   q2.ID,
			Executions:         21,
			LastExecuted:       lastExec,
			PackName:           "Global",
			ScheduledQueryName: q2.Name,
		},
	}
	execs, err = ds.AsyncBatchSaveHostsScheduledQueryStats(ctx, map[uint][]fleet.ScheduledQueryStats{h1.ID: h1Stats}, batchSize)
	require.NoError(t, err)
	require.Equal(t, 1, execs)
	assertStats(map[uint][]fleet.ScheduledQueryStats{h1.ID: {h1Stats[0], h1Stats[2]}})
	ExecAdhocSQL(t, ds, func(tx sqlx.ExtContext) error {
		var count int
		err := sqlx.GetContext(ctx, tx, &count, `SELECT COUNT(*) FROM scheduled_query_stats WHERE scheduled_query_id = 0`)
		if err != nil {
			return err
		}
		require.Zero(t, count)
		return nil
	})
}
