package backoff

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/time/rate"
)

// waitFor is a wrapper over the stock go limiter pattern. It can be
// used to retry things until the timeout is reached, with a delay in
// between attempts. It does not implement an exponential backoff, and
// is intended for simple startup logic.
//
// Contrary to documentation, it appears go will always retry
// once. Even if the interval exceeds the timeout.
func WaitFor(fn func() error, timeout, interval time.Duration) error {
	deadlineCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	limiter := rate.NewLimiter(rate.Every(interval), 1)
	counter := 0
	for {
		counter += 1

		err := fn()

		if err == nil {
			return nil
		}

		// Did we timeout? If so, send the error
		if limiter.Wait(deadlineCtx) != nil {
			return fmt.Errorf("timeout after %s (%d attempts): %w", timeout, counter, err)
		}
	}
}
