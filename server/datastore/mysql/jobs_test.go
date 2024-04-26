package mysql

import (
	"context"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

func TestJobs(t *testing.T) {
	ds := CreateMySQLDS(t)
	// call TruncateTables before the first test, because a DB migation may have
	// created job entries.
	TruncateTables(t, ds)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"QueueAndProcessJobs", testQueueAndProcessJobs},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)
			c.fn(t, ds)
		})
	}
}

func testQueueAndProcessJobs(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	// no jobs yet
	jobs, err := ds.GetQueuedJobs(ctx, 10, time.Time{})
	require.NoError(t, err)
	require.Empty(t, jobs)

	// add a couple of jobs, one immediate, one delayed
	j1 := &fleet.Job{Name: "j1", State: fleet.JobStateQueued}
	j2 := &fleet.Job{Name: "j2", State: fleet.JobStateQueued, NotBefore: time.Now().Add(time.Hour)}
	j1, err = ds.NewJob(ctx, j1)
	require.NoError(t, err)
	require.NotZero(t, j1.ID)
	j2, err = ds.NewJob(ctx, j2)
	require.NoError(t, err)
	require.NotZero(t, j2.ID)

	// only j1 is returned
	jobs, err = ds.GetQueuedJobs(ctx, 10, time.Time{})
	require.NoError(t, err)
	require.Len(t, jobs, 1)
	require.Equal(t, j1.ID, jobs[0].ID)
	require.NotZero(t, jobs[0].NotBefore)
	require.False(t, jobs[0].NotBefore.After(time.Now())) // before or equal

	// update j1 as successful
	j1.State = fleet.JobStateSuccess
	_, err = ds.UpdateJob(ctx, j1.ID, j1)
	require.NoError(t, err)

	// no jobs queued for now
	jobs, err = ds.GetQueuedJobs(ctx, 10, time.Time{})
	require.NoError(t, err)
	require.Empty(t, jobs)

	// update j2 not before timestamp to now (-1s just to be safe)
	j2.NotBefore = time.Now().Add(-time.Second)
	_, err = ds.UpdateJob(ctx, j2.ID, j2)
	require.NoError(t, err)

	// j2 is returned
	jobs, err = ds.GetQueuedJobs(ctx, 10, time.Time{})
	require.NoError(t, err)
	require.Len(t, jobs, 1)
	require.Equal(t, j2.ID, jobs[0].ID)
	require.NotZero(t, jobs[0].NotBefore)
	require.False(t, jobs[0].NotBefore.After(time.Now())) // before or equal
}
