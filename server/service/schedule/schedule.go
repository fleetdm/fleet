// Package schedule allows periodic run of a list of jobs.
//
// Type Schedule allows grouping a set of Jobs to run at specific intervals.
// Each Job is executed serially in the order they were added to the Schedule.
package schedule

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/getsentry/sentry-go"
	kitlog "github.com/go-kit/kit/log"
	"github.com/go-kit/log/level"
)

// ReloadInterval reloads and returns a new interval.
type ReloadInterval func(ctx context.Context) (time.Duration, error)

// Schedule runs a list of jobs serially at a given schedule.
//
// Each job is executed one after the other in the order they were added.
// If one of the job fails, an error is logged and the scheduler
// continues with the next.
type Schedule struct {
	ctx        context.Context
	name       string
	instanceID string
	logger     kitlog.Logger

	muChecks       sync.Mutex // protects configInterval and schedInterval.
	configInterval time.Duration
	schedInterval  time.Duration

	reloadInterval ReloadInterval
	locker         Locker

	jobs []Job
}

// JobFn is the signature of a Job.
type JobFn func(context.Context) error

// Job represents a job that can be added to Scheduler.
type Job struct {
	// ID is the unique identifier for the job.
	ID string
	// Fn is the job itself.
	Fn func(context.Context) error
}

// Locker allows a Schedule to acquire and release a lock before running jobs.
type Locker interface {
	Lock(ctx context.Context, name string, owner string, expiration time.Duration) (bool, error)
	Unlock(ctx context.Context, name string, owner string) error
}

// Option allows configuring a Schedule.
type Option func(*Schedule)

// WithLogger sets a logger for the Schedule.
func WithLogger(l kitlog.Logger) Option {
	return func(s *Schedule) {
		s.logger = l
	}
}

// WithReloadInterval allows setting a reload interval function,
// that will allow updating the interval of a running schedule.
//
// If not set, then the schedule performs no interval reloading.
func WithReloadInterval(fn ReloadInterval) Option {
	return func(s *Schedule) {
		s.reloadInterval = fn
	}
}

// WithJob adds a job to the Schedule.
//
// Each job is executed in the order they are added.
func WithJob(id string, fn JobFn) Option {
	return func(s *Schedule) {
		s.jobs = append(s.jobs, Job{
			ID: id,
			Fn: fn,
		})
	}
}

// New creates and returns a Schedule.
// Jobs are added with the WithJob Option.
//
// The jobs are executed serially in order at the provided interval.
//
// The provided locker is used to acquire/release a lock before running the jobs.
// The provided name and instanceID of the Schedule is used as the locking identifier.
func New(
	ctx context.Context,
	name string,
	instanceID string,
	interval time.Duration,
	locker Locker,
	opts ...Option,
) *Schedule {
	sch := &Schedule{
		ctx:        ctx,
		name:       name,
		instanceID: instanceID,

		configInterval: 1 * time.Hour, // by default we will check for updated config once per hour
		schedInterval:  interval,
		locker:         locker,
	}
	for _, fn := range opts {
		fn(sch)
	}
	return sch
}

// Start starts running the added jobs.
//
// All jobs must be added before calling Start.
func (s *Schedule) Start() {
	currentStart := time.Now()
	currentWait := 10 * time.Second

	getWaitTimes := func() (start time.Time, wait time.Duration) {
		s.muChecks.Lock()
		defer s.muChecks.Unlock()
		return currentStart, currentWait
	}

	setWaitTimes := func(start time.Time, wait time.Duration) {
		s.muChecks.Lock()
		defer s.muChecks.Unlock()
		currentStart = start
		currentWait = wait
	}

	if i := s.getSchedInterval(); i < currentWait {
		setWaitTimes(currentStart, i)
	}

	// this is the main loop for the schedule
	schedTicker := time.NewTicker(currentWait)
	go func() {
		for {
			_, currWait := getWaitTimes()
			level.Debug(s.logger).Log("waiting", fmt.Sprint("current wait time... ", currWait))

			select {
			case <-s.ctx.Done():
				level.Debug(s.logger).Log("exit", fmt.Sprint("done with ", s.name))
				return

			case <-schedTicker.C:
				level.Debug(s.logger).Log("waiting", "done")

				schedInterval := s.getSchedInterval()
				schedTicker.Reset(schedInterval)

				newStart := time.Now()
				newWait := schedInterval
				setWaitTimes(newStart, newWait)

				if ok := s.acquireLock(); !ok {
					continue
				}

				for _, job := range s.jobs {
					level.Debug(s.logger).Log(s.name, fmt.Sprint("starting job... ", job.ID))
					if err := job.Fn(s.ctx); err != nil {
						level.Error(s.logger).Log("job", job.ID, "err", err)
						sentry.CaptureException(err)
					}
				}
			}
		}
	}()

	// this periodically checks for config updates and resets the schedInterval for the main loop
	configTicker := time.NewTicker(200 * time.Millisecond)
	go func() {
		for {
			select {
			case <-configTicker.C:
				level.Debug(s.logger).Log(s.name, "config check...")

				configTicker.Reset(s.configInterval)

				schedInterval := s.getSchedInterval()
				currStart, _ := getWaitTimes()

				if s.reloadInterval == nil {
					level.Debug(s.logger).Log(s.name, "config check function has not been set... skipping...")
					continue
				}

				newInterval, err := s.reloadInterval(s.ctx)
				if err != nil {
					level.Error(s.logger).Log("config", "could not check for updates to schedule interval", "err", err)
					sentry.CaptureException(err)
					continue
				}
				if schedInterval == newInterval {
					level.Debug(s.logger).Log(s.name, "schedule interval unchanged")
					continue
				}
				s.setSchedInterval(newInterval)

				newWait := 10 * time.Millisecond
				if time.Since(currStart) < newInterval {
					newWait = newInterval - time.Since(currStart)
				}
				setWaitTimes(currStart, newWait)
				schedTicker.Reset(newWait)

				level.Debug(s.logger).Log("schedule", s.name, "new schedule interval", newInterval, "wait time until next job run: ", newWait)
			}
		}
	}()
}

func (s *Schedule) getSchedInterval() time.Duration {
	s.muChecks.Lock()
	defer s.muChecks.Unlock()

	return s.schedInterval
}

func (s *Schedule) setSchedInterval(interval time.Duration) {
	s.muChecks.Lock()
	defer s.muChecks.Unlock()

	s.schedInterval = interval
}

func (s *Schedule) acquireLock() bool {
	locked, err := s.locker.Lock(s.ctx, s.name, s.instanceID, s.schedInterval)
	if err != nil {
		level.Error(s.logger).Log("schedule", s.name, "err", err)
		sentry.CaptureException(err)
		return false
	}
	if locked {
		return true
	}
	level.Debug(s.logger).Log(s.name, "not the lock leader... Skipping...")
	return false
}
