package fleet

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	mdm_types "github.com/fleetdm/fleet/v4/server/mdm"
)

const (
	MDMPlatformApple     = "apple"
	MDMPlatformMicrosoft = "microsoft"

	MDMAppleDeclarationUUIDPrefix = "d"
	MDMAppleProfileUUIDPrefix     = "a"
	MDMWindowsProfileUUIDPrefix   = "w"

	// RefetchMDMUnenrollCriticalQueryDuration is the duration to set the
	// RefetchCriticalQueriesUntil field when migrating a device from a
	// third-party MDM solution to Fleet.
	RefetchMDMUnenrollCriticalQueryDuration = 3 * time.Minute
)

type AppleMDM struct {
	CommonName   string    `json:"common_name"`
	SerialNumber string    `json:"serial_number"`
	Issuer       string    `json:"issuer"`
	RenewDate    time.Time `json:"renew_date"`
}

func (a AppleMDM) AuthzType() string {
	return "mdm_apple"
}

type AppleBM struct {
	AppleID      string    `json:"apple_id"`
	OrgName      string    `json:"org_name"`
	MDMServerURL string    `json:"mdm_server_url"`
	RenewDate    time.Time `json:"renew_date"`
	DefaultTeam  string    `json:"default_team"`
}

func (a AppleBM) AuthzType() string {
	return "mdm_apple"
}

// TODO: during API implementation, remove AppleBM above or reconciliate those
// two types. We'll likely need a new authz type for the ABM token.
type ABMToken struct {
	ID                  uint      `db:"id" json:"id"`
	AppleID             string    `db:"apple_id" json:"apple_id"`
	OrganizationName    string    `db:"organization_name" json:"org_name"`
	RenewAt             time.Time `db:"renew_at" json:"renew_date"`
	TermsExpired        bool      `db:"terms_expired" json:"terms_expired"`
	MacOSDefaultTeamID  *uint     `db:"macos_default_team_id" json:"-"`
	IOSDefaultTeamID    *uint     `db:"ios_default_team_id" json:"-"`
	IPadOSDefaultTeamID *uint     `db:"ipados_default_team_id" json:"-"`
	EncryptedToken      []byte    `db:"token" json:"-"`

	// MDMServerURL is not a database field, it is computed from the AppConfig's
	// Server URL and the static path to the MDM endpoint (using
	// apple_mdm.ResolveAppleMDMURL).
	MDMServerURL string `db:"-" json:"mdm_server_url"`

	// the following fields are not in the abm_tokens table, they must be queried
	// by a LEFT JOIN on the corresponding team, coalesced to "No team" if
	// null (no team).
	MacOSTeamName  string `db:"macos_team" json:"-"`
	IOSTeamName    string `db:"ios_team" json:"-"`
	IPadOSTeamName string `db:"ipados_team" json:"-"`

	// These fields are composed of the ID and name fields above, and are used in API responses.
	MacOSTeam  ABMTokenTeam `json:"macos_team"`
	IOSTeam    ABMTokenTeam `json:"ios_team"`
	IPadOSTeam ABMTokenTeam `json:"ipados_team"`
}

type ABMTokenTeam struct {
	Name string `json:"name"`
	ID   uint   `json:"team_id"`
}

type AppleCSR struct {
	// NOTE: []byte automatically JSON-encodes as a base64-encoded string
	APNsKey  []byte `json:"apns_key"`
	SCEPCert []byte `json:"scep_cert"`
	SCEPKey  []byte `json:"scep_key"`
}

func (a AppleCSR) AuthzType() string {
	return "mdm_apple"
}

// ABMTermsUpdater is the minimal interface required to get and update the
// AppConfig, and set an ABM token's terms_expired flag as required to handle
// the DEP API errors to indicate that Apple's terms have changed and must be
// accepted. The Fleet Datastore satisfies this interface.
type ABMTermsUpdater interface {
	AppConfig(ctx context.Context) (*AppConfig, error)
	SaveAppConfig(ctx context.Context, info *AppConfig) error
	SetABMTokenTermsExpiredForOrgName(ctx context.Context, orgName string, expired bool) (wasSet bool, err error)
	CountABMTokensWithTermsExpired(ctx context.Context) (int, error)
}

// MDMIdPAccount contains account information of a third-party IdP that can be
// later used for MDM operations like creating local accounts.
type MDMIdPAccount struct {
	UUID     string
	Username string
	Fullname string
	Email    string
}

