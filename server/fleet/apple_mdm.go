package fleet

import (
	"context"
	"crypto/md5" // nolint: gosec
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/mdm"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/mobileconfig"
	"github.com/fleetdm/fleet/v4/server/mdm/nanodep/godep"
)

type MDMAppleCommandIssuer interface {
	InstallProfile(ctx context.Context, hostUUIDs []string, profile mobileconfig.Mobileconfig, uuid string) error
	RemoveProfile(ctx context.Context, hostUUIDs []string, identifier string, uuid string) error
	DeviceLock(ctx context.Context, host *Host, uuid string) (unlockPIN string, err error)
	EraseDevice(ctx context.Context, host *Host, uuid string) error
	InstallEnterpriseApplication(ctx context.Context, hostUUIDs []string, uuid string, manifestURL string) error
	InstallApplication(ctx context.Context, hostUUIDs []string, uuid string, adamID string) error
	RemoveApplication(ctx context.Context, hostUUIDs []string, identifier string, uuid string) error
	DeviceConfigured(ctx context.Context, hostUUID, cmdUUID string) error
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

// MDMAppleManualEnrollmentProfile is used for authorization checks to get the standard Fleet manual
// enrollment profile. The actual data is returned as raw bytes.
type MDMAppleManualEnrollmentProfile struct{}

// AuthzType implements authz.AuthzTyper
func (m MDMAppleManualEnrollmentProfile) AuthzType() string {
	return "mdm_apple_manual_enrollment_profile"
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
	// ProfileUUID is the unique identifier of the configuration profile in
	// Fleet. For Apple profiles, it is the letter "a" followed by a uuid.
	ProfileUUID string `db:"profile_uuid" json:"profile_uuid"`
	// Deprecated: ProfileID is the old unique id of the configuration profile in
	// Fleet. It is still maintained and generated for new profiles, but only
	// used in legacy API endpoints.
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
	Checksum         []byte                      `db:"checksum" json:"checksum,omitempty"`
	LabelsIncludeAll []ConfigurationProfileLabel `db:"-" json:"labels_include_all,omitempty"`
	LabelsIncludeAny []ConfigurationProfileLabel `db:"-" json:"labels_include_any,omitempty"`
	LabelsExcludeAny []ConfigurationProfileLabel `db:"-" json:"labels_exclude_any,omitempty"`
	CreatedAt        time.Time                   `db:"created_at" json:"created_at"`
	UploadedAt       time.Time                   `db:"uploaded_at" json:"updated_at"` // NOTE: JSON field is still `updated_at` for historical reasons, would be an API breaking change
	SecretsUpdatedAt *time.Time                  `db:"secrets_updated_at" json:"-"`
}

// MDMProfilesUpdates flags updates that were done during batch processing of profiles.
type MDMProfilesUpdates struct {
	AppleConfigProfile   bool
	WindowsConfigProfile bool
	AppleDeclaration     bool
}

// ConfigurationProfileLabel represents the many-to-many relationship between
// profiles and labels.
//
// NOTE: json representation of the fields is a bit awkward to match the
// required API response, as this struct is returned within profile
// responses.
//
// NOTE The fields in this struct other than LabelName and LabelID
// MAY NOT BE SET CORRECTLY, dependong on where they're being ingested from.
type ConfigurationProfileLabel struct {
	ProfileUUID string `db:"profile_uuid" json:"-"`
	LabelName   string `db:"label_name" json:"name"`
	LabelID     uint   `db:"label_id" json:"id,omitempty"`   // omitted if 0 (which is impossible if the label is not broken)
	Broken      bool   `db:"broken" json:"broken,omitempty"` // omitted (not rendered to JSON) if false
	Exclude     bool   `db:"exclude" json:"-"`               // not rendered in JSON, used to store the profile in LabelsIncludeAll, LabelsIncludeAny, or LabelsExcludeAny on the parent profile
	RequireAll  bool   `db:"require_all" json:"-"`           // not rendered in JSON, used to store the profile in  LabelsIncludeAll, LabelsIncludeAny, or LabelsIncludeAny on the parent profile
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

func (cp MDMAppleConfigProfile) ValidateUserProvided() error {
	// first screen the top-level object for reserved identifiers and names
	if _, ok := mobileconfig.FleetPayloadIdentifiers()[cp.Identifier]; ok {
		return fmt.Errorf("payload identifier %s is not allowed", cp.Identifier)
	}
	fleetNames := mdm.FleetReservedProfileNames()
	if _, ok := fleetNames[cp.Name]; ok {
		return fmt.Errorf("payload display name %s is not allowed", cp.Name)
	}

	// then screen the payload content for reserved identifiers, names, and types
	return cp.Mobileconfig.ScreenPayloads()
}

// HostMDMAppleProfile represents the status of an Apple MDM profile in a host.
type HostMDMAppleProfile struct {
	HostUUID      string             `db:"host_uuid" json:"-"`
	CommandUUID   string             `db:"command_uuid" json:"-"`
	ProfileUUID   string             `db:"profile_uuid" json:"profile_uuid"`
	Name          string             `db:"name" json:"name"`
	Identifier    string             `db:"identifier" json:"-"`
	Status        *MDMDeliveryStatus `db:"status" json:"status"`
	OperationType MDMOperationType   `db:"operation_type" json:"operation_type"`
	Detail        string             `db:"detail" json:"detail"`
}

// ToHostMDMProfile converts the HostMDMAppleProfile to a HostMDMProfile.
func (p HostMDMAppleProfile) ToHostMDMProfile(platform string) HostMDMProfile {
	return HostMDMProfile{
		HostUUID:      p.HostUUID,
		ProfileUUID:   p.ProfileUUID,
		Name:          p.Name,
		Identifier:    p.Identifier,
		Status:        p.Status,
		OperationType: p.OperationType,
		Detail:        p.Detail,
		Platform:      platform,
	}
}

// HostMDMCertificateProfile represents the status of an MDM certificate profile (SCEP payload) along with the
// associated certificate metadata.
type HostMDMCertificateProfile struct {
	HostUUID             string             `db:"host_uuid"`
	ProfileUUID          string             `db:"profile_uuid"`
	Status               *MDMDeliveryStatus `db:"status"`
	ChallengeRetrievedAt *time.Time         `db:"challenge_retrieved_at"`
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
	ProfileUUID       string             `db:"profile_uuid"`
	ProfileIdentifier string             `db:"profile_identifier"`
	ProfileName       string             `db:"profile_name"`
	HostUUID          string             `db:"host_uuid"`
	HostPlatform      string             `db:"host_platform"`
	Checksum          []byte             `db:"checksum"`
	SecretsUpdatedAt  *time.Time         `db:"secrets_updated_at"`
	Status            *MDMDeliveryStatus `db:"status" json:"status"`
	OperationType     MDMOperationType   `db:"operation_type"`
	Detail            string             `db:"detail"`
	CommandUUID       string             `db:"command_uuid"`
}

// DidNotInstallOnHost indicates whether this profile was not installed on the host (and
// therefore is not, as far as Fleet knows, currently on the host).
func (p *MDMAppleProfilePayload) DidNotInstallOnHost() bool {
	return p.Status != nil && (*p.Status == MDMDeliveryFailed || *p.Status == MDMDeliveryPending) && p.OperationType == MDMOperationTypeInstall
}

type MDMAppleBulkUpsertHostProfilePayload struct {
	ProfileUUID       string
	ProfileIdentifier string
	ProfileName       string
	HostUUID          string
	CommandUUID       string
	OperationType     MDMOperationType
	Status            *MDMDeliveryStatus
	Detail            string
	Checksum          []byte
	SecretsUpdatedAt  *time.Time
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

// MDMCustomEnrollmentProfileItem represents an MDM enrollment profile item that
// contains custom fields.
type MDMCustomEnrollmentProfileItem struct {
	EndUserEmail string
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
	EnableReleaseDeviceManually *bool `json:"enable_release_device_manually"`
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
	// ABMTokenID is the ID of the ABM token that was used to make this DEP assignment.
	ABMTokenID *uint `db:"abm_token_id"`
}

func (h *HostDEPAssignment) IsDEPAssignedToFleet() bool {
	if h == nil {
		return false
	}
	return h.HostID > 0 && !h.AddedAt.IsZero() && h.DeletedAt == nil
}

type DEPAssignProfileResponseStatus string

const (
	DEPAssignProfileResponseSuccess       DEPAssignProfileResponseStatus = "SUCCESS"
	DEPAssignProfileResponseNotAccessible DEPAssignProfileResponseStatus = "NOT_ACCESSIBLE"
	DEPAssignProfileResponseFailed        DEPAssignProfileResponseStatus = "FAILED"
)

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
	ID         uint            `json:"-" db:"id"`
	TeamID     *uint           `json:"team_id" db:"team_id"`
	Name       string          `json:"name" db:"name"`
	Profile    json.RawMessage `json:"enrollment_profile" db:"profile"`
	UploadedAt time.Time       `json:"uploaded_at" db:"uploaded_at"`
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

// SCEPIdentityCertificate represents a certificate issued during MDM
// enrollment.
type SCEPIdentityCertificate struct {
	Serial         string    `db:"serial"`
	NotValidAfter  time.Time `db:"not_valid_after"`
	CertificatePEM []byte    `db:"certificate_pem"`
}

// SCEPIdentityAssociation represents an association between an identity
// certificate an a specific host.
type SCEPIdentityAssociation struct {
	HostUUID         string `db:"host_uuid"`
	SHA256           string `db:"sha256"`
	EnrollReference  string `db:"enroll_reference"`
	RenewCommandUUID string `db:"renew_command_uuid"`
	// EnrolledFromMigration is used for devices migrated via datababse
	// dumps (ie: "touchless")
	EnrolledFromMigration bool `db:"enrolled_from_migration"`
}

// MDMAppleDeclaration represents a DDM JSON declaration.
type MDMAppleDeclaration struct {
	// DeclarationUUID is the unique identifier of the declaration in
	// Fleet. Since we use the same endpoints for declarations and profiles:
	//    - This is marshalled as profile_uuid
	//    - The value has a prefix (TODO: @jahzielv to determine and document this)
	DeclarationUUID string `db:"declaration_uuid" json:"profile_uuid"`

	// TeamID is the id of the team with which the declaration is associated. A nil team id
	// represents a declaration that is not associated with any team.
	TeamID *uint `db:"team_id" json:"team_id"`

	// Identifier corresponds to the "Identifier" key of the associated declaration.
	// Fleet requires that Identifier must be unique in combination with the Name and TeamID.
	Identifier string `db:"identifier" json:"identifier"`

	// Name corresponds to the file name of the associated JSON declaration payload.
	// Fleet requires that Name must be unique in combination with the Identifier and TeamID.
	Name string `db:"name" json:"name"`

	// RawJSON is the raw JSON content of the declaration
	RawJSON json.RawMessage `db:"raw_json" json:"-"`

	// Token is used to identify if declaration needs to be re-applied.
	// It contains the checksum of the JSON contents and secrets updated timestamp (if secret variables are present).
	Token string `db:"token" json:"-"`

	// labels associated with this Declaration
	LabelsIncludeAll []ConfigurationProfileLabel `db:"-" json:"labels_include_all,omitempty"`
	LabelsIncludeAny []ConfigurationProfileLabel `db:"-" json:"labels_include_any,omitempty"`
	LabelsExcludeAny []ConfigurationProfileLabel `db:"-" json:"labels_exclude_any,omitempty"`

	CreatedAt        time.Time  `db:"created_at" json:"created_at"`
	UploadedAt       time.Time  `db:"uploaded_at" json:"uploaded_at"`
	SecretsUpdatedAt *time.Time `db:"secrets_updated_at" json:"-"`
}

type MDMAppleRawDeclaration struct {
	// Type is the "Type" field on the raw declaration JSON.
	Type       string `json:"Type"`
	Identifier string `json:"Identifier"`
}

// ForbiddenDeclTypes is a set of declaration types that are not allowed to be
// added by users into Fleet.
var ForbiddenDeclTypes = map[string]struct{}{
	"com.apple.configuration.account.caldav":               {},
	"com.apple.configuration.account.carddav":              {},
	"com.apple.configuration.account.exchange":             {},
	"com.apple.configuration.account.google":               {},
	"com.apple.configuration.account.ldap":                 {},
	"com.apple.configuration.account.mail":                 {},
	"com.apple.configuration.screensharing.connection":     {},
	"com.apple.configuration.security.certificate":         {},
	"com.apple.configuration.security.identity":            {},
	"com.apple.configuration.security.passkey.attestation": {},
	"com.apple.configuration.services.configuration-files": {},
	"com.apple.configuration.watch.enrollment":             {},
}

func (r *MDMAppleRawDeclaration) ValidateUserProvided() error {
	var err error

	// Check against types we don't allow
	if r.Type == `com.apple.configuration.softwareupdate.enforcement.specific` {
		return NewInvalidArgumentError(r.Type, "Declaration profile can’t include OS updates settings. To control these settings, go to OS updates.")
	}

	if _, forbidden := ForbiddenDeclTypes[r.Type]; forbidden {
		return NewInvalidArgumentError(r.Type, "Only configuration declarations that don’t require an asset reference are supported.")
	}

	if r.Type == "com.apple.configuration.management.status-subscriptions" {
		return NewInvalidArgumentError(r.Type, "Declaration profile can’t include status subscription type. To get host’s vitals, please use queries and policies.")
	}

	if !strings.HasPrefix(r.Type, "com.apple.configuration") {
		return NewInvalidArgumentError(r.Type, "Only configuration declarations (com.apple.configuration) are supported.")
	}

	return err
}

func GetRawDeclarationValues(raw []byte) (*MDMAppleRawDeclaration, error) {
	var rawDecl MDMAppleRawDeclaration
	if err := json.Unmarshal(raw, &rawDecl); err != nil {
		return nil, NewInvalidArgumentError("declaration", fmt.Sprintf("Couldn't upload. The file should include valid JSON: %s", err)).WithStatus(http.StatusBadRequest)
	}

	return &rawDecl, nil
}

// MDMAppleHostDeclaration represents the state of a declaration on a host
type MDMAppleHostDeclaration struct {
	// HostUUID is the uuid of the host affected by this declaration
	HostUUID string `db:"host_uuid" json:"-"`

	// DeclarationUUID is the unique identifier of the declaration in
	// Fleet. Since we use the same endpoints for declarations and profiles:
	//    - This is marshalled as profile_uuid
	//    - The value has a prefix (TODO: @jahzielv to determine and document this)
	DeclarationUUID string `db:"declaration_uuid" json:"profile_uuid"`

	// Name corresponds to the file name of the associated JSON declaration payload.
	Name string `db:"declaration_name" json:"name"`

	// Identifier corresponds to the "Identifier" key of the associated declaration.
	Identifier string `db:"declaration_identifier" json:"-"`

	// Status represent the current state of the declaration, as known by the Fleet server.
	Status *MDMDeliveryStatus `db:"status" json:"status"`

	// Operation type represents the operation being performed.
	OperationType MDMOperationType `db:"operation_type" json:"operation_type"`

	// Detail contains any messages that must be surfaced to the user,
	// either by the MDM protocol or the Fleet server.
	Detail string `db:"detail" json:"detail"`

	// Token is used to identify if declaration needs to be re-applied.
	// It contains the checksum of the JSON contents and secrets updated timestamp (if secret variables are present).
	Token string `db:"token" json:"-"`

	// SecretsUpdatedAt is the timestamp when the secrets were last updated or when this declaration was uploaded.
	SecretsUpdatedAt *time.Time `db:"secrets_updated_at" json:"-"`
}

func (p MDMAppleHostDeclaration) Equal(other MDMAppleHostDeclaration) bool {
	statusEqual := p.Status == nil && other.Status == nil || p.Status != nil && other.Status != nil && *p.Status == *other.Status
	secretsEqual := p.SecretsUpdatedAt == nil && other.SecretsUpdatedAt == nil || p.SecretsUpdatedAt != nil && other.SecretsUpdatedAt != nil && p.SecretsUpdatedAt.Equal(*other.SecretsUpdatedAt)
	return statusEqual &&
		p.HostUUID == other.HostUUID &&
		p.DeclarationUUID == other.DeclarationUUID &&
		p.Name == other.Name &&
		p.Identifier == other.Identifier &&
		p.OperationType == other.OperationType &&
		p.Detail == other.Detail &&
		p.Token == other.Token &&
		secretsEqual
}

func NewMDMAppleDeclaration(raw []byte, teamID *uint, name string, declType, ident string) *MDMAppleDeclaration {
	var decl MDMAppleDeclaration

	decl.Identifier = ident
	decl.Name = name
	decl.RawJSON = raw
	decl.TeamID = teamID

	return &decl
}

// MDMAppleDDMTokensResponse is the response from the DDM tokens endpoint.
//
// https://developer.apple.com/documentation/devicemanagement/tokensresponse
type MDMAppleDDMTokensResponse struct {
	SyncTokens MDMAppleDDMDeclarationsToken
}

// MDMAppleDDMDeclarationsToken is dictionary describes the state of declarations on the server.
//
// https://developer.apple.com/documentation/devicemanagement/synchronizationtokens
type MDMAppleDDMDeclarationsToken struct {
	DeclarationsToken string `db:"token"`
	// Timestamp must JSON marshal to format YYYY-mm-ddTHH:MM:SSZ
	Timestamp time.Time `db:"latest_created_timestamp"`
}

// MDMAppleDDMDeclarationItemsResponse is the response from the DDM declaration items endpoint.
//
// https://developer.apple.com/documentation/devicemanagement/declarationitemsresponse
type MDMAppleDDMDeclarationItemsResponse struct {
	Declarations      MDMAppleDDMManifestItems
	DeclarationsToken string
}

// MDMAppleDDMManifestItems is a dictionary that contains the lists of declarations available on the
// server.
//
// https://developer.apple.com/documentation/devicemanagement/declarationitemsresponse/manifestdeclarationitems
type MDMAppleDDMManifestItems struct {
	Activations    []MDMAppleDDMManifest
	Assets         []MDMAppleDDMManifest
	Configurations []MDMAppleDDMManifest
	Management     []MDMAppleDDMManifest
}

// MDMAppleDDMManifest is a dictionary that describes a declaration.
//
// https://developer.apple.com/documentation/devicemanagement/declarationitemsresponse/manifestdeclarationitems
type MDMAppleDDMManifest struct {
	Identifier  string
	ServerToken string
}

// MDMAppleDDMDeclarationItem represents a declaration item in the datastore. It is used to
// construct the DDM `declaration-items` endpoint response.
//
// https://developer.apple.com/documentation/devicemanagement/declarationitemsresponse
type MDMAppleDDMDeclarationItem struct {
	Identifier  string `db:"identifier"`
	ServerToken string `db:"token"`
}

// MDMAppleDDMDeclarationResponse represents a declaration in the datastore. It is used for the DDM
// `declaration/.../...` enpoint response.
//
// https://developer.apple.com/documentation/devicemanagement/declarationresponse
type MDMAppleDDMDeclarationResponse struct {
	Identifier  string          `db:"identifier"`
	Type        string          `db:"type"`
	Payload     json.RawMessage `db:"payload"`
	ServerToken string          `db:"server_token"`
}

// MDMAppleDDMStatusReport represents a report of the device's current state.
//
// https://developer.apple.com/documentation/devicemanagement/statusreport
type MDMAppleDDMStatusReport struct {
	StatusItems MDMAppleDDMStatusItems `json:"StatusItems"`
	Errors      []MDMAppleDDMErrors    `json:"Errors"`
}

// MDMAppleDDMStatusItems are the status items for a report.
//
// https://developer.apple.com/documentation/devicemanagement/statusreport/statusitems
type MDMAppleDDMStatusItems struct {
	Management MDMAppleDDMStatusManagement `json:"management"`
}

// MDMAppleDDMStatusManagement represents status report of the client's
// processed declarations.
//
// https://developer.apple.com/documentation/devicemanagement/statusmanagementdeclarations
type MDMAppleDDMStatusManagement struct {
	Declarations MDMAppleDDMStatusDeclarations `json:"declarations"`
}

// MDMAppleDDMStatusDeclarations represents a collection of the client's
// processed declarations.
//
// https://developer.apple.com/documentation/devicemanagement/statusmanagementdeclarationsdeclarationsobject
type MDMAppleDDMStatusDeclarations struct {
	// Activations is an array of declarations that represent the client's
	// processed activation types.
	Activations []MDMAppleDDMStatusDeclaration `json:"activations"`
	// Configurations is an array of declarations that represent the
	// client's processed configuration types.
	Configurations []MDMAppleDDMStatusDeclaration `json:"configurations"`
	// Assets is an array of declarations that represent the client's
	// processed assets.
	Assets []MDMAppleDDMStatusDeclaration `json:"assets"`
	// Management is an array of declarations that represent the client's
	// processed declaration types.
	Management []MDMAppleDDMStatusDeclaration `json:"management"`
}

type MDMAppleDeclarationValidity string

const (
	MDMAppleDeclarationValid   MDMAppleDeclarationValidity = "valid"
	MDMAppleDeclarationInvalid MDMAppleDeclarationValidity = "invalid"
	MDMAppleDeclarationUnknown MDMAppleDeclarationValidity = "valid"
)

// MDMAppleDDMStatusDeclaration represents a processed declaration for the client.
//
// https://developer.apple.com/documentation/devicemanagement/statusmanagementdeclarationsdeclarationobject
type MDMAppleDDMStatusDeclaration struct {
	// Active signals if the declaration is active on the device.
	Active bool `json:"active"`
	// Identifier is the identifier of the declaration this status report refers to.
	Identifier string `json:"identifier"`
	// Valid defines the validity of the declaration. If it's invalid, the
	// reasons property contains more details.
	Valid MDMAppleDeclarationValidity `json:"valid"`
	// ServerToken of the declaration this status report refers to.
	ServerToken string `json:"server-token"`
	// Reasons are the details of any client errors.
	Reasons []MDMAppleDDMStatusErrorReason `json:"reasons,omitempty"`
}

// A status report's error that contains the status item and the reasons for
// the error.
//
// https://developer.apple.com/documentation/devicemanagement/statusreport/error
type MDMAppleDDMErrors struct {
	// StatusItem is the status item that this error pertains to.
	StatusItem string `json:"StatusItem"`
	// Reasons is an array of reasons for the error.
	Reasons []MDMAppleDDMStatusErrorReason `json:"Reasons"`
}

// A status report that contains details about an error.
//
// https://developer.apple.com/documentation/devicemanagement/statusreason
type MDMAppleDDMStatusErrorReason struct {
	// Code is the error code for this error.
	Code string `json:"Code"`
	// Description is a short error description.
	Description string `json:"Description"`
	// Details is a dictionary that contains further details about this
	// error.
	Details map[string]any `json:"Details"`
}

// MDMAppleDDMActivationPayload represents the payload of an activation declaration.
//
// https://developer.apple.com/documentation/devicemanagement/activationsimple
type MDMAppleDDMActivationPayload struct {
	Predicate              string   `json:"Predicate"`
	StandardConfigurations []string `json:"StandardConfigurations"`
}

// MDMAppleDDMActivation represents the declaration of an activation. It combines the base
// declaation with the activation payload.
//
// https://developer.apple.com/documentation/devicemanagement/declarationbase
// https://developer.apple.com/documentation/devicemanagement/activationsimple
type MDMAppleDDMActivation struct {
	Identifier  string                       `json:"Identifier"`
	Payload     MDMAppleDDMActivationPayload `json:"Payload"`
	ServerToken string                       `json:"ServerToken"`
	Type        string                       `json:"Type"` // "com.apple.activation.simple"
}

// MDMBootstrapPackageStore is the interface to store and retrieve bootstrap
// package files. Fleet supports storing to the database and to an S3 bucket.
type MDMBootstrapPackageStore interface {
	Get(ctx context.Context, packageID string) (io.ReadCloser, int64, error)
	Put(ctx context.Context, packageID string, content io.ReadSeeker) error
	Exists(ctx context.Context, packageID string) (bool, error)
	Cleanup(ctx context.Context, usedPackageIDs []string, removeCreatedBefore time.Time) (int, error)
}

// MDMAppleMachineInfo is a [device's information][1] sent as part of an MDM enrollment profile request
//
// [1]: https://developer.apple.com/documentation/devicemanagement/machineinfo
type MDMAppleMachineInfo struct {
	IMEI                        string `plist:"IMEI,omitempty"`
	Language                    string `plist:"LANGUAGE,omitempty"`
	MDMCanRequestSoftwareUpdate bool   `plist:"MDM_CAN_REQUEST_SOFTWARE_UPDATE"`
	MEID                        string `plist:"MEID,omitempty"`
	OSVersion                   string `plist:"OS_VERSION"`
	PairingToken                string `plist:"PAIRING_TOKEN,omitempty"`
	Product                     string `plist:"PRODUCT"`
	Serial                      string `plist:"SERIAL"`
	SoftwareUpdateDeviceID      string `plist:"SOFTWARE_UPDATE_DEVICE_ID,omitempty"`
	SupplementalBuildVersion    string `plist:"SUPPLEMENTAL_BUILD_VERSION,omitempty"`
	SupplementalOSVersionExtra  string `plist:"SUPPLEMENTAL_OS_VERSION_EXTRA,omitempty"`
	UDID                        string `plist:"UDID"`
	Version                     string `plist:"VERSION"`
}

// MDMAppleSoftwareUpdateRequiredCode is the [code][1] specified by Apple to indicate that the device
// needs to perform a software update before enrollment and setup can proceed.
//
// [1]: https://developer.apple.com/documentation/devicemanagement/errorcodesoftwareupdaterequired
const MDMAppleSoftwareUpdateRequiredCode = "com.apple.softwareupdate.required"

// MDMAppleSoftwareUpdateRequiredDetails is the [details][1] specified by Apple for the
// required software update.
//
// [1]: https://developer.apple.com/documentation/devicemanagement/errorcodesoftwareupdaterequired/details
type MDMAppleSoftwareUpdateRequiredDetails struct {
	OSVersion    string `json:"OSVersion"`
	BuildVersion string `json:"BuildVersion"`
}

// MDMAppleSoftwareUpdateRequired is the [error response][1] specified by Apple to indicate that the device
// needs to perform a software update before enrollment and setup can proceed.
//
// [1]: https://developer.apple.com/documentation/devicemanagement/errorcodesoftwareupdaterequired
type MDMAppleSoftwareUpdateRequired struct {
	Code    string                                `json:"code"` // "com.apple.softwareupdate.required"
	Details MDMAppleSoftwareUpdateRequiredDetails `json:"details"`
}

func NewMDMAppleSoftwareUpdateRequired(asset MDMAppleSoftwareUpdateAsset) *MDMAppleSoftwareUpdateRequired {
	return &MDMAppleSoftwareUpdateRequired{
		Code:    MDMAppleSoftwareUpdateRequiredCode,
		Details: MDMAppleSoftwareUpdateRequiredDetails{OSVersion: asset.ProductVersion, BuildVersion: asset.Build},
	}
}

type MDMAppleSoftwareUpdateAsset struct {
	ProductVersion string `json:"ProductVersion"`
	Build          string `json:"Build"`
}

type MDMBulkUpsertManagedCertificatePayload struct {
	ProfileUUID          string
	HostUUID             string
	ChallengeRetrievedAt *time.Time
}
