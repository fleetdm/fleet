package api

import "context"

// ApplyActionService carries out what an end user's device reported about one
// of its notifications.
type ApplyActionService interface {
	// ApplyAction validates the device's report and carries it out. hostID
	// scopes the notification to the device that reported it.
	ApplyAction(ctx context.Context, hostID uint, notificationUUID string, action EndUserNotificationAction) error
}
