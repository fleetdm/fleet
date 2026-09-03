package fleet

import (
	"github.com/fleetdm/fleet/v4/server/notifications/api"
)

// NotificationsWriteService is the subset of the notifications bounded
// context service used by the rest of the Service implementation.
type NotificationsWriteService interface {
	api.RecordOutcomeService
	api.NotificationLookupService
	api.CreateNotificationService
}
