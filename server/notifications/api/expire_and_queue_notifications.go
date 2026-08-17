package api

import "context"

// ExpireAndQueueNotificationsService is the one pass the cron makes over the
// notification queue.
type ExpireAndQueueNotificationsService interface {
	// ExpireAndQueueNotifications gives up on notifications that are out of
	// time, then queues a script for each one that is due.
	ExpireAndQueueNotifications(ctx context.Context) error
}
