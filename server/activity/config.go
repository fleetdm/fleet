package activity

import "context"

// ActivitiesWebhookSettings contains webhook settings for activities.
type ActivitiesWebhookSettings struct {
	Enable         bool
	DestinationURL string
}

// HostActivitiesWebhook is one fleet's enabled host-activities webhook
// destination, together with the subset of the activity's hosts belonging to
// the fleet(s) configured with it.
type HostActivitiesWebhook struct {
	DestinationURL string
	HostIDs        []uint
}

// AppConfigProvider provides access to app configuration needed by the activity bounded context.
type AppConfigProvider interface {
	GetActivitiesWebhookConfig(ctx context.Context) (*ActivitiesWebhookSettings, error)
	GetHostActivitiesWebhooks(ctx context.Context, hostIDs []uint) ([]HostActivitiesWebhook, error)
}
