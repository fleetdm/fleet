package api

import "context"

// RecordOutcomeService records how a queued script attempt ended, when that
// execution belongs to a notification. server/service calls this when it
// saves a script result; most script results have nothing to do with a
// notification, which RecordOutcome treats as a no-op rather than an error.
type RecordOutcomeService interface {
	RecordOutcome(ctx context.Context, executionID string, exitCode int64, output string) error
}
