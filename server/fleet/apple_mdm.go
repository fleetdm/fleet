package fleet

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/micromdm/nanodep/godep"
	"github.com/micromdm/nanomdm/mdm"
	"go.mozilla.org/pkcs7"
	"howett.net/plist"
)

type MDMAppleCommandIssuer interface {
	InstallProfile(ctx context.Context, hostUUIDs []string, profile Mobileconfig, uuid string) error
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

// Mobileconfig is the byte slice corresponding to an XML property list (i.e. plist) representation
// of an Apple MDM configuration profile in Fleet.
//
// Configuration profiles are used to configure Apple devices. See also
// https://developer.apple.com/documentation/devicemanagement/configuring_multiple_devices_using_profiles.
type Mobileconfig []byte

// ParseConfigProfile attempts to parse the Mobileconfig byte slice as a Fleet MDMAppleConfigProfile.
//
// The byte slice must be XML or PKCS7 parseable. Fleet also requires that it contains both
// a PayloadIdentifier and a PayloadDisplayName and that it has PayloadType set to "Configuration".
//
// Adapted from https://github.com/micromdm/micromdm/blob/main/platform/profile/profile.go
func (mc Mobileconfig) ParseConfigProfile() (*MDMAppleConfigProfile, error) {
	mcBytes := mc
	if !bytes.HasPrefix(mcBytes, []byte("<?xml")) {
		p7, err := pkcs7.Parse(mcBytes)
		if err != nil {
			return nil, fmt.Errorf("mobileconfig is not XML nor PKCS7 parseable: %w", err)
		}
		err = p7.Verify()
		if err != nil {
			return nil, err
		}
		mcBytes = Mobileconfig(p7.Content)
	}
	var parsed struct {
		PayloadIdentifier  string
		PayloadDisplayName string
		PayloadType        string
	}
	_, err := plist.Unmarshal(mcBytes, &parsed)
	if err != nil {
		return nil, err
	}
	if parsed.PayloadType != "Configuration" {
		return nil, fmt.Errorf("invalid PayloadType: %s", parsed.PayloadType)
	}
	if parsed.PayloadIdentifier == "" {
		return nil, errors.New("empty PayloadIdentifier in profile")
	}
	if parsed.PayloadDisplayName == "" {
		return nil, errors.New("empty PayloadDisplayName in profile")
	}

	return &MDMAppleConfigProfile{
		Identifier:   parsed.PayloadIdentifier,
		Name:         parsed.PayloadDisplayName,
		Mobileconfig: mc,
	}, nil
}

// GetPayloadTypes attempts to parse the PayloadContent list of the Mobileconfig's TopLevel object.
// It returns the PayloadType for each PayloadContentItem.
//
// See also https://developer.apple.com/documentation/devicemanagement/toplevel
func (mc Mobileconfig) GetPayloadTypes() ([]string, error) {
	mcBytes := mc
	if !bytes.HasPrefix(mcBytes, []byte("<?xml")) {
		p7, err := pkcs7.Parse(mcBytes)
		if err != nil {
			return nil, fmt.Errorf("mobileconfig is not XML nor PKCS7 parseable: %w", err)
		}
		err = p7.Verify()
		if err != nil {
			return nil, err
		}
		mcBytes = Mobileconfig(p7.Content)
	}

	// unmarshal the values we need from the top-level object
	var tlo struct {
		IsEncrypted    bool
		PayloadContent []map[string]interface{}
		PayloadType    string
	}
	_, err := plist.Unmarshal(mcBytes, &tlo)
	if err != nil {
		return nil, err
	}
	// confirm that the top-level payload type matches the expected value
	if tlo.PayloadType != "Configuration" {
		return nil, &ErrInvalidPayloadType{tlo.PayloadType}
	}

	if len(tlo.PayloadContent) < 1 {
		if tlo.IsEncrypted {
			return nil, ErrEncryptedPayloadContent
		}
		return nil, ErrEmptyPayloadContent
	}

	// extract the payload types of each payload content item from the array of
	// payload dictionaries
	var result []string
	for _, payloadDict := range tlo.PayloadContent {
		pt, ok := payloadDict["PayloadType"]
		if !ok {
			continue
		}
		if s, ok := pt.(string); ok {
			result = append(result, s)
		}
	}

	return result, nil
}

type ErrInvalidPayloadType struct {
	payloadType string
}

func (e ErrInvalidPayloadType) Error() string {
	return fmt.Sprintf("invalid PayloadType: %s", e.payloadType)
}

var (
	ErrEmptyPayloadContent     = errors.New("empty PayloadContent")
	ErrEncryptedPayloadContent = errors.New("encrypted PayloadContent")
)

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
	Mobileconfig Mobileconfig `db:"mobileconfig" json:"-"`
	CreatedAt    time.Time    `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time    `db:"updated_at" json:"updated_at"`
}

// AuthzType implements authz.AuthzTyper.
func (cp MDMAppleConfigProfile) AuthzType() string {
	return "mdm_apple_config_profile"
}

// ScreenPayloadTypes screens the profile's Mobileconfig and returns an error if it
// detects certain PayloadTypes related to FileVault settings.
func (cp MDMAppleConfigProfile) ScreenPayloadTypes() error {
	pct, err := cp.Mobileconfig.GetPayloadTypes()
	if err != nil {
		switch {
		case errors.Is(err, ErrEmptyPayloadContent), errors.Is(err, ErrEncryptedPayloadContent):
			// ok, there's nothing for us to screen
		default:
			return err
		}
	}

	var screened []string
	for _, t := range pct {
		switch t {
		case "com.apple.security.FDERecoveryKeyEscrow", "com.apple.MCX.FileVault2", "com.apple.security.FDERecoveryRedirect":
			screened = append(screened, t)
		}
	}
	if len(screened) > 0 {
		return fmt.Errorf("unsupported PayloadType(s): %s", strings.Join(screened, ", "))
	}

	return nil
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

type MDMAppleProfilePayload struct {
	ProfileID         uint   `db:"profile_id"`
	ProfileIdentifier string `db:"profile_identifier"`
	HostUUID          string `db:"host_uuid"`
}

type MDMAppleBulkUpsertHostProfilePayload struct {
	ProfileID         uint
	ProfileIdentifier string
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

// TODO: docs
// QUESTION: what is `fleet` directory for?
type MDMAppleFileVaultSummary struct {
	Applied             uint `json:"applied" db:"applied"`
	ActionRequired      uint `json:"action_required" db:"action_required"`
	Enforcing           uint `json:"enforcing" db:"enforcing"`
	Failed              uint `json:"failed" db:"failed"`
	RemovingEnforcement uint `json:"removing_enforcement" db:"removing_enforcement"`
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
