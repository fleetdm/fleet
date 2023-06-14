package fleet

import "encoding/json"

// OrbitConfigNotifications are notifications that the fleet server sends to
// fleetd (orbit) so that it can run commands or more generally react to this
// information.
type OrbitConfigNotifications struct {
	RenewEnrollmentProfile      bool   `json:"renew_enrollment_profile,omitempty"`
	RotateDiskEncryptionKey     bool   `json:"rotate_disk_encryption_key,omitempty"`
	NeedsMDMMigration           bool   `json:"needs_mdm_migration,omitempty"`
	NeedsWindowsMDMEnrollment   bool   `json:"needs_windows_mdm_enrollment,omitempty"`
	WindowsMDMDiscoveryEndpoint string `json:"windows_mdm_discovery_endpoint,omitempty"`
}

type OrbitConfig struct {
	Flags         json.RawMessage          `json:"command_line_startup_flags,omitempty"`
	Extensions    json.RawMessage          `json:"extensions,omitempty"`
	NudgeConfig   *NudgeConfig             `json:"nudge_config,omitempty"`
	Notifications OrbitConfigNotifications `json:"notifications,omitempty"`
}

// OrbitHostInfo holds device information used during Orbit enroll.
type OrbitHostInfo struct {
	// HardwareUUID is the device's hardware UUID.
	HardwareUUID string
	// HardwareSerial is the device's serial number. Only set for
	// macOS and Linux hosts.
	HardwareSerial string
	// Hostname is the device hostname.
	Hostname string
	// Platform is the device's platform as defined by osquery.
	Platform string
}
