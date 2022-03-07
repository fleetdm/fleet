package schedule

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/go-kit/log"
	"github.com/stretchr/testify/require"
)

func nopStatsHandler(interface{}, error) {}

type nopLocker struct{}

// counter++
// should lock return tl.Lock()

func (nopLocker) Lock(ctx context.Context, name string, owner string, expiration time.Duration) (bool, error) {
	return true, nil
}

func (nopLocker) Unlock(ctx context.Context, name string, owner string) error {
	return nil
}

func TestNewSchedule(t *testing.T) {
	sched, err := New(context.Background(), "test_new_schedule", "test_instance", 10*time.Millisecond, nopLocker{}, log.NewNopLogger())
	require.NoError(t, err)

	runCheck := make(chan bool)
	failCheck := time.After(1 * time.Second)

	sched.AddJob("test_job", func(ctx context.Context) (interface{}, error) {
		runCheck <- true
		return nil, nil
	}, nopStatsHandler)

	runCount := 0

TEST:
	for {
		select {
		case <-runCheck:
			runCount++
			if runCount > 2 {
				break TEST
			}
		case <-failCheck:
			t.FailNow()
		}
	}
}

func TestStatsHandler(t *testing.T) {
	sched, err := New(context.Background(), "test_stats_handler", "test_instance", 10*time.Millisecond, nopLocker{}, log.NewNopLogger())
	require.NoError(t, err)

	runCheck := make(chan bool)
	failCheck := time.After(1 * time.Second)

	sched.AddJob("test_job", func(ctx context.Context) (interface{}, error) {
		runCheck <- true
		return "stats foo", fmt.Errorf("stats error")
	}, func(stats interface{}, err error) {
		require.Equal(t, "stats foo", stats)
		require.Equal(t, fmt.Errorf("stats error"), err)
	})

	runCount := 0

TEST:
	for {
		select {
		case <-runCheck:
			runCount++
			if runCount > 2 {
				break TEST
			}
		case <-failCheck:
			t.FailNow()
		}
	}
}

type testLocker struct {
	count *uint
}

func (l testLocker) Lock(ctx context.Context, name string, owner string, expiration time.Duration) (bool, error) {
	*l.count++
	return true, nil
}

func (testLocker) Unlock(ctx context.Context, name string, owner string) error {
	return nil
}

func TestScheduleLocker(t *testing.T) {
	locker := testLocker{count: ptr.Uint(0)}

	sched, err := New(context.Background(), "test_schedule_locker", "test_instance", 10*time.Millisecond, locker, log.NewNopLogger())
	require.NoError(t, err)

	runCheck := make(chan bool)
	failCheck := time.After(1 * time.Second)

	sched.AddJob("test_job", func(ctx context.Context) (interface{}, error) {
		runCheck <- true
		return nil, nil
	}, nopStatsHandler)

TEST:
	for {
		select {
		case <-runCheck:
			if *locker.count > 2 {
				break TEST
			}
		case <-failCheck:
			t.FailNow()
		}
	}
}

func statsHandlerFunc(jobName string, testStats map[string][]interface{}, testErrors map[string][]error) func(interface{}, error) {
	return func(stats interface{}, err error) {
		if err != nil {
			testErrors[jobName] = append(testErrors[jobName], err)
		}
		testStats[jobName] = append(testStats[jobName], stats)
	}
}

func TestMultipleSchedules(t *testing.T) {
	testStats := make(map[string][]interface{})
	testErrors := make(map[string][]error)

	sched1, err := New(context.Background(), "test_schedule_1", "test_instance", 10*time.Millisecond, nopLocker{}, log.NewNopLogger())
	require.NoError(t, err)

	runCheck := make(chan bool)
	failCheck := time.After(1 * time.Second)

	sched1.AddJob("test_job_1", func(ctx context.Context) (interface{}, error) {
		runCheck <- true
		return "stats_job_1", nil
	}, statsHandlerFunc("test_job_1", testStats, testErrors))

	sched2, err := New(context.Background(), "test_schedule_2", "test_instance", 100*time.Millisecond, nopLocker{}, log.NewNopLogger())
	require.NoError(t, err)

	sched2.AddJob("test_job_2", func(ctx context.Context) (interface{}, error) {
		runCheck <- true
		return "stats_job_2", nil
	}, statsHandlerFunc("test_job_2", testStats, testErrors))

	sched3, err := New(context.Background(), "test_schedule_3", "test_instance", 100*time.Millisecond, nopLocker{}, log.NewNopLogger())
	require.NoError(t, err)

	sched3.AddJob("test_job_3", func(ctx context.Context) (interface{}, error) {
		runCheck <- true
		return nil, fmt.Errorf("error_job_3")
	}, statsHandlerFunc("test_job_3", testStats, testErrors))

TEST:
	for {
		select {
		case <-runCheck:
			if (len(testStats["test_job_1"]) > 2) && (len(testStats["test_job_2"]) > 2) && (len(testErrors["test_job_3"]) > 2) {
				break TEST
			}
		case <-failCheck:
			t.FailNow()
		}
	}
}
