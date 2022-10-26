package schedule

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
)

type nopLocker struct{}

func (nopLocker) Lock(context.Context, string, string, time.Duration) (bool, error) {
	return true, nil
}

func (nopLocker) Unlock(context.Context, string, string) error {
	return nil
}

func TestNewSchedule(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	jobRan := false
	s := New(ctx, "test_new_schedule", "test_instance", 10*time.Millisecond, nopLocker{},
		WithJob("test_job", func(ctx context.Context) error {
			jobRan = true
			return nil
		}),
	)
	s.Start()

	time.Sleep(1 * time.Second)
	cancel()

	select {
	case <-s.Done():
		require.True(t, jobRan)
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

func (l *counterLocker) Unlock(context.Context, string, string) error {
	return nil
}

func TestScheduleLocker(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	locker := counterLocker{}
	jobRunCount := 0
	s := New(ctx, "test_schedule_locker", "test_instance", 10*time.Millisecond, &locker,
		WithJob("test_job", func(ctx context.Context) error {
			jobRunCount++
			return nil
		}),
	)
	s.Start()

	time.Sleep(1 * time.Second)
	cancel()

	select {
	case <-s.Done():
		require.Equal(t, locker.count, jobRunCount)
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
					ID: "test_job_1",
					Fn: func(ctx context.Context) error {
						setJobRun("test_job_1")
						return nil
					},
				},
			},
		},
		{
			name:       "test_schedule_2",
			instanceID: "test_instance",
			interval:   100 * time.Millisecond,
			jobs: []Job{
				{
					ID: "test_job_2",
					Fn: func(ctx context.Context) error {
						setJobRun("test_job_2")
						return nil
					},
				},
			},
		},
		{
			name:       "test_schedule_3",
			instanceID: "test_instance",
			interval:   100 * time.Millisecond,
			jobs: []Job{
				{
					ID: "test_job_3",
					Fn: func(ctx context.Context) error {
						setJobRun("test_job_3")
						return errors.New("job 3") // job 3 fails, job 4 should still run.
					},
				},
				{
					ID: "test_job_4",
					Fn: func(ctx context.Context) error {
						setJobRun("test_job_4")
						return nil
					},
				},
			},
		},
	} {
		var opts []Option
		for _, job := range tc.jobs {
			opts = append(opts, WithJob(job.ID, job.Fn))
			jobNames = append(jobNames, job.ID)
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
		WithJob("test_job_1", func(ctx context.Context) error {
			jobs <- 1
			return nil
		}),
		WithJob("test_job_2", func(ctx context.Context) error {
			jobs <- 2
			return errors.New("test_job_2")
		}),
		WithJob("test_job_3", func(ctx context.Context) error {
			jobs <- 3
			return nil
		}),
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

	jobRan := false
	s := New(ctx, "test_schedule", "test_instance", 200*time.Millisecond, nopLocker{},
		WithConfigReloadInterval(100*time.Millisecond, func(_ context.Context) (time.Duration, error) {
			return 50 * time.Millisecond, nil
		}),
		WithJob("test_job", func(ctx context.Context) error {
			jobRan = true
			return nil
		}),
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
		require.True(t, jobRan)
	case <-time.After(5 * time.Second):
		t.Error("timeout")
	}
}

func TestJobPanicRecover(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	jobRan := false

	s := New(ctx, "test_new_schedule", "test_instance", 10*time.Millisecond, nopLocker{},
		WithJob("job_1", func(ctx context.Context) error {
			panic("job_1")
		}),
		WithJob("job_2", func(ctx context.Context) error {
			jobRan = true
			return nil
		}))
	s.Start()

	time.Sleep(1 * time.Second)
	cancel()

	select {
	case <-s.Done():
		// job 2 should still run even though job 1 panicked
		require.True(t, jobRan)
	case <-time.After(5 * time.Second):
		t.Error("timeout")
	}
}

func TestScheduleReleaseLock(t *testing.T) {
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	start := time.Now()

	ds := new(mock.Store)
	var mockLock struct {
		owner  string
		expiry time.Time
		count  int

		mu sync.Mutex
	}

	ds.LockFunc = func(ctx context.Context, name string, owner string, expiration time.Duration) (bool, error) {
		mockLock.mu.Lock()
		defer mockLock.mu.Unlock()

		now := time.Now()
		if mockLock.owner == owner || now.After(mockLock.expiry) {
			mockLock.owner = owner
			mockLock.expiry = now.Add(expiration)
			mockLock.count = mockLock.count + 1

			return true, nil
		}
		return false, nil
	}

	unlock := make(chan int)
	ds.UnlockFunc = func(context.Context, string, string) error {
		unlock <- 1
		return nil
	}

	schedInterval := 100 * time.Millisecond
	jobDuration := 90 * time.Millisecond

	jobCount := 0
	s := New(ctx, "test_sched", "test_instance", schedInterval, ds, WithJob("test_job", func(ctx context.Context) error {
		time.Sleep(jobDuration)
		jobCount++
		return nil
	}))
	s.Start()

	select {
	// schedule starts first run and acquires a lock at 100ms
	// schedule extends the lock every 80ms (i.e. 8/10ths of the schedule interval)
	// schedule takes 90ms to complete its first run and unlocks at 190ms
	case <-unlock:
		require.Equal(t, 1, jobCount)       // schedule job starts at 100ms and finishes at 190ms
		require.Equal(t, 2, mockLock.count) // schedule locks at 100ms (acquire lock), 180ms (hold lock)
	case <-time.After(3 * time.Second):
		t.Errorf("timeout")
	}
	require.WithinRange(t, time.Now(), start.Add(190*time.Millisecond), start.Add(200*time.Millisecond))

	select {
	// schedule starts second run at 200ms (i.e. at the start of the next full interval following
	// the completion of the first run)
	case <-unlock:
		require.Equal(t, 2, jobCount)
		require.Equal(t, 4, mockLock.count)
	case <-time.After(3 * time.Second):
		t.Errorf("timeout")
	}
	require.WithinRange(t, time.Now(), start.Add(290*time.Millisecond), start.Add(300*time.Millisecond))
}

func TestScheduleHoldLock(t *testing.T) {
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	start := time.Now()

	ds := new(mock.Store)
	var mockLock struct {
		owner  string
		expiry time.Time
		count  int

		mu sync.Mutex
	}

	ds.LockFunc = func(ctx context.Context, name string, owner string, expiration time.Duration) (bool, error) {
		mockLock.mu.Lock()
		defer mockLock.mu.Unlock()

		now := time.Now()
		if mockLock.owner == owner || now.After(mockLock.expiry) {
			mockLock.owner = owner
			mockLock.expiry = now.Add(expiration)
			mockLock.count = mockLock.count + 1

			return true, nil
		}
		return false, nil
	}

	unlock := make(chan int)
	ds.UnlockFunc = func(context.Context, string, string) error {
		unlock <- 1
		return nil
	}

	schedInterval := 100 * time.Millisecond
	jobDuration := 210 * time.Millisecond

	jobCount := 0
	s := New(ctx, "test_sched", "test_instance", schedInterval, ds, WithJob("test_job", func(ctx context.Context) error {
		time.Sleep(jobDuration)
		jobCount++
		return nil
	}))
	s.Start()

	select {
	// schedule starts first run and acquires a lock at 100ms
	// schedule extends the lock every 80ms (i.e. 8/10ths of the schedule interval)
	// schedule takes 210ms to complete its first run and unlocks at 310ms
	case <-unlock:
		require.Equal(t, 1, jobCount)       // schedule job starts at 100ms and finishes at 300ms
		require.Equal(t, 3, mockLock.count) // schedule locks at 100ms (acquire lock), 180ms (hold lock), 260ms (hold lock)
	case <-time.After(3 * time.Second):
		t.Errorf("timeout")
	}
	require.WithinRange(t, time.Now(), start.Add(310*time.Millisecond), start.Add(320*time.Millisecond))

	select {
	// schedule starts second run at 400ms (i.e. at the start of the next full interval following
	// the completion of the first run)
	case <-unlock:
		require.Equal(t, 2, jobCount)
		require.Equal(t, 6, mockLock.count)
	case <-time.After(3 * time.Second):
		t.Errorf("timeout")
	}
	require.WithinRange(t, time.Now(), start.Add(610*time.Millisecond), start.Add(620*time.Millisecond))
}

func TestMultipleScheduleInstancesConfigChanges(t *testing.T) {
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	ds := new(mock.Store)
	mockLock := struct {
		owner  string
		expiry time.Time
		mu     sync.Mutex
	}{
		owner:  "a",
		expiry: time.Now().Add(1 * time.Hour),
	}
	ds.LockFunc = func(ctx context.Context, name string, owner string, expiration time.Duration) (bool, error) {
		mockLock.mu.Lock()
		defer mockLock.mu.Unlock()

		now := time.Now()
		if mockLock.owner == owner || now.After(mockLock.expiry) {
			mockLock.owner = owner
			mockLock.expiry = now.Add(expiration)
			return true, nil
		}
		return false, nil
	}
	ds.UnlockFunc = func(context.Context, string, string) error {
		return nil
	}

	mockConfig := struct {
		sync.Mutex
		duration time.Duration
	}{
		duration: 1 * time.Hour,
	}
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		mockConfig.Lock()
		defer mockConfig.Unlock()

		return &fleet.AppConfig{
			WebhookSettings: fleet.WebhookSettings{
				Interval: fleet.Duration{
					Duration: mockConfig.duration,
				},
			},
		}, nil
	}
	setMockConfigInterval := func(d time.Duration) {
		mockConfig.Lock()
		defer mockConfig.Unlock()
		mockConfig.duration = d
	}
	// simulate changes to app config
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		onTick := map[int]func(){
			1: func() { setMockConfigInterval(1 * time.Second) },
			2: func() { setMockConfigInterval(3 * time.Second) },
			5: func() { setMockConfigInterval(4 * time.Second) },
		}
		tick := 0
		for {
			select {
			case <-ticker.C:
				tick++
				if fn, ok := onTick[tick]; ok {
					fn()
				}
			case <-time.After(10 * time.Second):
				return
			}
		}
	}()

	jobsRun := 0
	newInstanceWithSchedule := func(id string) {
		s := New(
			ctx, "test_schedule", id, 1*time.Hour, ds,
			WithConfigReloadInterval(200*time.Millisecond, func(ctx context.Context) (time.Duration, error) {
				ac, _ := ds.AppConfigFunc(ctx)
				return ac.WebhookSettings.Interval.Duration, nil
			}),
			WithJob("test_job", func(ctx context.Context) error {
				jobsRun++
				return nil
			}),
		)
		s.Start()
	}
	// simulate multiple schedule instances
	go func() {
		instanceIDs := strings.Split("abcdefghijklmnopqrstuvwxyz", "")
		for _, id := range instanceIDs {
			time.Sleep(300 * time.Millisecond)
			newInstanceWithSchedule(id)
		}
	}()

	<-time.After(10 * time.Second)
	require.Equal(t, 3, jobsRun)
}
