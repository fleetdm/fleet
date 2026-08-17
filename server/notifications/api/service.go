package api

// Service is the composite interface for the notifications bounded context.
// It embeds all method-specific interfaces. Bootstrap returns this type.
type Service interface {
	ExpireAndQueueNotificationsService
	GetNotificationService
	ApplyActionService
	RecordOutcomeService
	NotificationLookupService
	DelayNotificationService
	RegisterKindService
}
