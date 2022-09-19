package apple_mdm

const DEPName = "fleet"

const (
	EnrollmentProfileID = "com.github.fleetdm.fleet.mdm.enroll"
	PayloadTypeMDM      = "com.apple.mdm"

	SCEPPath = "/mdm/apple/scep"
	MDMPath  = "/mdm/apple/mdm"

	EnrollPath    = "/api/mdm/apple/enroll"
	InstallerPath = "/api/mdm/apple/installer"
)

// Payload contains payload keys common to all payloads. Including profiles.
// See https://developer.apple.com/documentation/devicemanagement/configuring_multiple_devices_using_profiles#3234127
type Payload struct {
	PayloadDescription  string `plist:",omitempty"`
	PayloadDisplayName  string `plist:",omitempty"`
	PayloadIdentifier   string // something like com.github.fleetdm.fleet.mdm.enroll, maybe random UUID?
	PayloadOrganization string `plist:",omitempty"`
	PayloadUUID         string
	PayloadType         string
	PayloadVersion      int
}

// MDMPayload is used for mdm enrollment profiles
type MDMPayload struct {
	Payload
	IdentityCertificateUUID           string
	Topic                             string
	ServerURL                         string
	ServerCapabilities                []string `plist:",omitempty"`
	SignMessage                       bool     `plist:",omitempty"`
	CheckInURL                        string   `plist:",omitempty"`
	CheckOutWhenRemoved               bool     `plist:",omitempty"`
	AccessRights                      int
	UseDevelopmentAPNS                bool     `plist:",omitempty"`
	ServerURLPinningCertificateUUIDs  []string `plist:",omitempty"`
	CheckInURLPinningCertificateUUIDs []string `plist:",omitempty"`
	PinningRevocationCheckRequired    bool     `plist:",omitempty"`
}

// MDM commands

type CommandPayload struct {
	CommandUUID string `json:"command_uuid"`
	// Command is one of the commands below eg InstallProfile. Note that RequestType must be set, even though it's redundant.
	Command interface{} `json:"command"`
}

type InstallProfile struct {
	RequestType string `json:"request_type"`
	Payload     []byte `json:"payload"`
}

type RemoveProfile struct {
	RequestType string `json:"request_type"`
	Identifier  string `json:"identifier"`
}

type ProfileList struct {
	RequestType string `json:"request_type"`
}
