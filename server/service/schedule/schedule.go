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
	preflightCheck func() bool

	mu   sync.Mutex
	jobs map[string]job
}

func (s *schedule) SetPreflightCheckFunc(fn func() bool) {
	s.preflightCheck = fn
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
	go func() {
		step := 1
		currentWait := 10 * time.Second
		if currentWait > s.interval {
			currentWait = s.interval
		}
		for {
			fmt.Println(s.name, "loop", step)
			select {
			case <-s.ctx.Done():
				return
			case <-time.Tick(currentWait):
				currentWait = s.interval
				if s.preflightCheck != nil {
					if ok := s.preflightCheck(); !ok {
						level.Debug(s.Logger).Log(s.name, "Preflight check failed. Skipping...")
						step++
						continue
					}
				}

				if locked, err := s.locker.Lock(s.ctx, s.name, s.instanceID, s.interval); err != nil || !locked {
					level.Debug(s.Logger).Log(s.name, "Not the lock leader. Skipping...")
					step++
					continue
				}

				s.mu.Lock()
				for id, job := range s.jobs {
					fmt.Println("starting job... ", id)
					job.statsHandler(job.exec(s.ctx)) // start new go routine for each job?
				}
				s.mu.Unlock()
			}
			step++
		}
	}()
}
