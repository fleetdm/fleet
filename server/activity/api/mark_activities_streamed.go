package api

import "context"

// MarkActivitiesAsStreamedService marks activities as streamed.
type MarkActivitiesAsStreamedService interface {
	MarkActivitiesAsStreamed(ctx context.Context, activityIDs []uint) error
}
