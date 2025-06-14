package mysql

import (
	"context"
	"fmt"
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
		{"CleanupWorkerJobs", testCleanupWorkerJobs},
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

func testCleanupWorkerJobs(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	// no job yet
	n, err := ds.CleanupWorkerJobs(ctx, time.Second, time.Second)
	require.NoError(t, err)
	require.EqualValues(t, 0, n)

	setJobTimestamp := func(j *fleet.Job, subtract time.Duration) {
		j.NotBefore = time.Now().UTC().Add(-subtract)
		_, err = ds.UpdateJob(ctx, j.ID, j)
		require.NoError(t, err)
	}
	setJobStatus := func(j *fleet.Job, state fleet.JobState) {
		j.State = state
		_, err = ds.UpdateJob(ctx, j.ID, j)
		require.NoError(t, err)
	}

	// enqueue a job
	j1 := &fleet.Job{Name: "j1", State: fleet.JobStateQueued}
	j1, err = ds.NewJob(ctx, j1)
	require.NoError(t, err)

	setJobTimestamp(j1, 2*time.Second)

	// still clears nothing as it is not in a final state
	n, err = ds.CleanupWorkerJobs(ctx, time.Second, time.Second)
	require.NoError(t, err)
	require.EqualValues(t, 0, n)

	// job is still returned as a queued job
	jobs, err := ds.GetQueuedJobs(ctx, 10, time.Time{})
	require.NoError(t, err)
	require.Len(t, jobs, 1)
	require.Equal(t, j1.ID, jobs[0].ID)

	// mark it as done
	setJobStatus(j1, fleet.JobStateSuccess)

	// does not clear it if the completed duration is not far enough in the past
	n, err = ds.CleanupWorkerJobs(ctx, time.Second, time.Minute)
	require.NoError(t, err)
	require.EqualValues(t, 0, n)

	// does clear it if the completed duration is far enough in the past
	n, err = ds.CleanupWorkerJobs(ctx, time.Second, time.Second)
	require.NoError(t, err)
	require.EqualValues(t, 1, n)

	jobs, err = ds.GetQueuedJobs(ctx, 10, time.Time{})
	require.NoError(t, err)
	require.Len(t, jobs, 0)

	// enqueue a few more jobs
	queuedJobs := make([]*fleet.Job, 0, 4)
	for i := 0; i < 4; i++ {
		j := &fleet.Job{Name: "j" + fmt.Sprint(i+1), State: fleet.JobStateQueued}
		j, err = ds.NewJob(ctx, j)
		require.NoError(t, err)
		setJobTimestamp(j, time.Duration(i+1)*time.Minute)
		queuedJobs = append(queuedJobs, j)
	}

	// make jobs[1] and jobs[3] failed, jobs[2] successful, jobs[0] queued
	setJobStatus(queuedJobs[1], fleet.JobStateFailure) // failed 2m ago
	setJobStatus(queuedJobs[3], fleet.JobStateFailure) // failed 4m ago
	setJobStatus(queuedJobs[2], fleet.JobStateSuccess) // successful 3m ago

	// cleanup failed > 3m, success > 1m, should delete jobs[3] and jobs[2]
	n, err = ds.CleanupWorkerJobs(ctx, 3*time.Minute, time.Minute)
	require.NoError(t, err)
	require.EqualValues(t, 2, n)

	// cleanup failed > 1m, success > 10m, should delete jobs[1]
	n, err = ds.CleanupWorkerJobs(ctx, 1*time.Minute, 10*time.Minute)
	require.NoError(t, err)
	require.EqualValues(t, 1, n)

	// jobs[0] is still queued
	jobs, err = ds.GetQueuedJobs(ctx, 10, time.Time{})
	require.NoError(t, err)
	require.Len(t, jobs, 1)
	require.Equal(t, queuedJobs[0].ID, jobs[0].ID)
}
