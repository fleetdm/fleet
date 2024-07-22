package worker

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	kitlog "github.com/go-kit/log"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
	"github.com/tj/assert"
)

type testJob struct {
	name string
	run  func(ctx context.Context, argsJSON json.RawMessage) error
}

func (t testJob) Name() string {
	return t.name
}

func (t testJob) Run(ctx context.Context, argsJSON json.RawMessage) error {
	return t.run(ctx, argsJSON)
}

func TestWorker(t *testing.T) {
	ds := new(mock.Store)

	// set up mocks
	getQueuedJobsCalled := 0
	ds.GetQueuedJobsFunc = func(ctx context.Context, maxNumJobs int, now time.Time) ([]*fleet.Job, error) {
		if getQueuedJobsCalled > 0 {
			return nil, nil
		}
		getQueuedJobsCalled++

		argsJSON := json.RawMessage(`{"arg1":"foo"}`)
		return []*fleet.Job{
			{
				ID:   1,
				Name: "test",
				Args: &argsJSON,
			},
		}, nil
	}
	ds.UpdateJobFunc = func(ctx context.Context, id uint, job *fleet.Job) (*fleet.Job, error) {
		assert.Equal(t, fleet.JobStateSuccess, job.State)
		return job, nil
	}

	logger := kitlog.NewNopLogger()
	w := NewWorker(ds, logger)

	// register a test job
	jobCalled := false
	j := testJob{
		name: "test",
		run: func(ctx context.Context, argsJSON json.RawMessage) error {
			jobCalled = true

			assert.Equal(t, json.RawMessage(`{"arg1":"foo"}`), argsJSON)
			return nil
		},
	}

	w.Register(j)

	err := w.ProcessJobs(context.Background())
	require.NoError(t, err)

	require.True(t, ds.GetQueuedJobsFuncInvoked)
	require.True(t, ds.UpdateJobFuncInvoked)

	require.True(t, jobCalled)
}

func TestWorkerRetries(t *testing.T) {
	ds := new(mock.Store)

	// set up mocks
	argsJSON := json.RawMessage(`{"arg1":"foo"}`)
	theJob := &fleet.Job{
		ID:      1,
		Name:    "test",
		Args:    &argsJSON,
		State:   fleet.JobStateQueued,
		Retries: 0,
	}
	ds.GetQueuedJobsFunc = func(ctx context.Context, maxNumJobs int, now time.Time) ([]*fleet.Job, error) {
		if theJob.State == fleet.JobStateQueued {
			return []*fleet.Job{theJob}, nil
		}
		return nil, nil
	}

	jobFailed := false
	ds.UpdateJobFunc = func(ctx context.Context, id uint, job *fleet.Job) (*fleet.Job, error) {
		assert.Equal(t, "unknown error", job.Error)
		if job.State == fleet.JobStateFailure {
			jobFailed = true
			assert.Equal(t, maxRetries, job.Retries)
		}

		return job, nil
	}

	logger := kitlog.NewNopLogger()
	w := NewWorker(ds, logger)

	// register a test job
	jobCalled := 0
	j := testJob{
		name: "test",
		run: func(ctx context.Context, argsJSON json.RawMessage) error {
			jobCalled++
			return errors.New("unknown error")
		},
	}
	w.Register(j)

	// the worker stops a ProcessJobs batch once it receives the same job again,
	// so run it multiple times to test its retries.
	for i := 0; i < maxRetries+1; i++ {
		err := w.ProcessJobs(context.Background())
		require.NoError(t, err)

		require.True(t, ds.GetQueuedJobsFuncInvoked)
		require.True(t, ds.UpdateJobFuncInvoked)
		ds.GetQueuedJobsFuncInvoked = false
		ds.UpdateJobFuncInvoked = false

		require.Equal(t, i+1, jobCalled)
		require.Equal(t, i == maxRetries, jobFailed) // true on last iteration, false otherwise
	}

	// processing again does nothing as the job is not queued anymore, it is failed
	err := w.ProcessJobs(context.Background())
	require.NoError(t, err)
	require.Equal(t, maxRetries+1, jobCalled)
}

