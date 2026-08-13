package fleet

import (
	"encoding/json"
	"time"
)

const (
	EndUserNotificationPending    = "pending"
	EndUserNotificationDispatched = "dispatched"
	EndUserNotificationDisplayed  = "displayed"
	EndUserNotificationExpired    = "expired"
)

// EndUserNotificationScript is queued for every notification. It is the same
// script every time, so all notifications share one script_contents row and the
// URL is what tells them apart.
//
// TODO(JK): stub. The real script checks that Fleet Desktop is installed and
// advertises the notify capability, finds the console user, and runs the binary
// as them. See https://github.com/fleetdm/fleet/issues/39178
const EndUserNotificationScript = `#!/bin/sh
echo "notification url: $FLEET_VAR_PATCH_NOTIFICATION_URL"
`

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
// their notifications. The fields it sets say which action it took.
type EndUserNotificationAction struct {
	DisplayedAt   *time.Time `json:"displayed_at"`
	NextAttemptAt *time.Time `json:"next_attempt_at"`
	LastReason    *string    `json:"last_reason"`
}
