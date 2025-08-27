package limit

import (
	"context"
	"sync/atomic"
	"time"
)

// limiter is a super simply leaky-bucket Goroutine rate limiter.
//
// limiter _must_ be instantiated via the constructor ('New'), this is why the
// type itself is not exported.
type limiter struct {
	jobsQueued        atomic.Int32
	jobsConcurrent    atomic.Int32
	jobsConcurrentMax int32
	_                 noCopy
}

func New(maxConcurrentJobs int32) *limiter {
	return &limiter{
		jobsConcurrentMax: maxConcurrentJobs,
	}
}

func (self *limiter) Go(fn func()) {
	self.jobsQueued.Add(1)
	go func() {
		self.waitOne()
		self.jobsQueued.Add(-1)
		self.jobsConcurrent.Add(1)
		fn()
		self.jobsConcurrent.Add(-1)
	}()
}

func (self *limiter) Wait() {
	for self.jobsConcurrent.Load() > 0 {
		<-time.After(time.Millisecond)
	}
	for self.jobsQueued.Load() > 0 {
		<-time.After(time.Millisecond)
	}
}

func (self *limiter) WaitContext(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return context.DeadlineExceeded
		case <-time.After(time.Millisecond):
			if self.jobsConcurrent.Load() > 0 {
				continue
			}
			if self.jobsQueued.Load() > 0 {
				continue
			}
			return nil
		}
	}
}

func (self *limiter) waitOne() {
	for self.jobsConcurrent.Load() >= self.jobsConcurrentMax {
		<-time.After(time.Millisecond)
	}
}
