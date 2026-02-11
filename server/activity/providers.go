package activity

// DataProviders combines all external dependency interfaces for the activity
// bounded context. The ACL adapter implements this single interface.
type DataProviders interface {
	UserProvider
	HostProvider
	AppConfigProvider
	UpcomingActivityActivator
	WebhookSender
	URLMasker
}
