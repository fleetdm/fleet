package schedule

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
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

type nopStatsStore struct{}

func (nopStatsStore) GetLatestCronStats(ctx context.Context, name string) (*fleet.CronStats, error) {
	return &fleet.CronStats{}, nil
}

func (nopStatsStore) InsertCronStats(ctx context.Context, statsType fleet.CronStatsType, name string, instance string, status fleet.CronStatsStatus) (int, error) {
	return 0, nil
}

func (nopStatsStore) UpdateCronStats(ctx context.Context, id int, status fleet.CronStatsStatus) error {
	return nil
}

func TestNewSchedule(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	jobRan := false
	s := New(ctx, "test_new_schedule", "test_instance", 10*time.Millisecond, nopLocker{}, nopStatsStore{},
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
	locker := setupMockLocker(name, instance, time.Now().Add(-interval))

	jobRunCount := 0
	s := New(ctx, name, instance, interval, locker, &nopStatsStore{},
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
		require.Equal(t, locker.getLockCount(), jobRunCount)
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
		s := New(ctx, tc.name, tc.instanceID, tc.interval, nopLocker{}, nopStatsStore{}, opts...)
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

	s := New(ctx, "test_schedule", "test_instance", 1000*time.Millisecond, nopLocker{}, nopStatsStore{},
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
	s := New(ctx, "test_schedule", "test_instance", initialSchedInterval, nopLocker{}, nopStatsStore{},
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

	s := New(ctx, "test_new_schedule", "test_instance", 10*time.Millisecond, nopLocker{}, nopStatsStore{},
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

	ml := setupMockLocker(name, instance, time.Now().Add(-schedInterval))
	err := ml.addChannels(t, "unlocked")
	require.NoError(t, err)

	ms := setUpMockStatsStore(name, fleet.CronStats{
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
	case <-ml.unlocked:
		require.Equal(t, 1, jobCount)          // schedule job starts at 2000ms and finishes at 3900ms
		require.Equal(t, 2, ml.getLockCount()) // schedule locks at 2000ms (acquire lock), 3600ms (hold lock)
	}

	// schedule starts second run at 4000ms (i.e. at the start of the next full interval following
	// the completion of the first run)
	select {
	case <-time.After(10 * time.Second):
		t.Errorf("timeout")
		t.FailNow()
	case <-ml.unlocked:
		require.Equal(t, 2, jobCount)
		require.Equal(t, 4, ml.getLockCount())
		require.WithinRange(t, time.Now(),
			start.Add(2*schedInterval).Add(jobDuration),
			start.Add(2*schedInterval).Add(jobDuration).Add(20*time.Millisecond),
		)
	}
}

func TestScheduleHoldLock(t *testing.T) {
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	name := "test_schedule_hold_lock"
	instance := "test_instance"
	schedInterval := 2000 * time.Millisecond
	jobDuration := 2100 * time.Millisecond

	ml := setupMockLocker(name, instance, time.Now().Add(-schedInterval))
	ml.addChannels(t, "unlocked")

	ms := setUpMockStatsStore(name, fleet.CronStats{
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
	case <-ml.unlocked:
		require.Equal(t, 1, jobCount)          // schedule job starts at 2000ms and finishes at 5100ms
		require.Equal(t, 2, ml.getLockCount()) // schedule locks at 2000ms (acquire lock), 3600ms (hold lock)
	}

	// schedule starts second run at 6000ms (i.e. at the start of the next full interval following
	// the completion of the first run)
	select {
	case <-time.After(10 * time.Second):
		t.Errorf("timeout")
		t.FailNow()
	case <-ml.unlocked:
		require.Equal(t, 2, jobCount)
		require.Equal(t, 4, ml.getLockCount())
		require.WithinRange(t, time.Now(),
			start.Add(3*schedInterval).Add(jobDuration),
			start.Add(3*schedInterval).Add(jobDuration).Add(20*time.Millisecond),
		)
	}
}

func TestMultipleScheduleInstancesConfigChanges(t *testing.T) {
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

	jobsRun := 0
	newInstanceWithSchedule := func(id string) {
		s := New(
			ctx, name, id, initialSchedInterval, ds, ds,
			WithConfigReloadInterval(1*time.Second, func(ctx context.Context) (time.Duration, error) {
				ac, _ := ds.AppConfig(ctx)
				return ac.WebhookSettings.Interval.Duration, nil
			}),
			WithJob("test_job", func(ctx context.Context) error {
				time.Sleep(500 * time.Millisecond)
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
			time.Sleep(600 * time.Millisecond)
			newInstanceWithSchedule(id)
		}
	}()

	<-time.After(10 * time.Second)
	require.Equal(t, 3, jobsRun)
}

type mockLock struct {
	mu sync.Mutex

	name      string
	owner     string
	expiresAt time.Time

	locked    chan struct{}
	lockCount int

	unlocked    chan struct{}
	unlockCount int
}

func (ml *mockLock) Lock(ctx context.Context, name string, owner string, expiration time.Duration) (bool, error) {
	ml.mu.Lock()
	defer ml.mu.Unlock()

	if name != ml.name {
		return false, fmt.Errorf("name doesn't match")
	}

	now := time.Now()
	if ml.owner == owner || now.After(ml.expiresAt) {
		ml.owner = owner
		ml.expiresAt = now.Add(expiration)
		ml.lockCount = ml.lockCount + 1
		if ml.locked != nil {
			ml.locked <- struct{}{}
		}
		return true, nil
	}

	return false, nil
}

func (ml *mockLock) Unlock(ctx context.Context, name string, owner string) error {
	ml.mu.Lock()
	defer ml.mu.Unlock()

	if name != ml.name {
		return fmt.Errorf("name doesn't match")
	}
	if owner != ml.owner {
		return fmt.Errorf("owner doesn't match")
	}
	ml.unlockCount = ml.unlockCount + 1
	if ml.unlocked != nil {
		ml.unlocked <- struct{}{}
	}
	return nil
}

func (ml *mockLock) getLockCount() int {
	ml.mu.Lock()
	defer ml.mu.Unlock()

	return ml.lockCount
}

func (ml *mockLock) getUnlockCount() int {
	ml.mu.Lock()
	defer ml.mu.Unlock()

	return ml.unlockCount
}

func (ml *mockLock) addChannels(t *testing.T, chanNames ...string) error {
	ml.mu.Lock()
	defer ml.mu.Unlock()

	for _, n := range chanNames {
		switch n {
		case "locked":
			ml.locked = make(chan struct{})
		case "unlocked":
			ml.unlocked = make(chan struct{})
		default:
			t.Errorf("unrecognized channel name")
			t.FailNow()
		}
	}

	return nil
}

func setupMockLocker(name string, owner string, expiresAt time.Time) *mockLock {
	return &mockLock{name: name, owner: owner, expiresAt: expiresAt}
}

type mockStatsStore struct {
	sync.Mutex
	stats map[int]fleet.CronStats

	getStatsCalled    chan struct{}
	insertStatsCalled chan struct{}
	updateStatsCalled chan struct{}
}

func (m *mockStatsStore) GetLatestCronStats(ctx context.Context, name string) (*fleet.CronStats, error) {
	m.Lock()
	defer m.Unlock()

	var res *fleet.CronStats
	for _, s := range m.stats {
		s := s
		switch {
		case s.Name != name:
			continue
		case res != nil && res.CreatedAt.After(s.CreatedAt) && res.ID > s.ID:
			continue
		default:
			res = &s
		}
	}
	if res == nil {
		return nil, sql.ErrNoRows
	}

	return res, nil
}

func (m *mockStatsStore) InsertCronStats(ctx context.Context, statsType fleet.CronStatsType, name string, instance string, status fleet.CronStatsStatus) (int, error) {
	m.Lock()
	defer m.Unlock()

	id := len(m.stats) + 1
	m.stats[id] = fleet.CronStats{ID: id, StatsType: statsType, Name: name, Instance: instance, Status: status, CreatedAt: time.Now().Truncate(time.Second), UpdatedAt: time.Now().Truncate(time.Second)}

	return id, nil
}

func (m *mockStatsStore) UpdateCronStats(ctx context.Context, id int, status fleet.CronStatsStatus) error {
	m.Lock()
	defer m.Unlock()

	s, ok := m.stats[id]
	if !ok {
		return fmt.Errorf("update failed, id not found")
	}
	s.Status = status
	s.UpdatedAt = time.Now().Truncate(time.Second)
	m.stats[id] = s

	return nil
}

func (m *mockStatsStore) addChannels(t *testing.T, chanNames ...string) error {
	m.Lock()
	defer m.Unlock()

	for _, n := range chanNames {
		switch n {
		case "getStatsCalled":
			m.getStatsCalled = make(chan struct{})
		case "insertStatsCalled":
			m.insertStatsCalled = make(chan struct{})
		case "updateStatsCalled":
			m.updateStatsCalled = make(chan struct{})
		default:
			t.Errorf("unrecognized channel name")
			t.FailNow()
		}
	}

	return nil
}

func setUpMockStatsStore(name string, initialStats ...fleet.CronStats) *mockStatsStore {
	stats := make(map[int]fleet.CronStats)
	for _, s := range initialStats {
		stats[s.ID] = s
	}
	store := mockStatsStore{stats: stats, getStatsCalled: make(chan struct{}), insertStatsCalled: make(chan struct{}), updateStatsCalled: make(chan struct{})}

	return &store
}
