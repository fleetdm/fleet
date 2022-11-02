// Package retry has utilities to retry operations.
package retry

import (
	"time"
)

type config struct {
	interval    time.Duration
	maxAttempts int
}

// Option allows to configure the behavior of retry.Do
type Option func(*config)

// WithRetryInterval allows to specify a custom duration to wait
// between retries.
func WithInterval(i time.Duration) Option {
	return func(c *config) {
		c.interval = i
	}
}

// WithMaxAttempts allows to specify a maximum number of attempts
// before the doer gives up.
func WithMaxAttempts(a int) Option {
	return func(c *config) {
		c.maxAttempts = a
	}
}

// Do executes the provided function, if the function returns a
// non-nil error it performs a retry according to the options
// provided.
//
// By default operations are retried an unlimited number of times for 30
// seconds
func Do(fn func() error, opts ...Option) error {
	cfg := &config{
		interval: 30 * time.Second,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	attempts := 0
	ticker := time.NewTicker(cfg.interval)
	defer ticker.Stop()

	for {
		attempts++
		err := fn()
		if err == nil {
			return nil
		}

		if cfg.maxAttempts != 0 && attempts >= cfg.maxAttempts {
			return err
		}

		<-ticker.C
	}
}