type MDMAppleBootstrapPackage struct {
	Name      string    `json:"name"`
	TeamID    uint      `json:"team_id" db:"team_id"`
	Bytes     []byte    `json:"bytes,omitempty" db:"bytes"`
	Sha256    []byte    `json:"sha256" db:"sha256"`
	Token     string    `json:"token"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"-" db:"updated_at"`
}

func (bp MDMAppleBootstrapPackage) AuthzType() string {
	return "mdm_apple_bootstrap_package"
}

func (bp *MDMAppleBootstrapPackage) URL(host string) (string, error) {
	pkgURL, err := url.Parse(host)
	if err != nil {
		return "", err
	}
	pkgURL.Path = "/api/latest/fleet/mdm/bootstrap"
	pkgURL.RawQuery = fmt.Sprintf("token=%s", bp.Token)
	return pkgURL.String(), nil
}

// MDMEULA represents an EULA (End User License Agreement) file.
type MDMEULA struct {
	Name      string    `json:"name"`
	Bytes     []byte    `json:"bytes"`
	Token     string    `json:"token"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

func (e MDMEULA) AuthzType() string {
	return "mdm_apple"
}

// ExpectedMDMProfile represents an MDM profile that is expected to be installed on a host.
type ExpectedMDMProfile struct {
	// Identifier is the unique identifier used by macOS profiles
	Identifier string `db:"identifier"`
	// Name is the unique name used by Windows profiles
	Name string `db:"name"`
	// EarliestInstallDate is the earliest updated_at of all team profiles with the same checksum.
	// It is used to assess the case where a host has installed a profile with the identifier
	// expected by the host's current team, but the host's install_date is earlier than the
	// updated_at expected by the host's current. This can happen, for example, if a host is
	// transferred to a team created after the host installed the profile. To avoid treating this as
	// a missing profile, we use the earliest_updated_at of all profiles with the same checksum.
	// Ideally, we would simply compare the checksums of the installed and expected profiles, but
	// the checksums are not available in the osquery profiles table.
	EarliestInstallDate time.Time `db:"earliest_install_date"`
	// RawProfile contains the raw profile contents
	RawProfile []byte `db:"raw_profile"`
	// CountProfileLabels is used to enable queries that filter based on profile <-> label mappings.
	CountProfileLabels uint `db:"count_profile_labels"`
	// CountHostLabels is used to enable queries that filter based on profile <-> label mappings.
	CountHostLabels uint `db:"count_host_labels"`
	// CountNonBrokenLabels is used to enable queries that filter based on profile <-> label mappings.
	CountNonBrokenLabels uint `db:"count_non_broken_labels"`
}

// IsWithinGracePeriod returns true if the host is within the grace period for the profile.
//
// The grace period is defined as 1 hour after the profile was updated. It is checked against the
// host's detail_updated_at timestamp to allow for the host to check in at least once before the
// profile is considered failed. If the host is online, it should report detail queries hourly by
// default. If the host is offline, it should report detail queries shortly after it comes back
// online.
//
// Note: The host detail timestamp is updated after the current set is ingested
// see https://github.com/fleetdm/fleet/blob/e9fd28717d474668ca626efbacdd0615d42b2e0a/server/service/osquery.go#L950
func (ep ExpectedMDMProfile) IsWithinGracePeriod(hostDetailUpdatedAt time.Time) bool {
	gracePeriod := 1 * time.Hour
	return hostDetailUpdatedAt.Before(ep.EarliestInstallDate.Add(gracePeriod))
}

// HostMDMProfileRetryCount represents the number of times Fleet has attempted to install
// the identified profile on a host.
type HostMDMProfileRetryCount struct {
	// Identifier is the unique identifier used by macOS profiles
	ProfileIdentifier string `db:"profile_identifier"`
	// ProfileName is the unique name used by Windows profiles
	ProfileName string `db:"profile_name"`
	Retries     uint   `db:"retries"`
}

// TeamIDSetter defines the method to set a TeamID value on a struct,
// which helps define authorization helpers based on teams.
type TeamIDSetter interface {
	SetTeamID(tid *uint)
}

// CommandEnqueueResult is the result of a command execution on enrolled Apple devices.
type CommandEnqueueResult struct {
	// CommandUUID is the unique identifier for the command.
	CommandUUID string `json:"command_uuid,omitempty"`
	// RequestType is the name of the command.
	RequestType string `json:"request_type,omitempty"`
	// FailedUUIDs is the list of host UUIDs that failed to receive the command.
	FailedUUIDs []string `json:"failed_uuids,omitempty"`
	// Platform is the platform of the hosts targeted by the command.
	// Current possible values are "darwin" or "windows".
	// Here "darwin" means "Apple" devices (iOS/iPadOS/macOS).
	Platform string `json:"platform"`
}

// MDMCommandAuthz is used to check user authorization to read/write an
// MDM command.
type MDMCommandAuthz struct {
	TeamID *uint `json:"team_id"` // required for authorization by team
}

// SetTeamID implements the TeamIDSetter interface.
func (m *MDMCommandAuthz) SetTeamID(tid *uint) {
	m.TeamID = tid
}

// AuthzType implements authz.AuthzTyper.
func (m MDMCommandAuthz) AuthzType() string {
	return "mdm_command"
}

// MDMCommandResult holds the result of a command execution provided by
// the target device.
type MDMCommandResult struct {
	// HostUUID is the MDM enrollment ID. Note: For Windows devices, host uuid is distinct from
	// device id.
	HostUUID string `json:"host_uuid" db:"host_uuid"`
	// CommandUUID is the unique identifier of the command.
	CommandUUID string `json:"command_uuid" db:"command_uuid"`
	// Status is the command status. One of Acknowledged, Error, or NotNow for
	// Apple, or 200, 400, etc for Windows.
	Status string `json:"status" db:"status"`
	// UpdatedAt is the last update timestamp of the command result.
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
	// RequestType is the command's request type, which is basically the
	// command name.
	RequestType string `json:"request_type" db:"request_type"`
	// Result is the original command result XML plist. If the status is Error, it will include the
	// ErrorChain key with more information.
	Result []byte `json:"result" db:"result"`
	// Hostname is not filled by the query, it is filled in the service layer
	// afterwards. To make that explicit, the db field tag is explicitly ignored.
	Hostname string `json:"hostname" db:"-"`
	// Payload is the contents of the command
	Payload []byte `json:"payload" db:"payload"`
}

// MDMCommand represents an MDM command that has been enqueued for
// execution.
type MDMCommand struct {
	// HostUUID is the UUID of the host targeted by the command.
	HostUUID string `json:"host_uuid" db:"host_uuid"`
	// CommandUUID is the unique identifier of the command.
	CommandUUID string `json:"command_uuid" db:"command_uuid"`
	// UpdatedAt is the last update timestamp of the command result.
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
	// RequestType is the command's request type, which is basically the
	// command name.
	RequestType string `json:"request_type" db:"request_type"`
	// Status is the command status. One of Pending, Acknowledged, Error, or NotNow.
	Status string `json:"status" db:"status"`
	// Hostname is the hostname of the host that executed the command.
	Hostname string `json:"hostname" db:"hostname"`
	// TeamID is the host's team, null if the host is in no team. This is used
	// to authorize the user to see the command, it is not returned as part of
	// the response payload.
	TeamID *uint `json:"-" db:"team_id"`
}

// MDMCommandListOptions defines the options to control the list of MDM
// Commands to return. Although it only supports the standard list
// options for now, in the future we expect to add filtering options.
//
// https://github.com/fleetdm/fleet/issues/11008#issuecomment-1503466119
type MDMCommandListOptions struct {
	ListOptions
	Filters MDMCommandFilters
}

type MDMCommandFilters struct {
	HostIdentifier string
	RequestType    string
}

type MDMPlatformsCounts struct {
	MacOS   uint `db:"macos" json:"macos"`
	Windows uint `db:"windows" json:"windows"`
	Linux   uint `db:"linux" json:"linux"`
}

type MDMDiskEncryptionSummary struct {
	Verified            MDMPlatformsCounts `db:"verified" json:"verified"`
	Verifying           MDMPlatformsCounts `db:"verifying" json:"verifying"`
	ActionRequired      MDMPlatformsCounts `db:"action_required" json:"action_required"`
	Enforcing           MDMPlatformsCounts `db:"enforcing" json:"enforcing"`
	Failed              MDMPlatformsCounts `db:"failed" json:"failed"`
	RemovingEnforcement MDMPlatformsCounts `db:"removing_enforcement" json:"removing_enforcement"`
}

// MDMProfilesSummary reports the number of hosts being managed with configuration
// profiles and/or disk encryption. Each host may be counted in only one of four mutually-exclusive categories:
// Failed, Pending, Verifying, or Verified.
type MDMProfilesSummary struct {
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

// HostMDMProfile is the status of an MDM profile on a host. It can be used to represent either
// a Windows or macOS profile.
type HostMDMProfile struct {
	HostUUID      string             `db:"-" json:"-"`
	CommandUUID   string             `db:"-" json:"-"`
	ProfileUUID   string             `db:"-" json:"profile_uuid"`
	Name          string             `db:"-" json:"name"`
	Identifier    string             `db:"-" json:"-"`
	Status        *MDMDeliveryStatus `db:"-" json:"status"`
	OperationType MDMOperationType   `db:"-" json:"operation_type"`
	Detail        string             `db:"-" json:"detail"`
	Platform      string             `db:"-" json:"platform"`
}

// MDMDeliveryStatus is the status of an MDM command to apply a profile
// to a device (whether it is installing or removing).
type MDMDeliveryStatus string

// List of possible MDMDeliveryStatus values. For a given host, the status
// of a profile can be either of those, or NULL. The meaning of the status is
// as follows:
//
//   - failed: the MDM command failed to apply, and it won't retry. This is
//     currently a terminal state. TODO(mna): for macOS currently we only retry if the
//     command failed to enqueue in ReconcileProfile (it resets the status to
//     NULL). A failure in the asynchronous actual response of the MDM command
//     (via MDMAppleCheckinAndCommandService.CommandAndReportResults) results in
//     a retry of mdm.MaxProfileRetries times and if it still reports as failed
//     it will be set to failed permanently.
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
//     the summary endpoints/functions or for the list hosts filter, a NULL
//     status is equivalent to a Pending status.
var (
	MDMDeliveryFailed    MDMDeliveryStatus = "failed"
	MDMDeliveryVerified  MDMDeliveryStatus = "verified"
	MDMDeliveryVerifying MDMDeliveryStatus = "verifying"
	MDMDeliveryPending   MDMDeliveryStatus = "pending"
)

type MDMOperationType string

const (
	MDMOperationTypeInstall MDMOperationType = "install"
	MDMOperationTypeRemove  MDMOperationType = "remove"
)

// MDMConfigProfileAuthz is used to check user authorization to read/write an
// MDM configuration profile.
type MDMConfigProfileAuthz struct {
	TeamID *uint `json:"team_id"` // required for authorization by team
}

// AuthzType implements authz.AuthzTyper.
func (m MDMConfigProfileAuthz) AuthzType() string {
	return "mdm_config_profile"
}

// MDMConfigProfilePayload is the platform-agnostic struct returned by
// endpoints that return MDM configuration profiles (get/list profiles).
type MDMConfigProfilePayload struct {
	ProfileUUID      string                      `json:"profile_uuid" db:"profile_uuid"`
	TeamID           *uint                       `json:"team_id" db:"team_id"` // null for no-team
	Name             string                      `json:"name" db:"name"`
	Platform         string                      `json:"platform" db:"platform"`               // "windows" or "darwin"
	Identifier       string                      `json:"identifier,omitempty" db:"identifier"` // only set for macOS
	Checksum         []byte                      `json:"checksum,omitempty" db:"checksum"`     // only set for macOS
	CreatedAt        time.Time                   `json:"created_at" db:"created_at"`
	UploadedAt       time.Time                   `json:"updated_at" db:"uploaded_at"` // NOTE: JSON field is still `updated_at` for historical reasons, would be an API breaking change
	LabelsIncludeAll []ConfigurationProfileLabel `json:"labels_include_all,omitempty" db:"-"`
	LabelsIncludeAny []ConfigurationProfileLabel `json:"labels_include_any,omitempty" db:"-"`
	LabelsExcludeAny []ConfigurationProfileLabel `json:"labels_exclude_any,omitempty" db:"-"`
}

// MDMProfileBatchPayload represents the payload to batch-set the profiles for
// a team or no-team.
type MDMProfileBatchPayload struct {
	Name     string `json:"name,omitempty"`
	Contents []byte `json:"contents,omitempty"`

	// Deprecated: Labels is the backwards-compatible way of specifying
	// LabelsIncludeAll.
	Labels           []string   `json:"labels,omitempty"`
	LabelsIncludeAll []string   `json:"labels_include_all,omitempty"`
	LabelsIncludeAny []string   `json:"labels_include_any,omitempty"`
	LabelsExcludeAny []string   `json:"labels_exclude_any,omitempty"`
	SecretsUpdatedAt *time.Time `json:"-"`
}

func NewMDMConfigProfilePayloadFromWindows(cp *MDMWindowsConfigProfile) *MDMConfigProfilePayload {
	var tid *uint
	if cp.TeamID != nil && *cp.TeamID > 0 {
		tid = cp.TeamID
	}
	return &MDMConfigProfilePayload{
		ProfileUUID:      cp.ProfileUUID,
		TeamID:           tid,
		Name:             cp.Name,
		Platform:         "windows",
		CreatedAt:        cp.CreatedAt,
		UploadedAt:       cp.UploadedAt,
		LabelsIncludeAll: cp.LabelsIncludeAll,
		LabelsIncludeAny: cp.LabelsIncludeAny,
		LabelsExcludeAny: cp.LabelsExcludeAny,
	}
}

func NewMDMConfigProfilePayloadFromApple(cp *MDMAppleConfigProfile) *MDMConfigProfilePayload {
	var tid *uint
	if cp.TeamID != nil && *cp.TeamID > 0 {
		tid = cp.TeamID
	}
	return &MDMConfigProfilePayload{
		ProfileUUID:      cp.ProfileUUID,
		TeamID:           tid,
		Name:             cp.Name,
		Identifier:       cp.Identifier,
		Platform:         "darwin",
		Checksum:         cp.Checksum,
		CreatedAt:        cp.CreatedAt,
		UploadedAt:       cp.UploadedAt,
		LabelsIncludeAll: cp.LabelsIncludeAll,
		LabelsIncludeAny: cp.LabelsIncludeAny,
		LabelsExcludeAny: cp.LabelsExcludeAny,
	}
}

func NewMDMConfigProfilePayloadFromAppleDDM(decl *MDMAppleDeclaration) *MDMConfigProfilePayload {
	var tid *uint
	if decl.TeamID != nil && *decl.TeamID > 0 {
		tid = decl.TeamID
	}
	return &MDMConfigProfilePayload{
		ProfileUUID:      decl.DeclarationUUID,
		TeamID:           tid,
		Name:             decl.Name,
		Identifier:       decl.Identifier,
		Platform:         "darwin",
		Checksum:         []byte(decl.Checksum),
		CreatedAt:        decl.CreatedAt,
		UploadedAt:       decl.UploadedAt,
		LabelsIncludeAll: decl.LabelsIncludeAll,
		LabelsIncludeAny: decl.LabelsIncludeAny,
		LabelsExcludeAny: decl.LabelsExcludeAny,
	}
}

// MDMProfileSpec represents the spec used to define configuration
// profiles via yaml files.
type MDMProfileSpec struct {
	Path string `json:"path,omitempty"`

	// Deprecated: the Labels field is now deprecated, it is superseded by
	// LabelsIncludeAll, so any value set via this field will be transferred to
	// LabelsIncludeAll.
	Labels []string `json:"labels,omitempty"`

	// LabelsIncludeAll is a list of label names that the host must be a member
	// of in order to receive the profile. It must be a member of all listed
	// labels.
	LabelsIncludeAll []string `json:"labels_include_all,omitempty"`
	// LabelsIncludeAny is a list of label names that the host must be a member
	// of in order to receive the profile. It may be a member of
	// any listed labels.
	LabelsIncludeAny []string `json:"labels_include_any,omitempty"`
	// LabelsExcludeAll is a list of label names that the host must not be a
	// member of in order to receive the profile. It must not be a member of any
	// of the listed labels.
	LabelsExcludeAny []string `json:"labels_exclude_any,omitempty"`
}

// UnmarshalJSON implements the json.Unmarshaler interface to add backwards
// compatibility to previous ways to define profile specs.
func (p *MDMProfileSpec) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	if lookAhead := bytes.TrimSpace(data); len(lookAhead) > 0 && lookAhead[0] == '"' {
		var backwardsCompat string
		if err := json.Unmarshal(data, &backwardsCompat); err != nil {
			return fmt.Errorf("unmarshal profile spec. Error using old format: %w", err)
		}
		p.Path = backwardsCompat

		// FIXME: equivalent of no label condition, should clear all labels slice?
		// p.Labels = nil
		// p.LabelsIncludeAll = nil
		// p.LabelsIncludeAny = nil
		// p.LabelsExcludeAny = nil
		return nil
	}

	// use an alias type to avoid recursively calling this function forever.
	type Alias MDMProfileSpec
	var aliasData Alias
	if err := json.Unmarshal(data, &aliasData); err != nil {
		return fmt.Errorf("unmarshal profile spec. Error using new format: %w", err)
	}
	// NOTE: we always want the newly unmarshaled profile spec to completely replace the old one
	// (rather than merging the new data into the old one).
	*p = MDMProfileSpec(aliasData)
	return nil
}

func (p *MDMProfileSpec) Clone() (Cloner, error) {
	return p.Copy(), nil
}

func (p *MDMProfileSpec) Copy() *MDMProfileSpec {
	if p == nil {
		return nil
	}

	clone := *p

	if len(p.Labels) > 0 {
		clone.Labels = make([]string, len(p.Labels))
		copy(clone.Labels, p.Labels)
	}
	if len(p.LabelsIncludeAll) > 0 {
		clone.LabelsIncludeAll = make([]string, len(p.LabelsIncludeAll))
		copy(clone.LabelsIncludeAll, p.LabelsIncludeAll)
	}
	if len(p.LabelsIncludeAny) > 0 {
		clone.LabelsIncludeAny = make([]string, len(p.LabelsIncludeAny))
		copy(clone.LabelsIncludeAny, p.LabelsIncludeAny)
	}
	if len(p.LabelsExcludeAny) > 0 {
		clone.LabelsExcludeAny = make([]string, len(p.LabelsExcludeAny))
		copy(clone.LabelsExcludeAny, p.LabelsExcludeAny)
	}

	return &clone
}

func labelCountMap(labels []string) map[string]int {
	counts := make(map[string]int)
	for _, label := range labels {
		counts[label]++
	}
	return counts
}

// MDMProfileSpecsMatch match checks if two slices contain the same spec
// elements, regardless of order.
func MDMProfileSpecsMatch(a, b []MDMProfileSpec) bool {
	if len(a) != len(b) {
		return false
	}

	pathLabelIncludeCounts := make(map[string]map[string]int)
	for _, v := range a {
		// the deprecated Labels field is only relevant if LabelsIncludeAll is
		// empty.
		if len(v.LabelsIncludeAll) > 0 {
			pathLabelIncludeCounts[v.Path] = labelCountMap(v.LabelsIncludeAll)
		} else {
			pathLabelIncludeCounts[v.Path] = labelCountMap(v.Labels)
		}
	}
	pathLabelsIncludeAnyCounts := make(map[string]map[string]int)
	for _, v := range a {
		pathLabelsIncludeAnyCounts[v.Path] = labelCountMap(v.LabelsIncludeAny)
	}
	pathLabelExcludeCounts := make(map[string]map[string]int)
	for _, v := range a {
		pathLabelExcludeCounts[v.Path] = labelCountMap(v.LabelsExcludeAny)
	}

	for _, v := range b {
		includeLabels, okIncl := pathLabelIncludeCounts[v.Path]
		includeAnyLabels, okInclAny := pathLabelsIncludeAnyCounts[v.Path]
		excludeLabels, okExcl := pathLabelExcludeCounts[v.Path]
		if !okIncl || !okExcl || !okInclAny {
			return false
		}

		var bLabelIncludeCounts map[string]int
		if len(v.LabelsIncludeAll) > 0 {
			bLabelIncludeCounts = labelCountMap(v.LabelsIncludeAll)
		} else {
			bLabelIncludeCounts = labelCountMap(v.Labels)
		}
		for label, count := range bLabelIncludeCounts {
			if includeLabels[label] != count {
				return false
			}
			includeLabels[label] -= count
		}
		for _, count := range includeLabels {
			if count != 0 {
				return false
			}
		}

		bLabelIncludeAnyCounts := labelCountMap(v.LabelsIncludeAny)
		for label, count := range bLabelIncludeAnyCounts {
			if includeAnyLabels[label] != count {
				return false
			}
			includeAnyLabels[label] -= count
		}
		for _, count := range includeAnyLabels {
			if count != 0 {
				return false
			}
		}

		bLabelExcludeCounts := labelCountMap(v.LabelsExcludeAny)
		for label, count := range bLabelExcludeCounts {
			if excludeLabels[label] != count {
				return false
			}
			excludeLabels[label] -= count
		}
		for _, count := range excludeLabels {
			if count != 0 {
				return false
			}
		}

		delete(pathLabelIncludeCounts, v.Path)
		delete(pathLabelsIncludeAnyCounts, v.Path)
		delete(pathLabelExcludeCounts, v.Path)
	}

	return len(pathLabelIncludeCounts) == 0 && len(pathLabelsIncludeAnyCounts) == 0 && len(pathLabelExcludeCounts) == 0
}

type MDMLabelsMode string

const (
	LabelsIncludeAll MDMLabelsMode = "labels_include_all"
	LabelsIncludeAny MDMLabelsMode = "labels_include_any"
	LabelsExcludeAny MDMLabelsMode = "labels_exclude_any"
)

type MDMAssetName string

const (
	// MDMAssetCACert is the name of the root CA certificate used by MDM, for
	// Apple this is the SCEP certificate, for Windows the WSTEP certificate
	MDMAssetCACert MDMAssetName = "ca_cert"
	// MDMAssetCAKey is the name of the root CA private key used by MDM, for
	// Apple this is the SCEP key, for Windows the WSTEP key
	MDMAssetCAKey MDMAssetName = "ca_key"
	// MDMAssetAPNSKey is the name of the APNs (Apple Push Notifications
	// service) private key used by MDM
	MDMAssetAPNSKey MDMAssetName = "apns_key"
	// MDMAssetAPNSCert is the name of the APNs (Apple Push Notifications
	// service) private key used by MDM
	MDMAssetAPNSCert MDMAssetName = "apns_cert"
	// MDMAssetABMKey is the name of the ABM (Apple Business Manager)
	// private key used to decrypt MDMAssetABMToken
	MDMAssetABMKey MDMAssetName = "abm_key"
	// MDMAssetABMCert is the name of the ABM (Apple Business Manager)
	// private key used to encrypt MDMAssetABMToken
	MDMAssetABMCert MDMAssetName = "abm_cert"
	// MDMAssetABMTokenDeprecated is an encrypted JSON file that contains a token
	// that can be used for the authentication process with the ABM API.
	// Deprecated: ABM tokens are now stored in the abm_tokens table, they are
	// not in mdm_config_assets anymore.
	MDMAssetABMTokenDeprecated MDMAssetName = "abm_token"
	// MDMAssetSCEPChallenge defines the shared secret used to issue SCEP
	// certificatges to Apple devices.
	MDMAssetSCEPChallenge MDMAssetName = "scep_challenge"
	// MDMAssetVPPTokenDeprecated is the name of the token used by MDM to
	// authenticate to Apple's VPP service.
	// Deprecated: VPP tokens are now stored in the vpp_tokens table, they are
	// not in mdm_config_assets anymore.
	MDMAssetVPPTokenDeprecated MDMAssetName = "vpp_token"
	// MDMAssetNDESPassword is the password used to retrieve SCEP challenge from
	// NDES SCEP server. It is used by Fleet's SCEP proxy.
	MDMAssetNDESPassword MDMAssetName = "ndes_password"
)

type MDMConfigAsset struct {
	Name        MDMAssetName `db:"name"`
	Value       []byte       `db:"value"`
	MD5Checksum string       `db:"md5_checksum"`
}

func (m MDMConfigAsset) Clone() (Cloner, error) {
	return m.Copy(), nil
}

func (m MDMConfigAsset) Copy() MDMConfigAsset {
	var clone MDMConfigAsset

	clone.Name = m.Name
	clone.MD5Checksum = m.MD5Checksum

	if len(m.Value) > 0 {
		clone.Value = make([]byte, len(m.Value))
		copy(clone.Value, m.Value)
	}

	return clone
}

// MDMPlatform returns "darwin" or "windows" as MDM platforms
// derived from a host's platform (hosts.platform field).
//
// Note that "darwin" as MDM platform means Apple (we keep it as "darwin"
// to keep backwards compatibility throughout the app).
func MDMPlatform(hostPlatform string) string {
	switch hostPlatform {
	case "darwin", "ios", "ipados":
		return "darwin"
	case "windows":
		return "windows"
	}
	return ""
}

// MDMSupported returns whether MDM is supported for a given host platform.
func MDMSupported(hostPlatform string) bool {
	return MDMPlatform(hostPlatform) != ""
}

// FilterMacOSOnlyProfilesFromIOSIPadOS will filter out profiles that are only for macOS devices
// if the profile target's platform is ios/ipados.
func FilterMacOSOnlyProfilesFromIOSIPadOS(profiles []*MDMAppleProfilePayload) []*MDMAppleProfilePayload {
	i := 0
	for _, profilePayload := range profiles {
		if (profilePayload.HostPlatform == "ios" || profilePayload.HostPlatform == "ipados") &&
			(profilePayload.ProfileName == mdm_types.FleetdConfigProfileName ||
				profilePayload.ProfileName == mdm_types.FleetFileVaultProfileName) {
			continue
		}
		profiles[i] = profilePayload
		i++
	}
	return profiles[:i]
}

// RefetchBaseCommandUUIDPrefix and below command prefixes are the prefixes used for MDM commands used to refetch information from iOS/iPadOS devices.
const (
	RefetchBaseCommandUUIDPrefix   = "REFETCH-"
	RefetchDeviceCommandUUIDPrefix = RefetchBaseCommandUUIDPrefix + "DEVICE-"
	RefetchAppsCommandUUIDPrefix   = RefetchBaseCommandUUIDPrefix + "APPS-"
)

// VPPTokenInfo is the representation of the VPP token that we send out via API.
type VPPTokenInfo struct {
	OrgName   string `json:"org_name"`
	RenewDate string `json:"renew_date"`
	Location  string `json:"location"`
}

// VPPTokenRaw is the representation of the decoded JSON object that is downloaded from ABM.
type VPPTokenRaw struct {
	OrgName string `json:"orgName"`
	Token   string `json:"token"`
	ExpDate string `json:"expDate"`
}

// VPPTokenData is the VPP data we store in the DB.
type VPPTokenData struct {
	// Location comes from an Apple API:
	// https://developer.apple.com/documentation/devicemanagement/client_config. It is the name of
	// the "library" of apps in ABM that is associated with this VPP token.
	Location string `json:"location"`

	// Token is the token that is downloaded from ABM. It is a base64 encoded JSON object with the
	// structure of `VPPTokenRaw`.
	Token string `json:"token"`
}

const VPPTimeFormat = "2006-01-02T15:04:05Z0700"

// VPPTokenDB represents a VPP token record in the DB
type VPPTokenDB struct {
	ID        uint      `db:"id" json:"id"`
	OrgName   string    `db:"organization_name" json:"org_name"`
	Location  string    `db:"location" json:"location"`
	RenewDate time.Time `db:"renew_at" json:"renew_date"`
	// Token is the token dowloaded from ABM. It is the base64 encoded
	// JSON object with the structure of `VPPTokenRaw`
	Token string      `db:"token" json:"-"`
	Teams []TeamTuple `json:"teams"`
	// CreatedAt    time.Time `json:"created_at" db:"created_at"`
	// UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

type TeamTuple struct {
	ID   uint   `json:"team_id"`
	Name string `json:"name"`
}

type NullTeamType string

const (
	// VPP token is inactive, only valid option if teamID is set.
	NullTeamNone NullTeamType = "none"
	// VPP token is available for all teams.
	NullTeamAllTeams NullTeamType = "allteams"
	// VPP token is available only for "No team" team.
	NullTeamNoTeam NullTeamType = "noteam"
)

func (n NullTeamType) PrettyName() string {
	switch n {
	case NullTeamAllTeams:
		return ReservedNameAllTeams
	case NullTeamNoTeam:
		return ReservedNameNoTeam
	default:
		return string(n)
	}
}

type AppleDevice int

const (
	MacOS AppleDevice = iota
	IOS
	IPadOS
)

type AppleDevicePlatform string

const (
	MacOSPlatform  AppleDevicePlatform = "darwin"
	IOSPlatform    AppleDevicePlatform = "ios"
	IPadOSPlatform AppleDevicePlatform = "ipados"
)

var VPPAppsPlatforms = []AppleDevicePlatform{IOSPlatform, IPadOSPlatform, MacOSPlatform}

type AppleDevicesToRefetch struct {
	HostID              uint                   `db:"host_id"`
	UUID                string                 `db:"uuid"`
	CommandsAlreadySent MDMCommandsAlreadySent `db:"commands_already_sent"`
}

type MDMCommandsAlreadySent []string

func (c *MDMCommandsAlreadySent) Scan(src interface{}) error {
	if src == nil {
		return nil
	}

	raw, ok := src.([]byte)
	if !ok {
		return fmt.Errorf("unexpected type for MDMCommandsAlreadySent: %T", src)
	}
	// Filter out [null] command types which MySQL returns when there are no commands_already_sent.
	// For details, see: https://dev.mysql.com/doc/refman/8.4/en/aggregate-functions.html#function_json-arrayagg
	if string(raw) == "[null]" {
		*c = nil
		return nil
	}

	var commands MDMCommandsAlreadySent
	if err := json.Unmarshal(raw, &commands); err != nil {
		return err
	}
	*c = commands
	return nil
}

type HostMDMCommand struct {
	HostID      uint   `db:"host_id"`
	CommandType string `db:"command_type"`
}
