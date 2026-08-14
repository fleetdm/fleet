// Package api provides the public API for the notifications bounded context.
// External code should use this package to interact with notifications.
package api

import (
	"context"
	"encoding/json"
	"time"
)

// Where a notification is in its delivery. Whether it reached the end user is
// displayed_at, not a status.
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

// Why the last delivery attempt ended the way it did. Internal only; the UI
// reads the script's exit code instead.
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

// How long a delayed notification waits. Fleet decides this, not the device.
const EndUserNotificationDelayInterval = time.Hour

// How long Fleet waits before retrying. Needing an admin to install or upgrade
// Fleet Desktop gets the longer one.
const (
	EndUserNotificationShortRetryInterval = 30 * time.Second
	EndUserNotificationLongRetryInterval  = 30 * time.Minute
)

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
// their notifications.
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

// NotificationKind owns what a notification's payload means and what happens
// when an end user acts on it or an attempt ends. Core owns delivery. Kinds
// are implemented in server/service and registered with RegisterKind, since a
// kind may need features (e.g. software) this context doesn't have.
type NotificationKind interface {
	// Name is the value stored in end_user_notifications.kind.
	Name() string
	// OnVerify sets when the notification was displayed on the host. Currently
	// unused because Fleet uses the script endpoint to report when the
	// notification was displayed.
	OnVerify(ctx context.Context, notification *EndUserNotification, displayedAt time.Time) error
	// OnDelay sets the notification to attempt later at a time the kind chooses.
	OnDelay(ctx context.Context, notification *EndUserNotification) error
	// OnOutcome runs after Fleet records how an attempt ended.
	OnOutcome(ctx context.Context, notification *EndUserNotification, outcome NotificationOutcome) error
}
