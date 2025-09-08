package limit

import (
	"context"
	"sync/atomic"
	"time"
)

// limiter is a super simple leaky-bucket Goroutine rate limiter.
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

func (l *limiter) Go(fn func()) {
	l.jobsQueued.Add(1)
	go func() {
		l.waitOne()
		l.jobsQueued.Add(-1)
		l.jobsConcurrent.Add(1)
		fn()
		l.jobsConcurrent.Add(-1)
	}()
}

func (l *limiter) Wait() {
	for l.jobsConcurrent.Load() > 0 {
		<-time.After(time.Millisecond)
	}
	for l.jobsQueued.Load() > 0 {
		<-time.After(time.Millisecond)
	}
}

func (l *limiter) WaitContext(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return context.DeadlineExceeded
		case <-time.After(time.Millisecond):
			if l.jobsConcurrent.Load() > 0 {
				continue
			}
			if l.jobsQueued.Load() > 0 {
				continue
			}
			return nil
		}
	}
}

func (l *limiter) waitOne() {
	for l.jobsConcurrent.Load() >= l.jobsConcurrentMax {
		<-time.After(time.Millisecond)
	}
}
