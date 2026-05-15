package api

import "context"

// ListHostPastActivitiesService lists past activities for a specific host.
type ListHostPastActivitiesService interface {
	ListHostPastActivities(ctx context.Context, hostID uint, opt ListOptions) ([]*Activity, *PaginationMetadata, error)
}

// ListHostPastActivitiesForDeviceService lists past activities for a host in a
// context where access to the host has already been established by the caller
// (e.g. via a device authentication token). Implementations skip user-mode
// authorization checks.
type ListHostPastActivitiesForDeviceService interface {
	ListHostPastActivitiesForDevice(ctx context.Context, hostID uint, opt ListOptions) ([]*Activity, *PaginationMetadata, error)
}
