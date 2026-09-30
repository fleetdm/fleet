package androidmgmt

import (
	"log/slog"
	"time"
)

// NewRetryClientWithDelays lets external tests use short retry delays.
func NewRetryClientWithDelays(client Client, delays []time.Duration) Client {
	return newRetryClient(client, slog.New(slog.DiscardHandler), delays)
}
