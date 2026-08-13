package fleet

import (
	"context"
	_ "embed"
	"encoding/json"
	"time"
)

// Where a notification is in its delivery, which is separate from whether it
// reached the end user: that is displayed_at being set.
const (
	EndUserNotificationPending    = "pending"
	EndUserNotificationDispatched = "dispatched"
	EndUserNotificationFailed     = "failed"
	EndUserNotificationExpired    = "expired"
)

// What an end user's device can do with a notification.
const (
	EndUserNotificationActionVerify = "verify"
	EndUserNotificationActionDelay  = "delay"
)

// Why the last attempt to display a notification ended the way it did. Recorded
// for Fleet to decide when to try again, not shown to anyone: the UI reads the
// script's exit code instead.
const (
	EndUserNotificationReasonDelayed           = "delayed"
	EndUserNotificationReasonBadInvocation     = "bad_invocation"
	EndUserNotificationReasonBadConfiguration  = "bad_configuration"
	EndUserNotificationReasonPageLoadFailed    = "page_load_failed"
	EndUserNotificationReasonHTTPError         = "http_error"
	EndUserNotificationReasonNoGUIUser         = "no_gui_user"
	EndUserNotificationReasonScreenLocked      = "screen_locked"
	EndUserNotificationReasonNoDisplay         = "no_display"
	EndUserNotificationReasonInternalError     = "internal_error"
	EndUserNotificationReasonDesktopMissing    = "fleet_desktop_missing"
	EndUserNotificationReasonDesktopTooOld     = "fleet_desktop_too_old"
	EndUserNotificationReasonUnexpectedFailure = "unexpected_failure"
)

// How long a delayed notification waits. Fleet decides this rather than the
// device, so an end user cannot push a notification out indefinitely.
const EndUserNotificationDelayInterval = time.Hour

// How long Fleet waits before trying a notification again. A host that needs an
// admin to install or upgrade Fleet Desktop waits longer, since retrying it every
// few minutes only fills the log with the same failure.
const (
	EndUserNotificationShortRetryInterval = 30 * time.Second
	EndUserNotificationLongRetryInterval  = 30 * time.Minute
)

// EndUserNotificationScript is queued for every notification. It is the same
// script every time, so all notifications share one script_contents row and the
// URL is what tells them apart.
//
//go:embed embedded_scripts/end_user_notification.sh
var EndUserNotificationScript string

type EndUserNotification struct {
	ID            uint            `db:"id"`
	UUID          string          `db:"uuid"`
	HostID        uint            `db:"host_id"`
	Status        string          `db:"status"`
	Kind          string          `db:"kind"`
	Payload       json.RawMessage `db:"payload"`
	AttemptCount  uint            `db:"attempt_count"`
	NextAttemptAt *time.Time      `db:"next_attempt_at"`
	DisplayedAt   *time.Time      `db:"displayed_at"`
	ExecutionID   *string         `db:"execution_id"`
	LastExitCode  *int64          `db:"last_exit_code"`
	LastReason    *string         `db:"last_reason"`
	ExpiresAt     *time.Time      `db:"expires_at"`
	CreatedAt     time.Time       `db:"created_at"`
	UpdatedAt     time.Time       `db:"updated_at"`
}

// EndUserNotificationAction is what an end user's device reported about one of
// their notifications. Action says what it did; the rest carries anything only
// the device knows, which for now is when the notification actually appeared.
// When it is delayed Fleet picks the new time itself.
type EndUserNotificationAction struct {
	Action      *string    `json:"action"`
	DisplayedAt *time.Time `json:"displayed_at"`
}

// NotificationOutcome is how an attempt to put a notification on screen ended.
type NotificationOutcome struct {
	Displayed   bool
	ExitCode    int64
	Reason      string
	Output      string
	ExecutionID string
}

// NotificationKind owns everything about a notification that is specific to why
// it was sent: what its payload means, what an end user acting on it should
// cause, and what a delivery outcome should cause. Core owns getting it to the
// host and knows none of that.
//
// Kinds are registered once when the service is built, in server/service.
type NotificationKind interface {
	// Name is the value stored in end_user_notifications.kind.
	Name() string
	// OnVerify runs after Fleet has recorded that the notification appeared on
	// screen, at the time the device says it did.
	OnVerify(ctx context.Context, notification *EndUserNotification, displayedAt time.Time) error
	// OnDelay runs after Fleet has put the notification back in the queue at the
	// end user's request.
	OnDelay(ctx context.Context, notification *EndUserNotification, nextAttemptAt time.Time) error
	// OnOutcome runs after Fleet has recorded how an attempt ended, whether or
	// not it will try again.
	OnOutcome(ctx context.Context, notification *EndUserNotification, outcome NotificationOutcome) error
}
