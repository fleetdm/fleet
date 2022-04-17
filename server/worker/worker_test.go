package worker

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	kitlog "github.com/go-kit/kit/log"
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
	ds.GetQueuedJobsFunc = func(ctx context.Context, maxNumJobs int) ([]*fleet.Job, error) {
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
	ds.GetQueuedJobsFunc = func(ctx context.Context, maxNumJobs int) ([]*fleet.Job, error) {
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
	ds.GetQueuedJobsFunc = func(ctx context.Context, maxNumJobs int) ([]*fleet.Job, error) {
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
