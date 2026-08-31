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
	// EndUserNotificationActed is set when the end user picked an action that
	// finishes the notification, so nothing more is sent for it. It is terminal
	// like failed and expired.
	EndUserNotificationActed = "acted"
)

// What an end user's device can do with a notification.
const (
	EndUserNotificationActionVerify = "verify"
	EndUserNotificationActionDelay  = "delay"
)

// Why the last delivery attempt ended the way it did. Internal only; the UI
// reads the script's exit code instead.
//
// Deferred is Fleet's own decision to hold a notification back behind one it
// already sent to that host. AnotherDisplayed is what Fleet Desktop reports
// when it finds a notification on screen, which is the same situation seen from
// the device rather than from the queue.
const (
	EndUserNotificationReasonDelayed           = "delayed"
	EndUserNotificationReasonDeferred          = "deferred"
	EndUserNotificationReasonBadInvocation     = "bad_invocation"
	EndUserNotificationReasonBadConfiguration  = "bad_configuration"
	EndUserNotificationReasonPageLoadFailed    = "page_load_failed"
	EndUserNotificationReasonHTTPError         = "http_error"
	EndUserNotificationReasonNoGUIUser         = "no_gui_user"
	EndUserNotificationReasonScreenLocked      = "screen_locked"
	EndUserNotificationReasonNoDisplay         = "no_display"
	EndUserNotificationReasonAnotherDisplayed  = "another_notification_displayed"
	EndUserNotificationReasonInternalError     = "internal_error"
	EndUserNotificationReasonDesktopMissing    = "fleet_desktop_missing"
	EndUserNotificationReasonDesktopTooOld     = "fleet_desktop_too_old"
	EndUserNotificationReasonScriptsDisabled   = "scripts_disabled"
	EndUserNotificationReasonURLUnresolved     = "url_unresolved"
	EndUserNotificationReasonUnexpectedFailure = "unexpected_failure"
)

// How long a delayed notification waits. Fleet decides this, not the device.
const EndUserNotificationDelayInterval = time.Hour

// How long Fleet waits before retrying. Needing an admin to install or upgrade
// Fleet Desktop gets the longer one.
const (
	EndUserNotificationShortRetryInterval = time.Minute
	EndUserNotificationLongRetryInterval  = time.Hour
)

const EndUserNotificationMaxLifetime = 24 * time.Hour

// How long a dispatched notification waits for a result before Fleet gives up
// on it. A host that never answers (wiped, fleetd removed) holds up every
// notification behind it.
const EndUserNotificationStuckDispatchTimeout = 24 * time.Hour

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

type EndUserNotificationAction struct {
	Action      *string    `json:"action"`
	DisplayedAt *time.Time `json:"displayed_at"`
}

// NotificationItem is one row of a notification's list, e.g. an app that is
// about to be updated.
type NotificationItem struct {
	SoftwareTitleID uint `json:"software_title_id"`
	// IconURL is null when the software has no icon; the page falls back to
	// matching on Name.
	IconURL     *string `json:"icon_url"`
	Name        string  `json:"name"`
	DisplayName string  `json:"display_name,omitempty"`
	// Status is an optional right-aligned label, e.g. "Installing...".
	Status string `json:"status,omitempty"`
}

// NotificationAction is a button the end user can press. ID is sent back
// verbatim on the actions endpoint. The last action is the primary one, and an
// action with the ID "dismiss" always closes the window.
type NotificationAction struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

// NotificationView is what the page inside Fleet Desktop's toast window draws.
// Kinds build it in Render, so it always reflects the notification as it is
// now rather than as it was when Fleet queued it. Title and Description may
// carry **bold** markup.
type NotificationView struct {
	UUID                string               `json:"uuid"`
	OrgLogoURLLightMode string               `json:"org_logo_url_light_mode"`
	OrgLogoURLDarkMode  string               `json:"org_logo_url_dark_mode"`
	Title               string               `json:"title"`
	Description         string               `json:"description"`
	Items               []NotificationItem   `json:"items"`
	Actions             []NotificationAction `json:"actions"`
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
	// Name is the value stored in notifications_end_user.kind.
	Name() string
	// Render builds what the end user sees. It runs on every fetch, so the copy,
	// the item list and the actions all reflect the notification's current state.
	Render(ctx context.Context, notification *EndUserNotification) (*NotificationView, error)
	// OnVerify sets when the notification was displayed on the host. Currently
	// unused because Fleet uses the script endpoint to report when the
	// notification was displayed.
	OnVerify(ctx context.Context, notification *EndUserNotification, displayedAt time.Time) error
	// OnDelay sets the notification to attempt later at a time the kind chooses.
	OnDelay(ctx context.Context, notification *EndUserNotification) error
	// OnAction handles an action ID the kind declared in Render and core doesn't
	// know about. Core keeps verify and delay for itself.
	OnAction(ctx context.Context, notification *EndUserNotification, actionID string) error
	// OnOutcome runs after Fleet records how an attempt ended.
	OnOutcome(ctx context.Context, notification *EndUserNotification, outcome NotificationOutcome) error
}
