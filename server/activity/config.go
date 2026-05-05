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
