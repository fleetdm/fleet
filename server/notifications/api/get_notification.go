package api

import "context"

// GetNotificationService looks up a notification on behalf of the device it
// belongs to.
type GetNotificationService interface {
	// GetNotificationForHost returns the notification, or a not-found error if
	// it doesn't exist or belongs to a different host than hostID.
	GetNotificationForHost(ctx context.Context, hostID uint, notificationUUID string) (*EndUserNotification, error)
}
