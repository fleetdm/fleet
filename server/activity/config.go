package activity

import "context"

// ActivitiesWebhookSettings contains webhook settings for activities.
type ActivitiesWebhookSettings struct {
	Enable         bool
	DestinationURL string
}

// AppConfigProvider provides access to app configuration needed by the activity bounded context.
type AppConfigProvider interface {
	GetActivitiesWebhookConfig(ctx context.Context) (*ActivitiesWebhookSettings, error)
}

// UpcomingActivityActivator activates the next upcoming activity in the queue.
// This is called when an activity implements ActivityActivator.
type UpcomingActivityActivator interface {
	ActivateNextUpcomingActivity(ctx context.Context, hostID uint, fromCompletedExecID string) error
}

// WebhookSender sends a JSON payload to a URL. Implementations handle timeout
// and TLS configuration. This interface decouples the activity bounded context
// from the server package's HTTP utilities.
type WebhookSender interface {
	SendWebhookPayload(ctx context.Context, url string, payload any) error
}
