package fleet

import "encoding/json"

// OrbitConfigNotifications are notifications that the fleet server sends to
// fleetd (orbit) so that it can run commands or more generally react to this
// information.
type OrbitConfigNotifications struct {
	RenewEnrollmentProfile  bool `json:"renew_enrollment_profile,omitempty"`
	RotateDiskEncryptionKey bool `json:"rotate_disk_encryption_key,omitempty"`
	NeedsMDMMigration       bool `json:"needs_mdm_migration,omitempty"`

	// NeedsProgrammaticWindowsMDMEnrollment is sent as true if Windows MDM is
	// enabled and the device should be enrolled as far as the server knows (e.g.
	// it is running Windows, is not already enrolled, etc., see
	// host.IsEligibleForWindowsMDMEnrollment for the list of conditions).
	NeedsProgrammaticWindowsMDMEnrollment bool `json:"needs_programmatic_windows_mdm_enrollment,omitempty"`
	// WindowsMDMDiscoveryEndpoint is the URL to use as Windows MDM discovery. It
	// must be sent when NeedsProgrammaticWindowsMDMEnrollment is true so that
	// the device knows where to enroll.
	WindowsMDMDiscoveryEndpoint string `json:"windows_mdm_discovery_endpoint,omitempty"`

	// NeedsProgrammaticWindowsMDMUnenrollment is sent as true if Windows MDM is
	// disabled and the device was enrolled in Fleet's MDM (see
	// host.IsEligibleForWindowsMDMUnenrollment for the list of conditions).
	NeedsProgrammaticWindowsMDMUnenrollment bool `json:"needs_programmatic_windows_mdm_unenrollment,omitempty"`

	// PendingScriptExecutionIDs lists the IDs of scripts that are pending
	// execution on that host. The scripts pending execution are those that
	// haven't received a result yet.
	PendingScriptExecutionIDs []string `json:"pending_script_execution_ids,omitempty"`
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

// ExtensionInfo holds the data of a osquery extension to apply to an Orbit client.
type ExtensionInfo struct {
	// Platform is one of "windows", "linux" or "macos".
	Platform string `json:"platform"`
	// Channel is the select TUF channel to listen for updates.
	Channel string `json:"channel"`
	// Labels are the label names the host must be member of to run this extension.
	Labels []string `json:"labels,omitempty"`
}

// Extensions holds a set of extensions to apply to an Orbit client.
// The key of the map is the extension name (as defined on the TUF server).
type Extensions map[string]ExtensionInfo

// FilterByHostPlatform filters out extensions that are not targeted for hostPlatform.
// It supports host platforms reported by osquery and by Go's runtime.GOOS.
func (es *Extensions) FilterByHostPlatform(hostPlatform string) {
	switch {
	case IsLinux(hostPlatform):
		hostPlatform = "linux"
	case hostPlatform == "darwin":
		// Osquery uses "darwin", whereas the extensions feature uses "macos".
		hostPlatform = "macos"
	}
	for extensionName, extensionInfo := range *es {
		if hostPlatform != extensionInfo.Platform {
			delete(*es, extensionName)
		}
	}
}
