package fleet

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/mdm/apple/mobileconfig"
	"github.com/micromdm/nanodep/godep"
	"github.com/micromdm/nanomdm/mdm"
)

type MDMAppleCommandIssuer interface {
	InstallProfile(ctx context.Context, hostUUIDs []string, profile mobileconfig.Mobileconfig, uuid string) error
	RemoveProfile(ctx context.Context, hostUUIDs []string, identifier string, uuid string) error
	DeviceLock(ctx context.Context, hostUUIDs []string, uuid string) error
	EraseDevice(ctx context.Context, hostUUIDs []string, uuid string) error
}

// MDMAppleEnrollmentType is the type for Apple MDM enrollments.
type MDMAppleEnrollmentType string

const (
	// MDMAppleEnrollmentTypeAutomatic is the value for automatic enrollments.
	MDMAppleEnrollmentTypeAutomatic MDMAppleEnrollmentType = "automatic"
	// MDMAppleEnrollmentTypeManual is the value for manual enrollments.
	MDMAppleEnrollmentTypeManual MDMAppleEnrollmentType = "manual"
)

// Well-known status responses
const (
	MDMAppleStatusAcknowledged       = "Acknowledged"
	MDMAppleStatusError              = "Error"
	MDMAppleStatusCommandFormatError = "CommandFormatError"
	MDMAppleStatusIdle               = "Idle"
	MDMAppleStatusNotNow             = "NotNow"
)

type MDMAppleDeliveryStatus string

var (
	MDMAppleDeliveryFailed  MDMAppleDeliveryStatus = "failed"
	MDMAppleDeliveryApplied MDMAppleDeliveryStatus = "applied"
	MDMAppleDeliveryPending MDMAppleDeliveryStatus = "pending"
)

func MDMAppleDeliveryStatusFromCommandStatus(cmdStatus string) *MDMAppleDeliveryStatus {
	switch cmdStatus {
	case MDMAppleStatusAcknowledged:
		return &MDMAppleDeliveryApplied
	case MDMAppleStatusError, MDMAppleStatusCommandFormatError:
		return &MDMAppleDeliveryFailed
	case MDMAppleStatusIdle, MDMAppleStatusNotNow:
		return &MDMAppleDeliveryPending
	default:
		return nil
	}
}

type MDMAppleOperationType string

const (
	MDMAppleOperationTypeInstall MDMAppleOperationType = "install"
	MDMAppleOperationTypeRemove  MDMAppleOperationType = "remove"
)

// MDMAppleEnrollmentProfilePayload contains the data necessary to create
// an enrollment profile in Fleet.
type MDMAppleEnrollmentProfilePayload struct {
	// Type is the type of the enrollment.
	Type MDMAppleEnrollmentType `json:"type"`
	// DEPProfile is the JSON object with the following Apple-defined fields:
	// https://developer.apple.com/documentation/devicemanagement/profile
	//
	// DEPProfile is nil when Type is MDMAppleEnrollmentTypeManual.
	DEPProfile *json.RawMessage `json:"dep_profile"`
	// Token should be auto-generated.
	Token string `json:"-"`
}

// MDMAppleEnrollmentProfile represents an Apple MDM enrollment profile in Fleet.
// Such enrollment profiles are used to enroll Apple devices to Fleet.
type MDMAppleEnrollmentProfile struct {
	// ID is the unique identifier of the enrollment in Fleet.
	ID uint `json:"id" db:"id"`
	// Token is a random identifier for an enrollment. Currently as the authentication
	// token to protect access to the enrollment.
	Token string `json:"token" db:"token"`
	// Type is the type of the enrollment.
	Type MDMAppleEnrollmentType `json:"type" db:"type"`
	// DEPProfile is the JSON object with the following Apple-defined fields:
	// https://developer.apple.com/documentation/devicemanagement/profile
	//
	// DEPProfile is nil when Type is MDMAppleEnrollmentTypeManual.
	DEPProfile *json.RawMessage `json:"dep_profile" db:"dep_profile"`
	// EnrollmentURL is the URL where an enrollement is served.
	EnrollmentURL string `json:"enrollment_url" db:"-"`

	UpdateCreateTimestamps
}

// AuthzType implements authz.AuthzTyper.
func (m MDMAppleEnrollmentProfile) AuthzType() string {
	return "mdm_apple_enrollment_profile"
}

// MDMAppleDEPKeyPair contains the DEP public key certificate and private key pair. Both are PEM encoded.
type MDMAppleDEPKeyPair struct {
	PublicKey  []byte `json:"public_key"`
	PrivateKey []byte `json:"private_key"`
}

// MDMAppleCommandResult holds the result of a command execution provided by the target device.
type MDMAppleCommandResult struct {
	// ID is the enrollment ID. This should be the same as the device ID.
	ID string `json:"id" db:"id"`
	// CommandUUID is the unique identifier of the command.
	CommandUUID string `json:"command_uuid" db:"command_uuid"`
	// Status is the command status. One of Acknowledged, Error, or NotNow.
	Status string `json:"status" db:"status"`
	// Result is the original command result XML plist. If the status is Error, it will include the
	// ErrorChain key with more information.
	Result []byte `json:"result" db:"result"`
}

