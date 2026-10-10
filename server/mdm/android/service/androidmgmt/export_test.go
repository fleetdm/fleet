package androidmgmt

import (
	"log/slog"
	"math"
	"time"
)

// unlimitedRetryBudget lets tests that aren't about the budget retry with any delay.
const unlimitedRetryBudget time.Duration = math.MaxInt64

// NewRetryClientWithDelays lets external tests use short retry delays, with no retry budget.
func NewRetryClientWithDelays(client Client, delays []time.Duration) Client {
	return NewRetryClientWithBudget(client, delays, unlimitedRetryBudget)
}

// NewRetryClientWithBudget lets external tests use short retry delays and a short retry budget.
func NewRetryClientWithBudget(client Client, delays []time.Duration, budget time.Duration) Client {
	return newRetryClient(client, slog.New(slog.DiscardHandler), delays, budget)
}
