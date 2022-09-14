package apple

import "github.com/gofrs/uuid"

const DEPName = "fleet"

const EnrollmentProfileID = "com.github.fleetdm.fleet.mdm.enroll"
const PayloadTypeMDM = "com.apple.mdm"

// Profile is an apple configuration profile
// TODO:use github.com/groob/plist for encoding, need to add tags
type Profile struct {
	ID                uint   `json:"id" db:"id"`     // should this be the profile identifer in the mobileconfig
	Name              string `json:"name" db:"name"` // unique name
	ProfileIdentifier string `json:"profile_identifier" db:"profile_identifier"`
	// Description  string `json:"description" db:"description"` // human readable
	Type         string `json:"type" db:"type"` // eg enrollment, use to validate mobileconfig?
	Mobileconfig string `json:"mobileconfig"`   // store as bytes in database? signed? binary?
}

type ProfileType string

const (
	ProfileTypeEnrollment ProfileType = "enrollment"
)

// mobileconfig contains the top leve properties for configuring Device
// Management Profiles.
//
// See https://developer.apple.com/documentation/devicemanagement/toplevel.
type mobileconfig struct {
	PayloadDescription       string
	PayloadDisplayName       string
	PayloadType              string      // Can only be Configuration
	PayloadContent           interface{} // based on type, can be an array or dict?
	PayloadIdentifier        uuid.UUID
	PayloadUUID              uuid.UUID
	PayloadVersion           int
	PayloadRemovalDisallowed bool
}

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
