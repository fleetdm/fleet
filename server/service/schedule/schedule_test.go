package schedule

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
)

type nopLocker struct{}

func (nopLocker) Lock(context.Context, string, string, time.Duration) (bool, error) {
	return true, nil
}

type testJobber struct {
	name        string
	jobRan      bool
	jobRunCount int
	shouldFail  bool
	shouldPanic bool
	log         func()
}

func (j *testJobber) run(ctx context.Context) error {
	j.jobRan = true
	j.jobRunCount += 1

	if j.log != nil {
		j.log()
	}
	if j.shouldFail {
		j.fail(ctx)
	}
	if j.shouldPanic {
		panic("panicked!!")
	}

	return nil
}

func (j *testJobber) fail(ctx context.Context) error {
	return errors.New(j.name)
}

func TestNewSchedule(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	j := &testJobber{jobRan: false}
	s := New(ctx, "test_new_schedule", "test_instance", 10*time.Millisecond, nopLocker{},
		WithJob("test_job", j),
	)
	s.Start()

	time.Sleep(1 * time.Second)
	cancel()

	select {
	case <-s.Done():
		require.True(t, j.jobRan)
	case <-time.After(5 * time.Second):
		t.Error("timeout")
	}
}

type counterLocker struct {
	mu    sync.Mutex
	count int
}

func (l *counterLocker) Lock(context.Context, string, string, time.Duration) (bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.count = l.count + 1
	return true, nil
}

func TestScheduleLocker(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	locker := counterLocker{}
	j := &testJobber{}
	s := New(ctx, "test_schedule_locker", "test_instance", 10*time.Millisecond, &locker,
		WithJob("test_job", j),
	)
	s.Start()

	time.Sleep(1 * time.Second)
	cancel()

	select {
	case <-s.Done():
		require.Equal(t, locker.count, j.jobRunCount)
	case <-time.After(5 * time.Second):
		t.Error("timeout")
	}
}

func TestMultipleSchedules(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	var ss []*Schedule

	var m sync.Mutex
	jobRun := make(map[string]struct{})
	setJobRun := func(id string) {
		m.Lock()
		defer m.Unlock()

		jobRun[id] = struct{}{}
	}
	var jobNames []string

	for _, tc := range []struct {
		name       string
		instanceID string
		interval   time.Duration
		jobs       []Job
	}{
		{
			name:       "test_schedule_1",
			instanceID: "test_instance",
			interval:   10 * time.Millisecond,
			jobs: []Job{
				{
					id:     "test_job_1",
					jobber: &testJobber{log: func() { setJobRun("test_job_1") }},
				},
			},
		},
		{
			name:       "test_schedule_2",
			instanceID: "test_instance",
			interval:   100 * time.Millisecond,
			jobs: []Job{
				{
					id:     "test_job_2",
					jobber: &testJobber{log: func() { setJobRun("test_job_2") }},
				},
			},
		},
		{
			name:       "test_schedule_3",
			instanceID: "test_instance",
			interval:   100 * time.Millisecond,
			jobs: []Job{
				{
					id: "test_job_3",
					// job 3 fails, job 4 should still run.
					jobber: &testJobber{name: "test_job_3", shouldFail: true, log: func() { setJobRun("test_job_3") }},
				},
				{
					id:     "test_job_4",
					jobber: &testJobber{name: "test_job_4", log: func() { setJobRun("test_job_4") }},
				},
			},
		},
	} {
		var opts []Option
		for _, job := range tc.jobs {
			opts = append(opts, WithJob(job.id, job.jobber))
			jobNames = append(jobNames, job.id)
		}
		s := New(ctx, tc.name, tc.instanceID, tc.interval, nopLocker{}, opts...)
		s.Start()
		ss = append(ss, s)
	}

	time.Sleep(1 * time.Second)
	cancel()

	for i, s := range ss {
		select {
		case <-s.Done():
			// OK
		case <-time.After(1 * time.Second):
			t.Errorf("timeout: %d", i)
		}
	}
	for _, s := range jobNames {
		_, ok := jobRun[s]
		require.True(t, ok, "job: %s", s)
	}
}

func TestMultipleJobsInOrder(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	jobs := make(chan int)

	s := New(ctx, "test_schedule", "test_instance", 100*time.Millisecond, nopLocker{},
		WithJob("test_job_1", &testJobber{log: func() {
			jobs <- 1
		}}),
		WithJob("test_job_2", &testJobber{shouldFail: true, log: func() {
			jobs <- 2
		}}),
		WithJob("test_job_3", &testJobber{log: func() {
			jobs <- 3
		}}),
	)
	s.Start()

	var g errgroup.Group
	g.Go(func() error {
		i := 1
		for {
			select {
			case job, ok := <-jobs:
				if !ok {
					return nil
				}
				if job != i {
					return fmt.Errorf("mismatch id: %d vs %d", job, i)
				}
				i++
				if i == 4 {
					i = 1
				}
			case <-time.After(5 * time.Second):
				return fmt.Errorf("timeout: %d", i)
			}
		}
	})

	time.Sleep(1 * time.Second)
	cancel()
	select {
	case <-s.Done():
		close(jobs)
	case <-time.After(5 * time.Second):
		t.Error("timeout")
	}

	err := g.Wait()
	require.NoError(t, err)
}

func TestConfigReloadCheck(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	job := &testJobber{}

	s := New(ctx, "test_schedule", "test_instance", 200*time.Millisecond, nopLocker{},
		WithConfigReloadInterval(100*time.Millisecond, func(_ context.Context) (time.Duration, error) {
			return 50 * time.Millisecond, nil
		}),
		WithJob("test_job", job),
	)

	require.Equal(t, s.getSchedInterval(), 200*time.Millisecond)
	require.Equal(t, s.configReloadInterval, 100*time.Millisecond)

	s.Start()

	time.Sleep(1 * time.Second)
	cancel()

	select {
	case <-s.Done():
		require.Equal(t, s.getSchedInterval(), 50*time.Millisecond)
		require.Equal(t, s.configReloadInterval, 100*time.Millisecond)
		require.True(t, job.jobRan)
	case <-time.After(5 * time.Second):
		t.Error("timeout")
	}
}

func TestJobPanicRecover(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	j1 := &testJobber{name: "test_job_1", jobRan: false, shouldPanic: true}
	j2 := &testJobber{name: "test_job_2", jobRan: false}

	s := New(ctx, "test_new_schedule", "test_instance", 10*time.Millisecond, nopLocker{},
		WithJob(j1.name, j1),
		WithJob(j2.name, j2),
	)
	s.Start()

	time.Sleep(1 * time.Second)
	cancel()

	select {
	case <-s.Done():
		require.True(t, j1.jobRan)
		// j2 should still run even though j1 panicked
		require.True(t, j2.jobRan)
	case <-time.After(5 * time.Second):
		t.Error("timeout")
	}
}
