package schedule

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
)

func TestNewSchedule(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	jobRan := false
	s := New(ctx, "test_new_schedule", "test_instance", 10*time.Millisecond, NopLocker{}, NopStatsStore{},
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

func TestScheduleLocker(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	name := "test_schedule_locker"
	instance := "test_instance"
	interval := 10 * time.Millisecond
	locker := SetupMockLocker(name, instance, time.Now().Add(-interval))

	jobRunCount := 0
	s := New(ctx, name, instance, interval, locker, &NopStatsStore{},
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
		require.Equal(t, locker.GetLockCount(), jobRunCount)
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
			interval:   1000 * time.Millisecond,
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
			interval:   2000 * time.Millisecond,
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
			interval:   1000 * time.Millisecond,
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
		s := New(ctx, tc.name, tc.instanceID, tc.interval, NopLocker{}, NopStatsStore{}, opts...)
		s.Start()
		ss = append(ss, s)
	}

	time.Sleep(2500 * time.Millisecond)
	cancel()

	for i, s := range ss {
		select {
		case <-s.Done():
			// OK
		case <-time.After(3000 * time.Millisecond):
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

	s := New(ctx, "test_schedule", "test_instance", 1000*time.Millisecond, NopLocker{}, NopStatsStore{},
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

	time.Sleep(1500 * time.Millisecond)
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
	initialSchedInterval := 200 * time.Millisecond
	newSchedInterval := 2600 * time.Millisecond

	jobsRun := 0
	s := New(ctx, "test_schedule", "test_instance", initialSchedInterval, NopLocker{}, NopStatsStore{},
		WithConfigReloadInterval(100*time.Millisecond, func(_ context.Context) (time.Duration, error) {
			return newSchedInterval, nil
		}),
		WithJob("test_job", func(ctx context.Context) error {
			jobsRun++
			return nil
		}),
	)

	require.Equal(t, s.getSchedInterval(), 1*time.Second) // schedule interval floor is 1s so initial schedule interval of 200ms becomes 1s
	require.Equal(t, s.configReloadInterval, 100*time.Millisecond)

	s.Start()

	time.Sleep(2 * time.Second)
	cancel()

	select {
	case <-s.Done():
		require.Equal(t, s.getSchedInterval(), 2*time.Second) // schedule intervals above the 1s floor are rounded down to the nearest second so 2600ms becomes 2s
		require.Equal(t, s.configReloadInterval, 100*time.Millisecond)
		require.Equal(t, 1, jobsRun)
	case <-time.After(5 * time.Second):
		t.Error("timeout")
	}
}

func TestJobPanicRecover(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	jobRan := false

	s := New(ctx, "test_new_schedule", "test_instance", 10*time.Millisecond, NopLocker{}, NopStatsStore{},
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

	name := "test_sched"
	instance := "test_instance"
	schedInterval := 2000 * time.Millisecond
	jobDuration := 1900 * time.Millisecond

	ml := SetupMockLocker(name, instance, time.Now().Add(-schedInterval))
	err := ml.AddChannels(t, "unlocked")
	require.NoError(t, err)

	ms := SetUpMockStatsStore(name, fleet.CronStats{
		ID:        1,
		StatsType: fleet.CronStatsTypeScheduled,
		Name:      name,
		Instance:  instance,
		CreatedAt: time.Now().Truncate(time.Second).Add(-schedInterval),
		UpdatedAt: time.Now().Truncate(time.Second).Add(-schedInterval),
		Status:    fleet.CronStatsStatusCompleted,
	})

	jobCount := 0
	s := New(ctx, name, instance, schedInterval, ml, ms, WithJob("test_job", func(ctx context.Context) error {
		time.Sleep(jobDuration)
		jobCount++
		return nil
	}))
	s.Start()
	start := time.Now()

	// schedule starts first run and acquires a lock at 2000ms
	// schedule extends the lock every 1600ms (i.e. 8/10ths of the schedule interval)
	// schedule takes 2100ms to complete its first run and unlocks at 4100ms
	select {
	case <-time.After(10 * time.Second):
		t.Errorf("timeout")
		t.FailNow()
	case <-ml.Unlocked:
		require.Equal(t, 1, jobCount)          // schedule job starts at 2000ms and finishes at 3900ms
		require.Equal(t, 2, ml.GetLockCount()) // schedule locks at 2000ms (acquire lock), 3600ms (hold lock)
	}

	// schedule starts second run at 4000ms (i.e. at the start of the next full interval following
	// the completion of the first run)
	select {
	case <-time.After(10 * time.Second):
		t.Errorf("timeout")
		t.FailNow()
	case <-ml.Unlocked:
		require.Equal(t, 2, jobCount)
		require.Equal(t, 4, ml.GetLockCount())
		require.True(t, time.Now().After(start.Add(2*schedInterval).Add(jobDuration)))
	}
}

func TestScheduleHoldLock(t *testing.T) {
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	name := "test_schedule_hold_lock"
	instance := "test_instance"
	schedInterval := 2000 * time.Millisecond
	jobDuration := 2100 * time.Millisecond

	ml := SetupMockLocker(name, instance, time.Now().Add(-schedInterval))
	ml.AddChannels(t, "unlocked")

	ms := SetUpMockStatsStore(name, fleet.CronStats{
		ID:        1,
		StatsType: fleet.CronStatsTypeScheduled,
		Name:      name,
		Instance:  instance,
		CreatedAt: time.Now().Truncate(time.Second).Add(-schedInterval),
		UpdatedAt: time.Now().Truncate(time.Second).Add(-schedInterval),
		Status:    fleet.CronStatsStatusCompleted,
	})

	jobCount := 0
	s := New(ctx, name, instance, schedInterval, ml, ms, WithJob("test_job", func(ctx context.Context) error {
		time.Sleep(jobDuration)
		jobCount++
		return nil
	}))
	s.Start()
	start := time.Now()

	// schedule starts first run and acquires a lock at 2000ms
	// schedule extends the lock every 1600ms (i.e. 8/10ths of the schedule interval)
	// schedule takes 2100ms to complete its first run and unlocks at 4100ms
	select {
	case <-time.After(10 * time.Second):
		t.Errorf("timeout")
		t.FailNow()
	case <-ml.Unlocked:
		require.Equal(t, 1, jobCount)          // schedule job starts at 2000ms and finishes at 5100ms
		require.Equal(t, 2, ml.GetLockCount()) // schedule locks at 2000ms (acquire lock), 3600ms (hold lock)
	}

	// schedule starts second run at 6000ms (i.e. at the start of the next full interval following
	// the completion of the first run)
	select {
	case <-time.After(10 * time.Second):
		t.Errorf("timeout")
		t.FailNow()
	case <-ml.Unlocked:
		require.Equal(t, 2, jobCount)
		require.Equal(t, 4, ml.GetLockCount())
		require.WithinRange(t, time.Now(),
			start.Add(3*schedInterval).Add(jobDuration),
			start.Add(3*schedInterval).Add(jobDuration).Add(2*time.Second))
	}
}

func TestMultipleScheduleInstancesConfigChangesDS(t *testing.T) {
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	name := "test_multiple_schedule_instances_config_change"
	initialSchedInterval := 1 * time.Hour

	ds := mysql.CreateMySQLDS(t)
	defer ds.Close()

	ds.InsertCronStats(ctx, fleet.CronStatsTypeScheduled, name, "a", fleet.CronStatsStatusCompleted)

	ac, err := ds.AppConfig(ctx)
	require.NoError(t, err)
	ac.WebhookSettings.Interval.Duration = initialSchedInterval
	err = ds.SaveAppConfig(ctx, ac)
	require.NoError(t, err)

	setMockConfigInterval := func(d time.Duration) {
		ac, err := ds.AppConfig(ctx)
		require.NoError(t, err)
		ac.WebhookSettings.Interval.Duration = d

		err = ds.SaveAppConfig(ctx, ac)
		require.NoError(t, err)
	}

	// simulate changes to app config
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		onTick := map[int]func(){
			1: func() { setMockConfigInterval(2 * time.Second) },
			2: func() { setMockConfigInterval(3 * time.Second) },
			5: func() { setMockConfigInterval(4 * time.Second) },
		}
		tick := 0
		for tick < 5 {
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

	jobsRun := uint32(0)
	newInstanceWithSchedule := func(id string) {
		s := New(
			ctx, name, id, initialSchedInterval, ds, ds,
			WithConfigReloadInterval(1*time.Second, func(ctx context.Context) (time.Duration, error) {
				ac, _ := ds.AppConfig(ctx)
				return ac.WebhookSettings.Interval.Duration, nil
			}),
			WithJob("test_job", func(ctx context.Context) error {
				time.Sleep(500 * time.Millisecond)
				atomic.AddUint32(&jobsRun, 1)
				return nil
			}),
		)
		s.Start()
	}
	// simulate multiple schedule instances
	go func() {
		instanceIDs := strings.Split("abcdefghijklmnopqrstuvwxyz", "")
		for _, id := range instanceIDs {
			time.Sleep(600 * time.Millisecond)
			newInstanceWithSchedule(id)
		}
	}()

	<-time.After(10 * time.Second)
	require.Equal(t, uint32(3), atomic.LoadUint32(&jobsRun))
}

func TestTriggerSingleInstance(t *testing.T) {
	ctx, cancelFn := context.WithCancel(context.Background())
	defer cancelFn()

	name := "test_trigger_single_instance"
	instanceID := "test_instance"
	schedInterval := 1 * time.Second
	jobRuntime := 200 * time.Millisecond

	locker := SetupMockLocker(name, instanceID, time.Now().Add(-schedInterval))
	statsStore := SetUpMockStatsStore(name, fleet.CronStats{
		ID:        1,
		StatsType: fleet.CronStatsTypeScheduled,
		Name:      name,
		Instance:  instanceID,
		CreatedAt: time.Now().Add(-schedInterval).Add(-jobRuntime),
		UpdatedAt: time.Now().Add(-schedInterval),
		Status:    fleet.CronStatsStatusCompleted,
	})

	jobsRun := uint32(0)
	s := New(
		ctx, name, instanceID, schedInterval, locker, statsStore,
		WithJob("test_job", func(ctx context.Context) error {
			time.Sleep(jobRuntime)
			atomic.AddUint32(&jobsRun, 1)
			return nil
		}),
	)
	s.Start()

	ticker := time.NewTicker(schedInterval) // 1s interval
	time.Sleep(100 * time.Millisecond)
	s.Trigger() // triggered run starts at 0.1s and runs until 0.3s
	s.Trigger() // ignored
	s.Trigger() // ignored
	s.Trigger() // ignored
	s.Trigger() // ignored

	<-ticker.C  // scheduled run starts at 1s and runs until 1.2s
	s.Trigger() // ignored
	time.Sleep(100 * time.Millisecond)
	s.Trigger() // ignored
	time.Sleep(200 * time.Millisecond)
	s.Trigger() // triggered run starts at 1.3s and runs until 1.5s

	<-ticker.C // sheduled run starts at 2s and runs until at 2.2s
	time.Sleep(900 * time.Millisecond)
	s.Trigger() // triggered run starts at 2.9s and runs until at 3.1s

	<-ticker.C // scheduled run at 3s gets skipped because triggered run is pending

	<-ticker.C // scheduled run starts at 4s and runs until at 4.2s
	time.Sleep(300 * time.Millisecond)
	ticker.Stop()

	require.Equal(t, uint32(6), atomic.LoadUint32(&jobsRun))
}

func TestTriggerMultipleInstances(t *testing.T) {
	schedInterval := 1 * time.Second
	testDuration := 3900 * time.Millisecond

	cases := []struct {
		name         string
		triggerDelay time.Duration
		jobRuntime   time.Duration
		jobsExpected int
	}{
		{
			name:         "test_trigger_long_runtime",
			triggerDelay: 10 * time.Millisecond,
			jobRuntime:   1300 * time.Millisecond,
			jobsExpected: 2, // 1 triggered plus 1 scheduled (at 2s)
		},
		{
			name:         "test_trigger_short_runtime",
			triggerDelay: 10 * time.Millisecond,
			jobRuntime:   400 * time.Millisecond,
			jobsExpected: 4, // 1 triggered plus 3 scheduled (at 1s, 2s, 3s)
		},
		{
			name:         "test_no_trigger_while_pending",
			triggerDelay: 1100 * time.Millisecond,
			jobRuntime:   200 * time.Millisecond,
			jobsExpected: 3, // none triggered, 3 scheduled (at 1s, 2s, 3s)
		},
	}

	for _, c := range cases {
		ctx, cancelFunc := context.WithCancel(context.Background())

		instanceIDs := strings.Split("abcdef", "")
		locker := SetupMockLocker(c.name, instanceIDs[0], time.Now().Add(-schedInterval))
		statsStore := SetUpMockStatsStore(c.name, fleet.CronStats{
			ID:        1,
			StatsType: fleet.CronStatsTypeScheduled,
			Name:      c.name,
			Instance:  instanceIDs[0],
			CreatedAt: time.Now().Add(-schedInterval).Add(-c.jobRuntime),
			UpdatedAt: time.Now().Add(-schedInterval),
			Status:    fleet.CronStatsStatusCompleted,
		})

		jobsRun := uint32(0)
		scheduleInstances := []*Schedule{}
		newInstanceWithSchedule := func(id string) {
			s := New(
				ctx, c.name, id, schedInterval, locker, statsStore,
				WithJob("test_job", func(ctx context.Context) error {
					time.Sleep(c.jobRuntime)
					atomic.AddUint32(&jobsRun, 1)
					return nil
				}),
			)
			s.Start()
			scheduleInstances = append(scheduleInstances, s)
		}
		// simulate multiple schedule instances
		for _, id := range instanceIDs {
			newInstanceWithSchedule(id)
		}

		timer := time.NewTimer(testDuration)

		go func() {
			time.Sleep(c.triggerDelay)
			scheduleInstances[1].Trigger()
		}()

		<-timer.C
		require.Equal(t, uint32(c.jobsExpected), atomic.LoadUint32(&jobsRun))
		cancelFunc()
	}
}