// AuthzType implements authz.AuthzTyper.
func (m MDMAppleCommandResult) AuthzType() string {
	return "mdm_apple_command_result"
}

// MDMAppleInstaller holds installer packages for Apple devices.
type MDMAppleInstaller struct {
	// ID is the unique identifier of the installer in Fleet.
	ID uint `json:"id" db:"id"`
	// Name is the name of the installer (usually the package file name).
	Name string `json:"name" db:"name"`
	// Size is the size of the installer package.
	Size int64 `json:"size" db:"size"`
	// Manifest is the manifest of the installer. Generated from the installer
	// contents and ready to use in `InstallEnterpriseApplication` commands.
	Manifest string `json:"manifest" db:"manifest"`
	// Installer is the actual installer contents.
	Installer []byte `json:"-" db:"installer"`
	// URLToken is a random token used for authentication to protect access to installers.
	// Applications deployede via InstallEnterpriseApplication must be publicly accessible,
	// this hard to guess token provides some protection.
	URLToken string `json:"url_token" db:"url_token"`
	// URL is the full URL where the installer is served.
	URL string `json:"url"`
}

// AuthzType implements authz.AuthzTyper.
func (m MDMAppleInstaller) AuthzType() string {
	return "mdm_apple_installer"
}

// MDMAppleDevice represents an MDM enrolled Apple device.
type MDMAppleDevice struct {
	// ID is the device hardware UUID.
	ID string `json:"id" db:"id"`
	// SerialNumber is the serial number of the Apple device.
	SerialNumber string `json:"serial_number" db:"serial_number"`
	// Enabled indicates whether the device is currently enrolled.
	// It's set to false when a device unenrolls from Fleet.
	Enabled bool `json:"enabled" db:"enabled"`
}

// AuthzType implements authz.AuthzTyper.
func (m MDMAppleDevice) AuthzType() string {
	return "mdm_apple_device"
}

// MDMAppleDEPDevice represents an Apple device in Apple Business Manager (ABM).
type MDMAppleDEPDevice struct {
	godep.Device
}

// AuthzType implements authz.AuthzTyper.
func (m MDMAppleDEPDevice) AuthzType() string {
	return "mdm_apple_dep_device"
}

// These following types are copied from nanomdm.

// EnrolledAPIResult is a per-enrollment API result.
type EnrolledAPIResult struct {
	PushError    string `json:"push_error,omitempty"`
	PushResult   string `json:"push_result,omitempty"`
	CommandError string `json:"command_error,omitempty"`
}

// EnrolledAPIResults is a map of enrollments to a per-enrollment API result.
type EnrolledAPIResults map[string]*EnrolledAPIResult

// CommandEnqueueResult is the result of a command execution on enrolled Apple devices.
type CommandEnqueueResult struct {
	// Status is the status of the command.
	Status EnrolledAPIResults `json:"status,omitempty"`
	// NoPush indicates whether the command was issued with no_push.
	// If this is true, then Fleet won't send a push notification to devices.
	NoPush bool `json:"no_push,omitempty"`
	// PushError indicates the error when trying to send push notification
	// to target devices.
	PushError string `json:"push_error,omitempty"`
	// CommandError holds the error when enqueueing the command.
	CommandError string `json:"command_error,omitempty"`
	// CommandUUID is the unique identifier for the command.
	CommandUUID string `json:"command_uuid,omitempty"`
	// RequestType is the name of the command.
	RequestType string `json:"request_type,omitempty"`
}

// MDMAppleCommand represents an Apple MDM command.
type MDMAppleCommand struct {
	*mdm.Command
}

// AuthzType implements authz.AuthzTyper.
func (m MDMAppleCommand) AuthzType() string {
	return "mdm_apple_command"
}

// MDMAppleHostDetails represents the device identifiers used to ingest an MDM device as a Fleet
// host pending enrollment.
// See also https://developer.apple.com/documentation/devicemanagement/authenticaterequest.
type MDMAppleHostDetails struct {
	SerialNumber string
	UDID         string
	Model        string
}

type MDMAppleCommandTimeoutError struct{}

func (e MDMAppleCommandTimeoutError) Error() string {
	return "Timeout waiting for MDM device to acknowledge command"
}

func (e MDMAppleCommandTimeoutError) StatusCode() int {
	return http.StatusGatewayTimeout
}

