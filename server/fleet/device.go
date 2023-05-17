package fleet

import "time"

// DesktopSummary is a summary of the status of a host that's used by Fleet
// Desktop to operate (show/hide menu items, etc)
type DesktopSummary struct {
	FailingPolicies *uint                `json:"failing_policies_count,omitempty"`
	Notifications   DesktopNotifications `json:"notifications,omitempty"`
}

// DesktopNotifications are notifications that the fleet server sends to
// Fleet Desktop so that it can run commands or more generally react to this
// information.
type DesktopNotifications struct {
	NeedsMDMMigration bool `json:"needs_mdm_migration,omitempty"`
}

type MigrateMDMDeviceWebhookPayload struct {
	Timestamp time.Time `json:"timestamp"`
	Host      struct {
		ID             uint   `json:"id"`
		UUID           string `json:"uuid"`
		HardwareSerial string `json:"hardware_serial"`
	} `json:"host"`
}