func TestWorkerMiddleJobFails(t *testing.T) {
	ds := new(mock.Store)

	// set up mocks
	jobs := []*fleet.Job{
		{
			ID:      1,
			Name:    "test",
			State:   fleet.JobStateQueued,
			Args:    ptr.RawMessage(json.RawMessage(`1`)),
			Retries: 0,
		},
		{
			ID:      2,
			Name:    "test",
			State:   fleet.JobStateQueued,
			Args:    ptr.RawMessage(json.RawMessage(`2`)),
			Retries: 0,
		},
		{
			ID:      3,
			Name:    "test",
			State:   fleet.JobStateQueued,
			Args:    ptr.RawMessage(json.RawMessage(`3`)),
			Retries: 0,
		},
	}
	ds.GetQueuedJobsFunc = func(ctx context.Context, maxNumJobs int, now time.Time) ([]*fleet.Job, error) {
		var queued []*fleet.Job
		for _, j := range jobs {
			if j.State == fleet.JobStateQueued {
				queued = append(queued, j)
			}
		}
		return queued, nil
	}

	ds.UpdateJobFunc = func(ctx context.Context, id uint, job *fleet.Job) (*fleet.Job, error) {
		return job, nil
	}

	logger := kitlog.NewNopLogger()
	w := NewWorker(ds, logger)

	// register a test job
	var jobCallCount int
	j := testJob{
		name: "test",
		run: func(ctx context.Context, argsJSON json.RawMessage) error {
			jobCallCount++

			var id int
			if err := json.Unmarshal(argsJSON, &id); err != nil {
				return err
			}

			if id == 2 {
				// fail the job with id 2
				return errors.New("unknown error")
			}
			return nil
		},
	}
	w.Register(j)

	// process the jobs, jobs 1 and 3 should be successful, job 2 still queued
	err := w.ProcessJobs(context.Background())
	require.NoError(t, err)

	require.True(t, ds.GetQueuedJobsFuncInvoked)
	require.True(t, ds.UpdateJobFuncInvoked)
	ds.GetQueuedJobsFuncInvoked = false
	ds.UpdateJobFuncInvoked = false

	require.Equal(t, fleet.JobStateSuccess, jobs[0].State)
	require.Equal(t, fleet.JobStateQueued, jobs[1].State)
	require.Equal(t, 1, jobs[1].Retries)
	require.Equal(t, fleet.JobStateSuccess, jobs[2].State)
	require.Equal(t, 3, jobCallCount)

	// processing again only processes job 2 (still queued)
	err = w.ProcessJobs(context.Background())
	require.NoError(t, err)

	require.True(t, ds.GetQueuedJobsFuncInvoked)
	require.True(t, ds.UpdateJobFuncInvoked)

	require.Equal(t, fleet.JobStateQueued, jobs[1].State)
	require.Equal(t, 2, jobs[1].Retries)
	require.Equal(t, 4, jobCallCount)
}

func TestWorkerWithRealDatastore(t *testing.T) {
	ctx := context.Background()
	ds := mysql.CreateMySQLDS(t)
	// call TruncateTables immediately, because a DB migration may create jobs
	mysql.TruncateTables(t, ds)

	oldDelayPerRetry := delayPerRetry
	delayPerRetry = []time.Duration{
		1: 0,
		2: 0,
		3: time.Hour,
	} // retry twice on the next cron, then not before an hour
	t.Cleanup(func() { delayPerRetry = oldDelayPerRetry })

	logger := kitlog.NewNopLogger()
	w := NewWorker(ds, logger)

	// register a test job
	var jobCallCount int
	j := testJob{
		name: "test",
		run: func(ctx context.Context, argsJSON json.RawMessage) error {
			jobCallCount++

			var name string
			if err := json.Unmarshal(argsJSON, &name); err != nil {
				return err
			}

			if name == "fail" {
				// always fail that job
				return errors.New("unknown error")
			}
			return nil
		},
	}
	w.Register(j)

	// add some jobs
	j1, err := QueueJob(ctx, ds, "test", "success")
	require.NoError(t, err)
	require.NotZero(t, j1.ID)
	j2, err := QueueJob(ctx, ds, "test", "fail")
	require.NoError(t, err)
	require.NotZero(t, j2.ID)
	j3, err := QueueJob(ctx, ds, "test", "success2")
	require.NoError(t, err)
	require.NotZero(t, j3.ID)

	// process the jobs a first time, jobs 1 and 3 should be successful, job 2 still queued
	err = w.ProcessJobs(ctx)
	require.NoError(t, err)
	require.Equal(t, 3, jobCallCount)
	jobCallCount = 0

	// add a delay because there may be a slight discrepency when truncating the
	// timestamp in mysql vs the one set in ProcessJobs (time.Now().Add(...)).
	time.Sleep(time.Second)

	jobs, err := ds.GetQueuedJobs(ctx, 10, time.Time{})
	require.NoError(t, err)
	require.Len(t, jobs, 1)
	require.Equal(t, j2.ID, jobs[0].ID)
	require.Equal(t, 1, jobs[0].Retries)

	// process the jobs a second time, job 2 still queued
	err = w.ProcessJobs(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, jobCallCount)
	jobCallCount = 0

	// add a delay because there may be a slight discrepency when truncating the
	// timestamp in mysql vs the one set in ProcessJobs (time.Now().Add(...)).
	time.Sleep(time.Second)

	jobs, err = ds.GetQueuedJobs(ctx, 10, time.Time{})
	require.NoError(t, err)
	require.Len(t, jobs, 1)
	require.Equal(t, j2.ID, jobs[0].ID)
	require.Equal(t, 2, jobs[0].Retries)

	// process the jobs a third time, job 2 still queued but now with a long delay
	beforeThirdTime := time.Now()
	err = w.ProcessJobs(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, jobCallCount)
	jobCallCount = 0

	time.Sleep(time.Second)

	jobs, err = ds.GetQueuedJobs(ctx, 10, time.Time{})
	require.NoError(t, err)
	require.Empty(t, jobs)

	var failedJob fleet.Job
	mysql.ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &failedJob, "SELECT * FROM jobs WHERE id = ?", j2.ID)
	})
	require.Equal(t, 3, failedJob.Retries)
	require.WithinDuration(t, beforeThirdTime.Add(time.Hour), failedJob.NotBefore, time.Minute)
}
