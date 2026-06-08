package api

import "context"

// ListHostPastActivitiesService lists past activities for a specific host.
type ListHostPastActivitiesService interface {
	ListHostPastActivities(ctx context.Context, hostID uint, opt ListOptions) ([]*Activity, *PaginationMetadata, error)
}
