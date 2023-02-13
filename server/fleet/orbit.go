package fleet

import "encoding/json"

// OrbitConfigNotifications are notifications that the fleet server sends to
// fleetd (orbit) so that it can run commands or more generally react to this
// information.
type OrbitConfigNotifications struct {
	RenewEnrollmentProfile bool `json:"renew_enrollment_profile,omitempty"`
}

type OrbitConfig struct {
	Flags         json.RawMessage          `json:"command_line_startup_flags,omitempty"`
	Extensions    json.RawMessage          `json:"extensions,omitempty"`
	NudgeConfig   *NudgeConfig             `json:"nudge_config,omitempty"`
	Notifications OrbitConfigNotifications `json:"notifications,omitempty"`
}
