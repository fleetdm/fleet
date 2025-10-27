// Package retry has utilities to retry operations.
package retry

import (
	"time"
)

type ErrorOutcome int

const (
	ErrorOutcomeNormalRetry ErrorOutcome = iota
	ErrorOutcomeResetAttempts
	ErrorOutcomeIgnore
	ErrorOutcomeDoNotRetry
)

type config struct {
	initialInterval   time.Duration
	backoffMultiplier int
	maxAttempts       int
	errorFilter       func(error) ErrorOutcome
}

// Option allows to configure the behavior of retry.Do
type Option func(*config)

// WithInterval allows to specify a custom duration to wait
// between retries.
func WithInterval(i time.Duration) Option {
	return func(c *config) {
		c.initialInterval = i
	}
}

// WithBackoffMultiplier allows to specify the backoff multiplier between retries.
func WithBackoffMultiplier(m int) Option {
	return func(c *config) {
		c.backoffMultiplier = m
	}
}

// WithMaxAttempts allows to specify a maximum number of attempts
// before the doer gives up.
func WithMaxAttempts(a int) Option {
	return func(c *config) {
		c.maxAttempts = a
	}
}

func WithErrorFilter(f func(error) ErrorOutcome) Option {
	return func(c *config) {
		c.errorFilter = f
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
		initialInterval: 30 * time.Second,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	attempts := 0
	ticker := time.NewTicker(cfg.initialInterval)
	defer ticker.Stop()

	backoff := 1
	for {
		attempts++
		err := fn()
		if err == nil {
			return nil
		}
		if cfg.errorFilter != nil {
			outcome := cfg.errorFilter(err)
			switch outcome {
			case ErrorOutcomeIgnore:
				return nil
			case ErrorOutcomeResetAttempts:
				attempts = 0
				backoff = 1
			case ErrorOutcomeDoNotRetry:
				return err
			default:
				// continue with normal retry
			}
		}

		if cfg.maxAttempts != 0 && attempts >= cfg.maxAttempts {
			return err
		}

		if cfg.backoffMultiplier != 0 {
			interval := time.Duration(backoff) * cfg.initialInterval
			backoff *= cfg.backoffMultiplier
			ticker.Reset(interval)
		}

		<-ticker.C
	}
}
