package api

import "context"

// ListHostPastActivitiesService lists past activities for a specific host.
type ListHostPastActivitiesService interface {
	// ListHostPastActivities lists the past activities of the host named by a host ID, hardware serial, UUID, or hostname.
	ListHostPastActivities(ctx context.Context, hostIdentifier string, opt ListOptions) ([]*Activity, *PaginationMetadata, error)
}
