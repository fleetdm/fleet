package api

import (
	"context"
	"encoding/json"
)

// JSONLogger defines an interface for loggers that can write JSON to various
// output sources.
type JSONLogger interface {
	// Write writes the JSON log entries to the appropriate destination,
	// returning any errors that occurred.
	Write(ctx context.Context, logs []json.RawMessage) error
}

// StreamActivitiesService streams activities to an audit logger.
type StreamActivitiesService interface {
	// StreamActivities streams unstreamed activities to the provided audit logger.
	// The systemCtx should be a context with system-level authorization (no user context).
	StreamActivities(systemCtx context.Context, auditLogger JSONLogger) error
}
