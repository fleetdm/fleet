// Package api provides the public API for the activity bounded context.
// External code should use this package to interact with activities.
package api

// Service is the composite interface for the activity bounded context.
// It embeds all method-specific interfaces. Bootstrap returns this type.
type Service interface {
	ListActivitiesService
	ListHostPastActivitiesService
	StreamActivitiesService
	NewActivityService
}
