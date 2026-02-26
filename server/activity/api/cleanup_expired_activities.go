package api

import (
	"context"
	"time"
)

// CleanupExpiredActivitiesService cleans up expired activities.
type CleanupExpiredActivitiesService interface {
	// CleanupExpiredActivities deletes up to maxCount expired activities
	// that were created before expiryThreshold and are not linked to any host.
	// Host-linked activities are preserved (they are cleaned up when the host activity is processed).
	CleanupExpiredActivities(ctx context.Context, maxCount int, expiryThreshold time.Time) error
}
