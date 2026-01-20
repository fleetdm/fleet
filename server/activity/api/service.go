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

// Service is the composite interface for the activity bounded context.
// It embeds all method-specific interfaces. Bootstrap returns this type.
type Service interface {
	ListActivitiesService
	ListHostPastActivitiesService
	MarkActivitiesAsStreamedService
	StreamActivitiesService
}

// ListHostPastActivitiesService lists past activities for a specific host.
type ListHostPastActivitiesService interface {
	ListHostPastActivities(ctx context.Context, hostID uint, opt ListOptions) ([]*Activity, *PaginationMetadata, error)
}

// MarkActivitiesAsStreamedService marks activities as streamed.
type MarkActivitiesAsStreamedService interface {
	MarkActivitiesAsStreamed(ctx context.Context, activityIDs []uint) error
}

// StreamActivitiesService streams activities to an audit logger.
type StreamActivitiesService interface {
	// StreamActivities streams unstreamed activities to the provided audit logger.
	// The systemCtx should be a context with system-level authorization (no user context).
	// batchSize controls how many activities are fetched per batch.
	StreamActivities(systemCtx context.Context, auditLogger JSONLogger, batchSize uint) error
}
