package api

import "context"

// CleanupHostActivitiesService cleans up activity_host_past rows when hosts are deleted.
type CleanupHostActivitiesService interface {
	// CleanupHostActivities removes activity_host_past rows for the given host IDs.
	CleanupHostActivities(ctx context.Context, hostIDs []uint) error
}
