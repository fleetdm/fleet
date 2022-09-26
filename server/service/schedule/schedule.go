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

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/getsentry/sentry-go"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
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
	logger     log.Logger

	schedIntervalMu sync.Mutex // protects schedInterval.
	schedInterval   time.Duration

	done chan struct{}

	configReloadInterval   time.Duration
	configReloadIntervalFn ReloadInterval

	locker Locker

	altLockName string

	jobs []Job
}

// JobFn is the signature of a Job.
type JobFn func(context.Context) error

// Job represents a job that can be added to Scheduler.
type Job struct {
	// ID is the unique identifier for the job.
	ID string
	// Fn is the job itself.
	Fn JobFn
}

// Locker allows a Schedule to acquire a lock before running jobs.
type Locker interface {
	Lock(ctx context.Context, scheduleName string, scheduleInstanceID string, expiration time.Duration) (bool, error)
	Unlock(ctx context.Context, scheduleName string, scheduleInstanceID string) error
}

// Option allows configuring a Schedule.
type Option func(*Schedule)

// WithLogger sets a logger for the Schedule.
func WithLogger(l log.Logger) Option {
	return func(s *Schedule) {
		s.logger = log.With(l, "schedule", s.name)
	}
}

// WithConfigReloadInterval allows setting a reload interval function,
// that will allow updating the interval of a running schedule.
//
// If not set, then the schedule performs no interval reloading.
func WithConfigReloadInterval(interval time.Duration, fn ReloadInterval) Option {
	return func(s *Schedule) {
		s.configReloadInterval = interval
		s.configReloadIntervalFn = fn
	}
}

// WithAltLockID sets an alternative identifier to use when acquiring the lock.
//
// If not set, then the Schedule's name is used for acquiring the lock.
func WithAltLockID(name string) Option {
	return func(s *Schedule) {
		s.altLockName = name
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
		ctx:                  ctx,
		name:                 name,
		instanceID:           instanceID,
		logger:               log.NewNopLogger(),
		done:                 make(chan struct{}),
		configReloadInterval: 1 * time.Hour, // by default we will check for updated config once per hour
		schedInterval:        interval,
		locker:               locker,
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
	var m sync.Mutex // protects currentStart and currentWait.
	currentStart := time.Now()
	currentWait := 10 * time.Second

	getWaitTimes := func() (start time.Time, wait time.Duration) {
		m.Lock()
		defer m.Unlock()

		return currentStart, currentWait
	}

	setWaitTimes := func(start time.Time, wait time.Duration) {
		m.Lock()
		defer m.Unlock()

		currentStart = start
		currentWait = wait
	}

	if schedInterval := s.getSchedInterval(); schedInterval < currentWait {
		setWaitTimes(currentStart, schedInterval)
	}

	var g sync.WaitGroup

	schedTicker := time.NewTicker(currentWait)
	g.Add(+1)
	go func() {
		defer g.Done()

		for {
			_, currWait := getWaitTimes()
			level.Debug(s.logger).Log("msg", "waiting", "current wait time", currWait)

			select {
			case <-s.ctx.Done():
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
					level.Debug(s.logger).Log("msg", "starting", "jobID", job.ID)
					if err := runJob(s.ctx, job.Fn); err != nil {
						level.Error(s.logger).Log("err", job.ID, "details", err)
						sentry.CaptureException(err)
						ctxerr.Handle(s.ctx, err)
					}
				}
			}
		}
	}()

	// Periodically check for config updates and resets the schedInterval for the previous loop.
	g.Add(+1)
	go func() {
		defer g.Done()
		configTicker := time.NewTicker(200 * time.Millisecond)

		for {
			select {
			case <-s.ctx.Done():
				level.Info(s.logger).Log("msg", "done")
				return
			case <-configTicker.C:
				level.Debug(s.logger).Log("msg", "config reload check")

				configTicker.Reset(s.configReloadInterval)

				schedInterval := s.getSchedInterval()
				currStart, _ := getWaitTimes()

				if s.configReloadIntervalFn == nil {
					level.Debug(s.logger).Log("msg", "config reload interval method not set")
					continue
				}

				newInterval, err := s.configReloadIntervalFn(s.ctx)
				if err != nil {
					level.Error(s.logger).Log("msg", "schedule interval config reload failed", "err", err)
					sentry.CaptureException(err)
					continue
				}
				if newInterval <= 0 {
					level.Debug(s.logger).Log("msg", "config reload interval method returned invalid interval")
					continue
				}
				if schedInterval == newInterval {
					level.Debug(s.logger).Log("msg", "schedule interval unchanged")
					continue
				}
				s.setSchedInterval(newInterval)

				newWait := 10 * time.Millisecond
				if time.Since(currStart) < newInterval {
					newWait = newInterval - time.Since(currStart)
				}
				setWaitTimes(currStart, newWait)
				schedTicker.Reset(newWait)

				level.Debug(s.logger).Log("new schedule interval", newInterval, "new wait", newWait)
			}
		}
	}()

	go func() {
		g.Wait()
		level.Debug(s.logger).Log("msg", "done")
		close(s.done) // communicates that the scheduler has finished running its goroutines
	}()
}

// runJob executes the job function with panic recovery
func runJob(ctx context.Context, fn JobFn) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v", r)
		}
	}()

	if err := fn(ctx); err != nil {
		return err
	}
	return nil
}

// Done returns a channel that will be closed when the scheduler's context is done
// and it has finished running its goroutines.
func (s *Schedule) Done() <-chan struct{} {
	return s.done
}

func (s *Schedule) getSchedInterval() time.Duration {
	s.schedIntervalMu.Lock()
	defer s.schedIntervalMu.Unlock()

	return s.schedInterval
}

func (s *Schedule) setSchedInterval(interval time.Duration) {
	s.schedIntervalMu.Lock()
	defer s.schedIntervalMu.Unlock()

	s.schedInterval = interval
}

func (s *Schedule) acquireLock() bool {
	name := s.name
	if s.altLockName != "" {
		name = s.altLockName
	}
	locked, err := s.locker.Lock(s.ctx, name, s.instanceID, s.getSchedInterval())
	if err != nil {
		level.Error(s.logger).Log("msg", "lock failed", "err", err)
		sentry.CaptureException(err)
		return false
	}
	if locked {
		return true
	}
	level.Debug(s.logger).Log("msg", "not the lock leader, skipping")
	return false
}
