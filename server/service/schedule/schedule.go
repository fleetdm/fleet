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

type schedule struct {
	ctx        context.Context
	name       string
	instanceID string
	Logger     kitlog.Logger

	muChecks       sync.Mutex
	configInterval time.Duration
	schedInterval  time.Duration
	locker         Locker
	configCheck    func(start time.Time, wait time.Duration) (*time.Duration, error)
	preflightCheck func() bool

	muJobs sync.Mutex
	jobs   map[string]job
}

func (s *schedule) getConfigInterval() time.Duration {
	s.muChecks.Lock()
	defer s.muChecks.Unlock()
	return s.configInterval
}

func (s *schedule) setConfigInterval(interval time.Duration) {
	s.muChecks.Lock()
	defer s.muChecks.Unlock()
	s.configInterval = interval
}

func (s *schedule) getSchedInterval() time.Duration {
	s.muChecks.Lock()
	defer s.muChecks.Unlock()
	return s.schedInterval
}

func (s *schedule) setSchedInterval(interval time.Duration) {
	s.muChecks.Lock()
	defer s.muChecks.Unlock()
	s.schedInterval = interval
}

func (s *schedule) SetPreflightCheck(fn func() bool) {
	s.muChecks.Lock()
	defer s.muChecks.Unlock()
	s.preflightCheck = fn
}

func (s *schedule) SetConfigCheck(fn func(start time.Time, wait time.Duration) (*time.Duration, error)) {
	s.muChecks.Lock()
	defer s.muChecks.Unlock()
	s.configCheck = fn
}

type job struct {
	run          func(context.Context) (interface{}, error)
	statsHandler func(interface{}, error)
}

type Locker interface {
	Lock(ctx context.Context, name string, owner string, expiration time.Duration) (bool, error)
	Unlock(ctx context.Context, name string, owner string) error
}

func New(ctx context.Context, name string, instanceID string, interval time.Duration, locker Locker, logger kitlog.Logger) (sched *schedule, err error) {
	sch := &schedule{
		ctx:        ctx,
		name:       name,
		instanceID: instanceID,
		Logger:     logger,

		configInterval: 1 * time.Hour, // by default we will check for updated config once per hour
		schedInterval:  interval,
		locker:         locker,

		jobs: make(map[string]job),
	}
	sch.run()
	return sch, nil
}

func (s *schedule) AddJob(id string, newJob func(ctx context.Context) (interface{}, error), statsHandler func(interface{}, error)) {
	s.muJobs.Lock()
	defer s.muJobs.Unlock()
	// TODO: guard for job id uniqueness?
	s.jobs[id] = job{run: newJob, statsHandler: statsHandler}
}

// each schedule runs in its own go routine
func (s *schedule) run() {
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
			level.Debug(s.Logger).Log("waiting", fmt.Sprint("current wait time... ", currWait))

			select {
			case <-s.ctx.Done():
				level.Debug(s.Logger).Log("exit", fmt.Sprint("done with ", s.name))
				return

			case <-schedTicker.C:
				level.Debug(s.Logger).Log("waiting", "done")

				schedInterval := s.getSchedInterval()
				schedTicker.Reset(schedInterval)

				newStart := time.Now()
				newWait := schedInterval
				setWaitTimes(newStart, newWait)

				if ok := s.runScheduleChecks(); !ok {
					continue
				}

				s.muJobs.Lock()
				for id, job := range s.jobs {
					level.Debug(s.Logger).Log(s.name, fmt.Sprint("starting job... ", id))
					job.statsHandler(job.run(s.ctx))
				}
				s.muJobs.Unlock()
			}
		}
	}()

	// this periodically checks for config updates and resets the schedInterval for the main loop
	configTicker := time.NewTicker(200 * time.Millisecond)
	go func() {
		for {
			select {
			case <-configTicker.C:
				level.Debug(s.Logger).Log(s.name, "config check...")

				configInterval := s.getConfigInterval()
				configTicker.Reset(configInterval)

				schedInterval := s.getSchedInterval()
				currStart, currWait := getWaitTimes()

				if s.configCheck == nil {
					level.Debug(s.Logger).Log(s.name, "config check function has not been set... skipping...")
					continue
				}

				newInterval, err := s.configCheck(currStart, currWait)

				if err != nil {
					level.Error(s.Logger).Log("config", "could not check for updates to schedule interval", "err", err)
					sentry.CaptureException(err)
					continue
				}
				if schedInterval == *newInterval {
					level.Debug(s.Logger).Log(s.name, "schedule interval unchanged")
					continue
				}
				s.setSchedInterval(*newInterval)

				newWait := 10 * time.Millisecond
				if time.Since(currStart) < *newInterval {
					newWait = *newInterval - time.Since(currStart)
				}
				setWaitTimes(currStart, newWait)
				schedTicker.Reset(newWait)

				level.Debug(s.Logger).Log(s.name, fmt.Sprint("new schedule interval: ", *newInterval))
				level.Debug(s.Logger).Log(s.name, fmt.Sprint("wait time until next job run: ", newWait))
			}
		}
	}()
}

func (s *schedule) runScheduleChecks() bool {
	s.muChecks.Lock()
	defer s.muChecks.Unlock()
	if s.preflightCheck != nil {
		if ok := s.preflightCheck(); !ok {
			level.Debug(s.Logger).Log(s.name, "preflight check failed... skipping...")
			return false
		}
	}
	if locked, err := s.locker.Lock(s.ctx, s.name, s.instanceID, s.schedInterval); err != nil || !locked {
		level.Debug(s.Logger).Log(s.name, "not the lock leader... Skipping...")
		return false
	}
	return true
}
