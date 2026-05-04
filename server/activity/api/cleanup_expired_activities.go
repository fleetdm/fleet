package api

import "context"

// CleanupExpiredActivitiesService cleans up expired activities.
type CleanupExpiredActivitiesService interface {
	// CleanupExpiredActivities deletes up to maxCount activities older than expiryWindowDays
	// that are not linked to any host. Host-linked activities are preserved
	// (they are cleaned up when the host activity is processed).
	CleanupExpiredActivities(ctx context.Context, maxCount int, expiryWindowDays int) error
}
