package schedule

import (
	"context"
	"fmt"
	"sync"
	"time"

	kitlog "github.com/go-kit/kit/log"
	"github.com/go-kit/log/level"
)

type schedule struct {
	ctx        context.Context
	name       string
	instanceID string
	interval   time.Duration
	locker     Locker
	Logger     kitlog.Logger

	muChecks       sync.Mutex
	configCheck    func(start time.Time, wait time.Duration) (*time.Duration, error)
	preflightCheck func() bool

	muJobs sync.Mutex
	jobs   map[string]job
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
		interval:   interval,
		locker:     locker,
		Logger:     logger,

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

func (s *schedule) run() {
	// each schedule runs in its own go routine
	currentStart := time.Now()
	currentWait := 10 * time.Second
	if currentWait > s.interval {
		currentWait = s.interval
	}
	schedTicker := time.NewTicker(currentWait)

	muTimes := sync.Mutex{}

	readTimes := func() (start time.Time, wait time.Duration, interval time.Duration) {
		muTimes.Lock()
		defer muTimes.Unlock()
		return currentStart, currentWait, s.interval
	}

	setTimes := func(start time.Time, wait time.Duration, interval time.Duration) {
		muTimes.Lock()
		defer muTimes.Unlock()
		currentStart = start
		currentWait = wait
		s.interval = interval
	}

	// this is the main loop for the schedule
	go func() {
		step := 1
		for {
			fmt.Println(s.name, " loop ", step)
			step++

			level.Debug(s.Logger).Log("waiting", fmt.Sprint("on ticker..."))

			select {
			case <-s.ctx.Done():
				level.Debug(s.Logger).Log("exit", fmt.Sprint("done with ", s.name))
				return

			case <-schedTicker.C:
				level.Debug(s.Logger).Log("waiting", "done")

				_, _, schedInterval := readTimes()
				newStart := time.Now()
				newWait := schedInterval

				schedTicker.Reset(schedInterval) // TODO: confirm we want to the next interval to run from completion of the jobs (not before)
				setTimes(newStart, newWait, schedInterval)

				if s.preflightCheck != nil {
					if ok := s.preflightCheck(); !ok {
						level.Debug(s.Logger).Log(s.name, "preflight check failed... skipping...")
						continue
					}
				}
				if locked, err := s.locker.Lock(s.ctx, s.name, s.instanceID, schedInterval); err != nil || !locked {
					level.Debug(s.Logger).Log(s.name, "not the lock leader... Skipping...")
					continue
				}

				s.muJobs.Lock()
				for id, job := range s.jobs {
					fmt.Println("starting job... ", id)
					job.statsHandler(job.exec(s.ctx)) // start new go routine for each job?
				}
				s.muJobs.Unlock()
			}
		}
	}()

	// this periodically checks for config updates and resets the interval for the main loop
	go func() {
		_, _, schedInterval := readTimes()
		w := 20 * time.Second
		configTicker := time.NewTicker(w)
		if w > schedInterval {
			w = schedInterval
			configTicker.Reset(w)
		}
		for {
			select {
			case <-configTicker.C:
				currStart, currWait, schedInterval := readTimes()

				fmt.Println("config check")
				level.Debug(s.Logger).Log(s.name, "config check...")
				if s.configCheck == nil {
					continue
				}

				newInterval, err := s.configCheck(currStart, currWait)
				fmt.Println(newInterval, s.interval)
				if err != nil {
					level.Error(s.Logger).Log("config", "could not check for updates to interval config", "err", err)
					// sentry.CaptureException(err)
					continue
				}

				if s.interval == *newInterval {
					level.Debug(s.Logger).Log(s.name, "interval unchanged")
					continue
				}

				// atomic.StoreInt64((*int64)(&s.interval)), int64(*newInterval))
				newWait := 10 * time.Millisecond

				if time.Since(currStart) < *newInterval {
					newWait = *newInterval - time.Since(currStart)
					// start = time.Now()
				}

				setTimes(currStart, newWait, *newInterval)
				schedTicker.Reset(newWait)
				configTicker.Reset(newWait)

				level.Debug(s.Logger).Log(s.name, fmt.Sprint("new interval: ", *newInterval))
			}
		}
	}()
}
