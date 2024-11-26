package fleet

import "encoding/json"

// OrbitConfigNotifications are notifications that the fleet server sends to
// fleetd (orbit) so that it can run commands or more generally react to this
// information.
type OrbitConfigNotifications struct {
	RenewEnrollmentProfile  bool `json:"renew_enrollment_profile,omitempty"`
	RotateDiskEncryptionKey bool `json:"rotate_disk_encryption_key,omitempty"`

	// NeedsMDMMigration is set to true if MDM is enabled for the host's
	// platform, MDM migration is enabled for that platform, and the host is
	// eligible for such a migration (e.g. it is enrolled in a third-party MDM
	// solution).
	NeedsMDMMigration bool `json:"needs_mdm_migration,omitempty"`

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

	// EnforceBitLockerEncryption is sent as true if Windows MDM is
	// enabled and the device should encrypt its disk volumes with BitLocker.
	EnforceBitLockerEncryption bool `json:"enforce_bitlocker_encryption,omitempty"`

	// PendingSoftwareInstallerIDs contains a list of software install_ids queued for installation
	PendingSoftwareInstallerIDs []string `json:"pending_software_installer_ids,omitempty"`

	// RunSetupExperience indicates whether or not Orbit should run the Fleet setup experience
	// during macOS Setup Assistant.
	RunSetupExperience bool `json:"run_setup_experience,omitempty"`

	// RunDiskEncryptionEscrow tells Orbit to prompt the end user to escrow disk encryption data
	RunDiskEncryptionEscrow bool `json:"run_disk_encryption_escrow,omitempty"`
}

type OrbitConfig struct {
	ScriptExeTimeout int                      `json:"script_execution_timeout,omitempty"`
	Flags            json.RawMessage          `json:"command_line_startup_flags,omitempty"`
	Extensions       json.RawMessage          `json:"extensions,omitempty"`
	NudgeConfig      *NudgeConfig             `json:"nudge_config,omitempty"`
	Notifications    OrbitConfigNotifications `json:"notifications,omitempty"`
	// UpdateChannels contains the TUF channels to use on fleetd components.
	//
	// If UpdateChannels is nil it means the server isn't using/setting this feature.
	UpdateChannels *OrbitUpdateChannels `json:"update_channels,omitempty"`
}

type OrbitConfigReceiver interface {
	Run(*OrbitConfig) error
}

type OrbitConfigReceiverFunc func(cfg *OrbitConfig) error

func (f OrbitConfigReceiverFunc) Run(cfg *OrbitConfig) error {
	return f(cfg)
}

// OrbitUpdateChannels hold the update channels that can be configured in fleetd agents.
type OrbitUpdateChannels struct {
	// Orbit holds the orbit channel.
	Orbit string `json:"orbit"`
	// Osqueryd holds the osqueryd channel.
	Osqueryd string `json:"osqueryd"`
	// Desktop holds the Fleet Desktop channel.
	Desktop string `json:"desktop"`
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
	// OsqueryIdentifier holds the identifier that osqueryd will use in its enrollment.
	// This is mainly used for scenarios where hosts have duplicate hardware UUID (e.g. VMs)
	// and a different identifier is used for each host (e.g. osquery's "instance" flag).
	//
	// If not set, then the HardwareUUID is used/set as the osquery identifier.
	OsqueryIdentifier string
	// ComputerName is the device's friendly name (optional).
	ComputerName string
	// HardwareModel is the device's hardware model. For example: Standard PC (Q35 + ICH9, 2009)
	HardwareModel string
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

// OrbitHostDiskEncryptionKeyPayload contains the disk encryption key for a host.
type OrbitHostDiskEncryptionKeyPayload struct {
	EncryptionKey []byte `json:"encryption_key"`
	ClientError   string `json:"client_error"`
}