// MDMAppleConfigProfile represents an Apple MDM configuration profile in Fleet.
// Configuration profiles are used to configure Apple devices .
// See also https://developer.apple.com/documentation/devicemanagement/configuring_multiple_devices_using_profiles.
type MDMAppleConfigProfile struct {
	// ProfileID is the unique id of the configuration profile in Fleet
	ProfileID uint `db:"profile_id" json:"profile_id"`
	// TeamID is the id of the team with which the configuration is associated. A nil team id
	// represents a configuration profile that is not associated with any team.
	TeamID *uint `db:"team_id" json:"team_id"`
	// Identifier corresponds to the payload identifier of the associated mobileconfig payload.
	// Fleet requires that Identifier must be unique in combination with the Name and TeamID.
	Identifier string `db:"identifier" json:"identifier"`
	// Name corresponds to the payload display name of the associated mobileconfig payload.
	// Fleet requires that Name must be unique in combination with the Identifier and TeamID.
	Name string `db:"name" json:"name"`
	// Mobileconfig is the byte slice corresponding to the XML property list (i.e. plist)
	// representation of the configuration profile. It must be XML or PKCS7 parseable.
	Mobileconfig mobileconfig.Mobileconfig `db:"mobileconfig" json:"-"`
	CreatedAt    time.Time                 `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time                 `db:"updated_at" json:"updated_at"`
}

func NewMDMAppleConfigProfile(raw []byte, teamID *uint) (*MDMAppleConfigProfile, error) {
	mc := mobileconfig.Mobileconfig(raw)
	cp, err := mc.ParseConfigProfile()
	if err != nil {
		return nil, fmt.Errorf("new MDMAppleConfigProfile: %w", err)
	}
	return &MDMAppleConfigProfile{
		TeamID:       teamID,
		Identifier:   cp.PayloadIdentifier,
		Name:         cp.PayloadDisplayName,
		Mobileconfig: mc,
	}, nil
}

// AuthzType implements authz.AuthzTyper.
func (cp MDMAppleConfigProfile) AuthzType() string {
	return "mdm_apple_config_profile"
}

func (cp MDMAppleConfigProfile) ValidateUserProvided() error {
	if _, ok := mobileconfig.FleetPayloadIdentifiers()[cp.Identifier]; ok {
		return fmt.Errorf("payload identifier %s is not allowed", cp.Identifier)
	}

	return cp.Mobileconfig.ScreenPayloads()
}

// HostMDMAppleProfile represents the status of an Apple MDM profile in a host.
type HostMDMAppleProfile struct {
	HostUUID      string                  `db:"host_uuid" json:"-"`
	CommandUUID   string                  `db:"command_uuid" json:"-"`
	ProfileID     uint                    `db:"profile_id" json:"profile_id"`
	Name          string                  `db:"name" json:"name"`
	Identifier    string                  `db:"identifier" json:"-"`
	Status        *MDMAppleDeliveryStatus `db:"status" json:"status"`
	OperationType MDMAppleOperationType   `db:"operation_type" json:"operation_type"`
	Detail        string                  `db:"detail" json:"detail"`
}

func (p HostMDMAppleProfile) IgnoreMDMClientError() bool {
	switch p.OperationType {
	case MDMAppleOperationTypeRemove:
		switch {
		case strings.Contains(p.Detail, "MDMClientError (89)"):
			return true
		}
	}
	return false
}

type MDMAppleProfilePayload struct {
	ProfileID         uint   `db:"profile_id"`
	ProfileIdentifier string `db:"profile_identifier"`
	ProfileName       string `db:"profile_name"`
	HostUUID          string `db:"host_uuid"`
}

type MDMAppleBulkUpsertHostProfilePayload struct {
	ProfileID         uint
	ProfileIdentifier string
	ProfileName       string
	HostUUID          string
	CommandUUID       string
	OperationType     MDMAppleOperationType
	Status            *MDMAppleDeliveryStatus
}

// MDMAppleHostsProfilesSummary reports the number of hosts being managed with MDM configuration
// profiles. Each host may be counted in only one of three mutually-exclusive categories:
// Failed, Pending, or Latest.
type MDMAppleHostsProfilesSummary struct {
	// Latest includes each host that has successfully applied all of the profiles currently
	// applicable to the host. If any of the profiles are pending or failed for the host, the host
	// is not counted as latest.
	Latest uint `json:"latest" db:"applied"`
	// Pending includes each host that has not yet applied one or more of the profiles currently
	// applicable to the host. If a host failed to apply any profiles, it is not counted as pending.
	Pending uint `json:"pending" db:"pending"`
	// Failed includes each host that has failed to apply one or more of the profiles currently
	// applicable to the host.
	Failed uint `json:"failing" db:"failed"`
}

// MDMAppleFleetdConfig contains the fields used to configure
// `fleetd` in macOS devices via a configuration profile.
type MDMAppleFleetdConfig struct {
	FleetURL     string
	EnrollSecret string
}

// MDMAppleSettingsPayload describes the payload accepted by the endpoint to
// update specific MDM macos settings for a team (or no team).
type MDMAppleSettingsPayload struct {
	TeamID               *uint `json:"team_id"`
	EnableDiskEncryption *bool `json:"enable_disk_encryption"`
}

// AuthzType implements authz.AuthzTyper.
func (p MDMAppleSettingsPayload) AuthzType() string {
	return "mdm_apple_settings"
}
