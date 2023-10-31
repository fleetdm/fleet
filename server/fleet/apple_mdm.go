package fleet

import (
	"context"
	"crypto/md5" // nolint: gosec
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/mdm/apple/mobileconfig"
	"github.com/micromdm/nanodep/godep"
)

type MDMAppleCommandIssuer interface {
	InstallProfile(ctx context.Context, hostUUIDs []string, profile mobileconfig.Mobileconfig, uuid string) error
	RemoveProfile(ctx context.Context, hostUUIDs []string, identifier string, uuid string) error
	DeviceLock(ctx context.Context, hostUUIDs []string, uuid string) error
	EraseDevice(ctx context.Context, hostUUIDs []string, uuid string) error
	InstallEnterpriseApplication(ctx context.Context, hostUUIDs []string, uuid string, manifestURL string) error
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

// MDMAppleDeliveryStatus is the status of an MDM command to apply a profile
// to a device (whether it is installing or removing).
type MDMAppleDeliveryStatus string

// List of possible MDMAppleDeliveryStatus values. For a given host, the status
// of a profile can be either of those, or NULL. The meaning of the status is
// as follows:
//
//   - failed: the MDM command failed to apply, and it won't retry. This is
//     currently a terminal state. TODO(mna): currently we only retry if the
//     command failed to enqueue in ReconcileProfile (it resets the status to
//     NULL). A failure in the asynchronous actual response of the MDM command
//     (via MDMAppleCheckinAndCommandService.CommandAndReportResults) results in
//     the failed state being applied and no retry. We should probably support
//     some retries for such failures, and determine a maximum number of retries
//     before giving up (either as a count of attempts - which would require
//     storing somewhere - or as a time period, which we could determine based on
//     the timestamps, e.g. time since created_at, if we added them to
//     host_mdm_apple_profiles).
//
//   - verified: the MDM command was successfully applied, and Fleet has
//     independently verified the status. This is a terminal state.
//
//   - verifying: the MDM command was successfully applied, but Fleet has not
//     independently verified the status. This is an intermediate state,
//     it may transition to failed, pending, or NULL.
//
//   - pending: the cron job that executes the MDM commands to apply profiles
//     is processing this host, and the MDM command may even be enqueued. This
//     is a temporary state, it may transition to failed, verifying, or NULL.
//
//   - NULL: the status set for profiles that need to be applied to a host
//     (installed or removed), e.g. because the profile just got added to the
//     host's team, or because the host moved to a new team, etc. This is a
//     temporary state, it may transition to pending when the cron job runs to
//     apply the profile. It may also be simply deleted from the host's profiles
//     without the need to run an MDM command if the profile becomes unneeded and
//     that status is for an Install operation (e.g. the profile got deleted from
//     the team, or the host was moved to a team that doesn't apply that profile)
//     or vice-versa if that status is for a Remove but the profile becomes
//     required again. For the sake of statistics, as reported by
//     GetMDMAppleHostsProfilesSummary or for the list hosts filter
//     (filterHostsByMacOSSettingsStatus), a NULL status is equivalent to a
//     Pending status.
var (
	MDMAppleDeliveryFailed    MDMAppleDeliveryStatus = "failed"
	MDMAppleDeliveryVerified  MDMAppleDeliveryStatus = "verified"
	MDMAppleDeliveryVerifying MDMAppleDeliveryStatus = "verifying"
	MDMAppleDeliveryPending   MDMAppleDeliveryStatus = "pending"
)

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
	// Checksum is an MD5 hash of the Mobileconfig bytes
	Checksum  []byte    `db:"checksum" json:"checksum,omitempty"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
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

type HostMDMProfileDetail string

const (
	HostMDMProfileDetailFailedWasVerified  HostMDMProfileDetail = "Failed, was verified"
	HostMDMProfileDetailFailedWasVerifying HostMDMProfileDetail = "Failed, was verifying"
)

// Message returns a human-friendly message for the detail.
func (d HostMDMProfileDetail) Message() string {
	switch d {
	case HostMDMProfileDetailFailedWasVerified:
		return "This setting had been verified by osquery, but has since been found missing on the host."
	case HostMDMProfileDetailFailedWasVerifying:
		return "The MDM protocol returned a success but the setting couldn’t be verified by osquery."
	default:
		return string(d)
	}
}

type MDMAppleProfilePayload struct {
	ProfileID         uint                    `db:"profile_id"`
	ProfileIdentifier string                  `db:"profile_identifier"`
	ProfileName       string                  `db:"profile_name"`
	HostUUID          string                  `db:"host_uuid"`
	Checksum          []byte                  `db:"checksum"`
	Status            *MDMAppleDeliveryStatus `db:"status" json:"status"`
	OperationType     MDMAppleOperationType   `db:"operation_type"`
	Detail            string                  `db:"detail"`
	CommandUUID       string                  `db:"command_uuid"`
}

type MDMAppleBulkUpsertHostProfilePayload struct {
	ProfileID         uint
	ProfileIdentifier string
	ProfileName       string
	HostUUID          string
	CommandUUID       string
	OperationType     MDMAppleOperationType
	Status            *MDMAppleDeliveryStatus
	Detail            string
	Checksum          []byte
}

// MDMAppleConfigProfilesSummary reports the number of hosts being managed with MDM configuration
// profiles. Each host may be counted in only one of four mutually-exclusive categories:
// Failed, Pending, Verifying, or Verified.
type MDMAppleConfigProfilesSummary struct {
	// Verified includes each host where Fleet has verified the installation of all of the
	// profiles currently applicable to the host. If any of the profiles are pending, failed, or
	// subject to verification for the host, the host is not counted as verified.
	Verified uint `json:"verified" db:"verified"`
	// Verifying includes each host where the MDM service has successfully delivered all of the
	// profiles currently applicable to the host. If any of the profiles are pending or failed for
	// the host, the host is not counted as verifying.
	Verifying uint `json:"verifying" db:"verifying"`
	// Pending includes each host that has not yet applied one or more of the profiles currently
	// applicable to the host. If a host failed to apply any profiles, it is not counted as pending.
	Pending uint `json:"pending" db:"pending"`
	// Failed includes each host that has failed to apply one or more of the profiles currently
	// applicable to the host.
	Failed uint `json:"failed" db:"failed"`
}

// MDMAppleFileVaultSummary reports the number of macOS hosts being managed with Apples disk
// encryption profiles. Each host may be counted in only one of six mutually-exclusive categories:
// Verified, Verifying, ActionRequired, Enforcing, Failed, RemovingEnforcement.
type MDMAppleFileVaultSummary struct {
	Verified            uint `json:"verified" db:"verified"`
	Verifying           uint `json:"verifying" db:"verifying"`
	ActionRequired      uint `json:"action_required" db:"action_required"`
	Enforcing           uint `json:"enforcing" db:"enforcing"`
	Failed              uint `json:"failed" db:"failed"`
	RemovingEnforcement uint `json:"removing_enforcement" db:"removing_enforcement"`
}

// MDMAppleBootstrapPackageSummary reports the number of hosts that are targeted to install the
// MDM bootstrap package. Each host may be counted in only one of three mutually-exclusive categories:
// Failed, Pending, or Installed.
type MDMAppleBootstrapPackageSummary struct {
	// Installed includes each host that has acknowledged the MDM command to install the bootstrap
	// package.
	Installed uint `json:"installed" db:"installed"`
	// Pending includes each host that has not acknowledged the MDM command to install the bootstrap
	// package or reported an error for such command.
	Pending uint `json:"pending" db:"pending"`
	// Failed includes each host that has reported an error for the MDM command to install the
	// bootstrap package.
	Failed uint `json:"failed" db:"failed"`
}

// MDMAppleFleetdConfig contains the fields used to configure
// `fleetd` in macOS devices via a configuration profile.
type MDMAppleFleetdConfig struct {
	FleetURL      string
	EnrollSecret  string
	EnableScripts bool
}

// MDMApplePreassignProfilePayload is the payload accepted by the endpoint that
// preassigns profiles to hosts before generating corresponding teams for each
// unique set of profiles and assigning hosts to those teams and profiles. For
// example, puppet scripts use this.
type MDMApplePreassignProfilePayload struct {
	ExternalHostIdentifier string `json:"external_host_identifier"`
	HostUUID               string `json:"host_uuid"`
	Profile                []byte `json:"profile"`
	Group                  string `json:"group"`
	Exclude                bool   `json:"exclude"`
}

// HexMD5Hash returns the hex-encoded MD5 hash of the profile. Note that MD5 is
// broken and we should consider moving to a better hash, but it needs to match
// the hashing algorithm used by the Mysql database for profiles (SHA2 would be
// an option: https://dev.mysql.com/doc/refman/5.7/en/encryption-functions.html#function_sha2).
func (p MDMApplePreassignProfilePayload) HexMD5Hash() string {
	sum := md5.Sum(p.Profile) //nolint: gosec

	// mysql's HEX function returns uppercase
	return strings.ToUpper(hex.EncodeToString(sum[:]))
}

// MDMApplePreassignHostProfiles represents the set of profiles that were
// pre-assigned to a given host identified by its UUID.
type MDMApplePreassignHostProfiles struct {
	HostUUID string
	Profiles []MDMApplePreassignProfile
}

// MDMApplePreassignProfile represents a single profile pre-assigned to a host.
type MDMApplePreassignProfile struct {
	Profile    []byte
	Group      string
	HexMD5Hash string
	Exclude    bool
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

// MDMAppleSetupPayload describes the payload accepted by the endpoint to
// update specific MDM macos setup values for a team (or no team).
type MDMAppleSetupPayload struct {
	TeamID                      *uint `json:"team_id"`
	EnableEndUserAuthentication *bool `json:"enable_end_user_authentication"`
}

// AuthzType implements authz.AuthzTyper.
func (p MDMAppleSetupPayload) AuthzType() string {
	return "mdm_apple_settings"
}

// HostDEPAssignment represents a row in the host_dep_assignments table.
type HostDEPAssignment struct {
	// HostID is the id of the host in Fleet.
	HostID uint `db:"host_id"`
	// AddedAt is the timestamp when Fleet was notified that device was added to the Fleet MDM
	// server in Apple Busines Manager (ABM).
	AddedAt time.Time `db:"added_at"`
	// DeletedAt is the timestamp  when Fleet was notified that device was deleted from the Fleet
	// MDM server in Apple Busines Manager (ABM).
	DeletedAt *time.Time `db:"deleted_at"`
}

func (h *HostDEPAssignment) IsDEPAssignedToFleet() bool {
	if h == nil {
		return false
	}
	return h.HostID > 0 && !h.AddedAt.IsZero() && h.DeletedAt == nil
}

// NanoEnrollment represents a row in the nano_enrollments table managed by
// nanomdm. It is meant to be used internally by the server, not to be returned
// as part of endpoints, and as a precaution its json-encoding is explicitly
// ignored.
type NanoEnrollment struct {
	ID               string `json:"-" db:"id"`
	DeviceID         string `json:"-" db:"device_id"`
	Type             string `json:"-" db:"type"`
	Enabled          bool   `json:"-" db:"enabled"`
	TokenUpdateTally int    `json:"-" db:"token_update_tally"`
}

// MDMCommandListOptions defines the options to control the list of MDM
// Apple Commands to return. Although it only supports the standard list
// options for now, in the future we expect to add filtering options.
//
// https://github.com/fleetdm/fleet/issues/11008#issuecomment-1503466119
type MDMCommandListOptions struct {
	ListOptions
}

// MDMAppleCommand represents an MDM Apple command that has been enqueued for
// execution. It is similar to MDMAppleCommandResult, but a separate struct is
// used as there are plans to evolve the `fleetctl get mdm-commands` command
// output in the future to list one row per command instead of one per
// command-host combination, and this fleetctl command is the only use of this
// struct at the moment. Also, it is filled a bit differently than what we do
// in MDMAppleCommandResult, since it needs to join with the hosts in the
// query to make authorization (retrieving the team id) manageable.
//
// https://github.com/fleetdm/fleet/issues/11008#issuecomment-1503466119
type MDMAppleCommand struct {
	// DeviceID is the MDM enrollment ID. This is the same as the host UUID.
	DeviceID string `json:"device_id" db:"device_id"`
	// CommandUUID is the unique identifier of the command.
	CommandUUID string `json:"command_uuid" db:"command_uuid"`
	// UpdatedAt is the last update timestamp of the command result.
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
	// RequestType is the command's request type, which is basically the
	// command name.
	RequestType string `json:"request_type" db:"request_type"`
	// Status is the command status. One of Acknowledged, Error, or NotNow.
	Status string `json:"status" db:"status"`
	// Hostname is the hostname of the host that executed the command.
	Hostname string `json:"hostname" db:"hostname"`
	// TeamID is the host's team, null if the host is in no team. This is used
	// to authorize the user to see the command, it is not returned as part of
	// the response payload.
	TeamID *uint `json:"-" db:"team_id"`
}

// MDMAppleSetupAssistant represents the setup assistant set for a given team
// or no team.
type MDMAppleSetupAssistant struct {
	ID          uint            `json:"-" db:"id"`
	TeamID      *uint           `json:"team_id" db:"team_id"`
	Name        string          `json:"name" db:"name"`
	Profile     json.RawMessage `json:"enrollment_profile" db:"profile"`
	ProfileUUID string          `json:"-" db:"profile_uuid"`
	UploadedAt  time.Time       `json:"uploaded_at" db:"uploaded_at"`
}

// AuthzType implements authz.AuthzTyper.
func (a MDMAppleSetupAssistant) AuthzType() string {
	return "mdm_apple_setup_assistant"
}

// ProfileMatcher defines the methods required to preassign and retrieve MDM
// profiles for matching with teams and associating with hosts. A Redis-based
// implementation is used in production.
type ProfileMatcher interface {
	PreassignProfile(ctx context.Context, payload MDMApplePreassignProfilePayload) error
	RetrieveProfiles(ctx context.Context, externalHostIdentifier string) (MDMApplePreassignHostProfiles, error)
}
