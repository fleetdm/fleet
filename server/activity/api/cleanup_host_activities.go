package api

import "context"

// CleanupHostActivitiesService cleans up host_activities rows when hosts are deleted.
type CleanupHostActivitiesService interface {
	// CleanupHostActivities removes host_activities rows for the given host IDs.
	CleanupHostActivities(ctx context.Context, hostIDs []uint) error
}
