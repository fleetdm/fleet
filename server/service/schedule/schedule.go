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

	MuChecks       sync.Mutex
	configInterval time.Duration
	schedInterval  time.Duration
	locker         Locker
	configCheck    func(start time.Time, wait time.Duration) (*time.Duration, error)
	preflightCheck func() bool

	muJobs sync.Mutex
	jobs   map[string]job
}

func (s *schedule) getConfigInterval() time.Duration {
	s.MuChecks.Lock()
	defer s.MuChecks.Unlock()
	return s.configInterval
}

func (s *schedule) setConfigInterval(interval time.Duration) {
	s.MuChecks.Lock()
	defer s.MuChecks.Unlock()
	s.configInterval = interval
}

func (s *schedule) getSchedInterval() time.Duration {
	s.MuChecks.Lock()
	defer s.MuChecks.Unlock()
	return s.schedInterval
}

func (s *schedule) setSchedInterval(interval time.Duration) {
	s.MuChecks.Lock()
	defer s.MuChecks.Unlock()
	s.schedInterval = interval
}

func (s *schedule) SetPreflightCheck(fn func() bool) {
	s.MuChecks.Lock()
	defer s.MuChecks.Unlock()
	s.preflightCheck = fn
}

func (s *schedule) SetConfigCheck(fn func(start time.Time, wait time.Duration) (*time.Duration, error)) {
	s.MuChecks.Lock()
	defer s.MuChecks.Unlock()
	s.configCheck = fn
}

type job struct {
	exec         func(context.Context) (interface{}, error)
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
	s.jobs[id] = job{exec: newJob, statsHandler: statsHandler}
}

// each schedule runs in its own go routine
func (s *schedule) run() {
	currentStart := time.Now()
	currentWait := 10 * time.Second

	getWaitTimes := func() (start time.Time, wait time.Duration) {
		s.MuChecks.Lock()
		defer s.MuChecks.Unlock()
		return currentStart, currentWait
	}

	setWaitTimes := func(start time.Time, wait time.Duration) {
		s.MuChecks.Lock()
		defer s.MuChecks.Unlock()
		currentStart = start
		currentWait = wait
	}

	if i := s.getSchedInterval(); i < currentWait {
		setWaitTimes(currentStart, i)
	}

	// this is the main loop for the schedule
	schedTicker := time.NewTicker(currentWait)
	go func() {
		// step := 1
		for {
			// fmt.Println(s.name, " loop ", step)
			// step++
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

				s.MuChecks.Lock() // TODO: talk with Tomas about this for locker and preflightFn setter
				if s.preflightCheck != nil {
					if ok := s.preflightCheck(); !ok {
						level.Debug(s.Logger).Log(s.name, "preflight check failed... skipping...")
						s.MuChecks.Unlock()
						continue
					}
				}
				if locked, err := s.locker.Lock(s.ctx, s.name, s.instanceID, schedInterval); err != nil || !locked {
					level.Debug(s.Logger).Log(s.name, "not the lock leader... Skipping...")
					s.MuChecks.Unlock()
					continue
				}
				s.MuChecks.Unlock()

				s.muJobs.Lock()
				for id, job := range s.jobs {
					level.Debug(s.Logger).Log(s.name, fmt.Sprint("starting job... ", id))
					job.statsHandler(job.exec(s.ctx)) // TODO: start new go routine for each job?
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
			// TODO: What if we simply lock MuChecks for the duration of this case?
			case <-configTicker.C:
				configInterval := s.getConfigInterval()
				configTicker.Reset(configInterval)

				schedInterval := s.getSchedInterval()
				currStart, currWait := getWaitTimes()

				fmt.Println("config check")
				level.Debug(s.Logger).Log(s.name, "config check...")
				if s.configCheck == nil {
					continue
				}

				newSchedInterval, err := s.configCheck(currStart, currWait)
				if err != nil {
					level.Error(s.Logger).Log("config", "could not check for updates to schedInterval", "err", err)
					sentry.CaptureException(err)
					continue
				}
				if schedInterval == *newSchedInterval {
					level.Debug(s.Logger).Log(s.name, "schedInterval unchanged")
					continue
				}
				s.setSchedInterval(*newSchedInterval)

				newWait := 10 * time.Millisecond
				if time.Since(currStart) < *newSchedInterval {
					newWait = *newSchedInterval - time.Since(currStart)
				}
				setWaitTimes(currStart, newWait)
				schedTicker.Reset(newWait)

				level.Debug(s.Logger).Log(s.name, fmt.Sprint("new schedInterval: ", *newSchedInterval))
				level.Debug(s.Logger).Log(s.name, fmt.Sprint("new wait: ", newWait))
			}
		}
	}()
}
