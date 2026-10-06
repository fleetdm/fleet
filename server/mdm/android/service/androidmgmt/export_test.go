package androidmgmt

import (
	"log/slog"
	"math"
	"time"
)

// NewRetryClientWithDelays lets external tests use short retry delays, with no retry budget.
func NewRetryClientWithDelays(client Client, delays []time.Duration) Client {
	return NewRetryClientWithBudget(client, delays, math.MaxInt64)
}

// NewRetryClientWithBudget lets external tests use short retry delays and a short retry budget.
func NewRetryClientWithBudget(client Client, delays []time.Duration, budget time.Duration) Client {
	return newRetryClient(client, slog.New(slog.DiscardHandler), delays, budget)
}
