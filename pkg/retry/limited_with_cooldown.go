package retry

import (
	"fmt"
	"sync"
	"time"
)

// ExcessRetriesError is returned when the number of retries for a specific
// hash exceeds the maximum allowed limit and the cooldown period has not yet
// elapsed.
//
// It indicates that the function associated with the hash will not
// be retried until the cooldown period is over.
type ExcessRetriesError struct {
	nextRetry time.Duration
}

func (e *ExcessRetriesError) Error() string {
	return fmt.Sprintf("max retries exceeded, retrying again in %v", e.nextRetry)
}

// LimitedWithCooldown manages function calls with a limit on retries and a
// cooldown period.
// It tracks retries and wait times for each unique hash key.
//
// NOTE: we also have a backoff package widely used in the repo
// (github.com/cenkalti/backoff/v4), but I'm implementing a custom struct due to:
//   - the very specific product requirements (retry n consecutive times per
//     hash, then wait for a defined cooldown.)
//   - this retry mechanism is used by the `fleetd` updater and needs to be
//     thread safe.
type LimitedWithCooldown struct {
	maxRetries int                  // maxRetries is the maximum number of retries allowed before entering cooldown.
	cooldown   time.Duration        // cooldown is the duration to wait before resetting the retry count.
	retries    map[string]int       // retries tracks the number of retries for each hash key.
	wait       map[string]time.Time // wait tracks the start of the cooldown period for each hash key.
	mu         sync.Mutex           // mu is used to ensure thread safety.
}

// NewLimitedWithCooldown creates a new instance of LimitedWithCooldown.
//   - maxRetries specifies the maximum number of retries allowed for a function call.
//   - cooldown specifies the duration to wait before allowing retries again.
func NewLimitedWithCooldown(maxRetries int, cooldown time.Duration) *LimitedWithCooldown {
	return &LimitedWithCooldown{
		maxRetries: maxRetries,
		cooldown:   cooldown,
		retries:    map[string]int{},
		wait:       map[string]time.Time{},
	}
}

// Do executes the provided function fn associated with the given hash.
// It applies the retry and cooldown logic based on previous attempts with the same hash.
//   - hash is a unique identifier for the function call context.
//   - fn is the function to be executed; it must return an error to indicate success or failure.
//
// If the retries exceed maxRetries and the cooldown period has not passed, ErrRetriesExceeded is returned.
// If fn executes successfully, the retry count and wait time for the hash are reset.
// Returns an error if fn fails and has not exceeded the max retries or cooldown period.
func (t *LimitedWithCooldown) Do(hash string, fn func() error) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.retries[hash] >= t.maxRetries &&
		time.Since(t.wait[hash]) <= t.cooldown {
		return &ExcessRetriesError{nextRetry: time.Until(t.wait[hash].Add(t.cooldown))}
	}

	if err := fn(); err != nil {
		t.retries[hash]++
		if t.retries[hash] >= t.maxRetries {
			t.wait[hash] = time.Now()
		}
		return err
	}

	t.retries[hash] = 0
	t.wait[hash] = time.Time{}
	return nil
}
