package backoff

import (
	"fmt"
	"time"
)

type Opt func(*Backoff)

func MaxAttempts(i int) Opt {
	return func(b *Backoff) {
		b.maxAttempt = i
	}
}

// Backoff is a quick retry function
type Backoff struct {
	count      int
	maxAttempt int
	delay      float32
	runFunc    func() error
}

// New returns a Backoff timer
func New(opts ...Opt) *Backoff {
	b := &Backoff{
		count:      0,
		delay:      1,
		maxAttempt: 20,
	}

	for _, opt := range opts {
		opt(b)
	}

	return b
}

// Run trys to run function several times until it succeeds or times out.
func (b *Backoff) Run(runFunc func() error) error {
	b.runFunc = runFunc
	return b.try()
}

func (b *Backoff) try() error {
	b.count += 1
	err := b.runFunc()
	if err != nil {
		if b.count >= b.maxAttempt {
			return fmt.Errorf("done trying: %w", err)
		}

		// Wait for amount of time
		timer := time.NewTimer(time.Second * time.Duration(b.delay))
		<-timer.C

		return b.try()
	}

	// err == nil, SUCCESS!
	return nil
}
