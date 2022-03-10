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
	ctx            context.Context
	name           string
	instanceID     string
	interval       time.Duration
	locker         Locker
	Logger         kitlog.Logger
	configCheck    func(start time.Time, wait time.Duration) (*time.Duration, error)
	preflightCheck func() bool

	mu   sync.Mutex
	jobs map[string]job
}

func (s *schedule) SetPreflightCheck(fn func() bool) {
	s.preflightCheck = fn
}

func (s *schedule) SetConfigCheck(fn func(start time.Time, wait time.Duration) (*time.Duration, error)) {
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
	s.mu.Lock()
	defer s.mu.Unlock()
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

	// this is the main loop for the schedule
	go func(start *time.Time, wait *time.Duration) {
		step := 1
		for {
			fmt.Println(s.name, " loop ", step)
			fmt.Println("start: ", *start, "wait: ", *wait)
			level.Debug(s.Logger).Log("waiting", fmt.Sprint("on ticker..."))
			select {
			case <-s.ctx.Done():
				level.Debug(s.Logger).Log("exit", fmt.Sprint("done with ", s.name))
				return

			case <-schedTicker.C:
				level.Debug(s.Logger).Log("waiting", "done")
				newStart := time.Now()

				if s.preflightCheck != nil {
					if ok := s.preflightCheck(); !ok {
						level.Debug(s.Logger).Log(s.name, "preflight check failed... skipping...")
						schedTicker.Reset(s.interval) // TODO: confirm we want to the next interval to run from completion of the jobs (not before)
						start = &newStart
						wait = &s.interval
						step++
						continue
					}
				}
				if locked, err := s.locker.Lock(s.ctx, s.name, s.instanceID, s.interval); err != nil || !locked {
					level.Debug(s.Logger).Log(s.name, "not the lock leader... Skipping...")
					schedTicker.Reset(s.interval) // TODO: confirm we want to the next interval to run from completion of the jobs (not before)
					start = &newStart
					wait = &s.interval
					step++
					continue
				}

				s.mu.Lock()
				for id, job := range s.jobs {
					fmt.Println("starting job... ", id)
					job.statsHandler(job.exec(s.ctx)) // start new go routine for each job?
				}
				s.mu.Unlock()

				schedTicker.Reset(s.interval) // TODO: confirm we want to the next interval to run from completion of the jobs (not before)
				start = &newStart
				wait = &s.interval
				step++
			}
		}
	}(&currentStart, &currentWait)

	// this periodically checks for config updates and resets the interval for the main loop
	go func(start *time.Time, wait *time.Duration) {
		w := 20 * time.Second
		configTicker := time.NewTicker(w)
		if w > s.interval {
			w = s.interval
			configTicker.Reset(w)
		}
		for {
			select {
			case <-configTicker.C:
				fmt.Println("config check")
				level.Debug(s.Logger).Log(s.name, "config check...")
				if s.configCheck != nil {
					newInterval, err := s.configCheck(*start, *wait)
					fmt.Println(newInterval, s.interval)
					if err != nil {
						level.Error(s.Logger).Log("config", "could not check for updates to interval config", "err", err)
						// sentry.CaptureException(err)
					} else if *newInterval == s.interval {
						level.Debug(s.Logger).Log(s.name, "interval unchanged")
					} else if *newInterval < time.Since(*start) {
						s.interval = *newInterval
						w = 10 * time.Millisecond
						wait = &w // TODO: How do we want to handle this?
						schedTicker.Reset(w)
						configTicker.Reset(w)
						level.Debug(s.Logger).Log(s.name, fmt.Sprint("new interval: ", *newInterval))
					} else {
						s.interval = *newInterval
						w = *newInterval - time.Since(*start)
						wait = &w
						schedTicker.Reset(w)
						configTicker.Reset(w)
						level.Debug(s.Logger).Log(s.name, fmt.Sprint("new interval: ", *newInterval))

						// start = time.Now()
					}
				}
			}
		}
	}(&currentStart, &currentWait)
}
