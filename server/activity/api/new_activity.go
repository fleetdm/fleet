package api

import (
	"context"
	"encoding/json"
	"time"
)

// ActivityAutomationAuthor is the name used for the actor when an activity
// is recorded as a result of an automated action (cron job, webhook, etc.)
// or policy automation (i.e. triggered by a failing policy).
const ActivityAutomationAuthor = "Fleet"

// ActivityWebhookContextKey is the context key used to indicate that the activity webhook
// has been processed. This is a sanity check to ensure callers use the service layer
// (which handles webhooks) rather than calling the datastore directly.
type activityWebhookContextKeyType struct{}

// ActivityWebhookContextKey is used to mark that the webhook was processed before storing the activity.
var ActivityWebhookContextKey = activityWebhookContextKeyType{}

// User represents user information for activity recording.
type User struct {
	ID      uint
	Name    string
	Email   string
	Deleted bool
}

// ActivityDetails defines the interface for activity detail types.
// This is satisfied by fleet.ActivityDetails types.
type ActivityDetails interface {
	ActivityName() string
}

// AutomatableActivity indicates the activity was initiated by automation.
type AutomatableActivity interface {
	WasFromAutomation() bool
}

// ActivityHosts indicates the activity is associated with specific hosts.
type ActivityHosts interface {
	HostIDs() []uint
}

// ActivityHostOnly indicates the activity is host-scoped only.
type ActivityHostOnly interface {
	HostOnly() bool
}

// ActivityActivator indicates the activity should activate the next upcoming activity.
type ActivityActivator interface {
	MustActivateNextUpcomingActivity() bool
	ActivateNextUpcomingActivityArgs() (hostID uint, cmdUUID string)
}

// WebhookPayload is the payload sent to the activities webhook.
type WebhookPayload struct {
	Timestamp     time.Time        `json:"timestamp"`
	ActorFullName *string          `json:"actor_full_name"`
	ActorID       *uint            `json:"actor_id"`
	ActorEmail    *string          `json:"actor_email"`
	Type          string           `json:"type"`
	Details       *json.RawMessage `json:"details"`
}

// NewActivityService is implemented by the activity bounded context for creating activities.
type NewActivityService interface {
	// NewActivity creates a new activity record and fires the webhook if configured.
	// user can be nil for automation-initiated activities.
	NewActivity(ctx context.Context, user *User, activity ActivityDetails) error
}
