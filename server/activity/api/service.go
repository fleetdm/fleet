package api

// Service is the composite interface for the activity bounded context.
// It embeds all method-specific interfaces. Bootstrap returns this type.
type Service interface {
	ListActivitiesService
}
