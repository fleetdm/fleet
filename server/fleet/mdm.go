package fleet

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"time"
)

const (
	MDMPlatformApple     = "apple"
	MDMPlatformMicrosoft = "microsoft"
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

type AppleCSR struct {
	// NOTE: []byte automatically JSON-encodes as a base64-encoded string
	APNsKey  []byte `json:"apns_key"`
	SCEPCert []byte `json:"scep_cert"`
	SCEPKey  []byte `json:"scep_key"`
}

func (a AppleCSR) AuthzType() string {
	return "mdm_apple"
}

// AppConfigUpdated is the minimal interface required to get and update the
// AppConfig, as required to handle the DEP API errors to flag that Apple's
// terms have changed and must be accepted. The Fleet Datastore satisfies
// this interface.
type AppConfigUpdater interface {
	AppConfig(ctx context.Context) (*AppConfig, error)
	SaveAppConfig(ctx context.Context, info *AppConfig) error
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
}

type MDMPlatformsCounts struct {
	MacOS   uint `db:"macos" json:"macos"`
	Windows uint `db:"windows" json:"windows"`
}

type MDMDiskEncryptionSummary struct {
	Verified            MDMPlatformsCounts `db:"verified" json:"verified"`
	Verifying           MDMPlatformsCounts `db:"verifying" json:"verifying"`
	ActionRequired      MDMPlatformsCounts `db:"action_required" json:"action_required"`
	Enforcing           MDMPlatformsCounts `db:"enforcing" json:"enforcing"`
	Failed              MDMPlatformsCounts `db:"failed" json:"failed"`
	RemovingEnforcement MDMPlatformsCounts `db:"removing_enforcement" json:"removing_enforcement"`
}

// MDMProfilesSummary reports the number of hosts being managed with MDM configuration
// profiles. Each host may be counted in only one of four mutually-exclusive categories:
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
	ProfileUUID string                      `json:"profile_uuid" db:"profile_uuid"`
	TeamID      *uint                       `json:"team_id" db:"team_id"` // null for no-team
	Name        string                      `json:"name" db:"name"`
	Platform    string                      `json:"platform" db:"platform"`               // "windows" or "darwin"
	Identifier  string                      `json:"identifier,omitempty" db:"identifier"` // only set for macOS
	Checksum    []byte                      `json:"checksum,omitempty" db:"checksum"`     // only set for macOS
	CreatedAt   time.Time                   `json:"created_at" db:"created_at"`
	UploadedAt  time.Time                   `json:"updated_at" db:"uploaded_at"` // NOTE: JSON field is still `updated_at` for historical reasons, would be an API breaking change
	Labels      []ConfigurationProfileLabel `json:"labels,omitempty" db:"-"`
}

// MDMProfileBatchPayload represents the payload to batch-set the profiles for
// a team or no-team.
type MDMProfileBatchPayload struct {
	Name     string   `json:"name,omitempty"`
	Contents []byte   `json:"contents,omitempty"`
	Labels   []string `json:"labels,omitempty"`
}

func NewMDMConfigProfilePayloadFromWindows(cp *MDMWindowsConfigProfile) *MDMConfigProfilePayload {
	var tid *uint
	if cp.TeamID != nil && *cp.TeamID > 0 {
		tid = cp.TeamID
	}
	return &MDMConfigProfilePayload{
		ProfileUUID: cp.ProfileUUID,
		TeamID:      tid,
		Name:        cp.Name,
		Platform:    "windows",
		CreatedAt:   cp.CreatedAt,
		UploadedAt:  cp.UploadedAt,
		Labels:      cp.Labels,
	}
}

func NewMDMConfigProfilePayloadFromApple(cp *MDMAppleConfigProfile) *MDMConfigProfilePayload {
	var tid *uint
	if cp.TeamID != nil && *cp.TeamID > 0 {
		tid = cp.TeamID
	}
	return &MDMConfigProfilePayload{
		ProfileUUID: cp.ProfileUUID,
		TeamID:      tid,
		Name:        cp.Name,
		Identifier:  cp.Identifier,
		Platform:    "darwin",
		Checksum:    cp.Checksum,
		CreatedAt:   cp.CreatedAt,
		UploadedAt:  cp.UploadedAt,
		Labels:      cp.Labels,
	}
}

func NewMDMConfigProfilePayloadFromAppleDDM(decl *MDMAppleDeclaration) *MDMConfigProfilePayload {
	var tid *uint
	if decl.TeamID != nil && *decl.TeamID > 0 {
		tid = decl.TeamID
	}
	return &MDMConfigProfilePayload{
		ProfileUUID: decl.DeclarationUUID,
		TeamID:      tid,
		Name:        decl.Name,
		Identifier:  decl.Identifier,
		Platform:    "darwin",
		Checksum:    []byte(decl.Checksum),
		CreatedAt:   decl.CreatedAt,
		UploadedAt:  decl.UploadedAt,
		Labels:      decl.Labels,
	}
}

// MDMProfileSpec represents the spec used to define configuration
// profiles via yaml files.
type MDMProfileSpec struct {
	Path   string   `json:"path,omitempty"`
	Labels []string `json:"labels,omitempty"`
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
		return nil
	}

	// use an alias type to avoid recursively calling this function forever.
	type Alias MDMProfileSpec
	aliasData := struct {
		*Alias
	}{
		Alias: (*Alias)(p),
	}
	if err := json.Unmarshal(data, &aliasData); err != nil {
		return fmt.Errorf("unmarshal profile spec. Error using new format: %w", err)
	}
	return nil
}

func (p *MDMProfileSpec) Clone() (Cloner, error) {
	return p.Copy(), nil
}

func (p *MDMProfileSpec) Copy() *MDMProfileSpec {
	if p == nil {
		return nil
	}

	var clone MDMProfileSpec
	clone = *p

	if len(p.Labels) > 0 {
		clone.Labels = make([]string, len(p.Labels))
		copy(clone.Labels, p.Labels)
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

	pathLabelCounts := make(map[string]map[string]int)
	for _, v := range a {
		pathLabelCounts[v.Path] = labelCountMap(v.Labels)
	}

	for _, v := range b {
		labels, ok := pathLabelCounts[v.Path]
		if !ok {
			return false
		}

		bLabelCounts := labelCountMap(v.Labels)
		for label, count := range bLabelCounts {
			if labels[label] != count {
				return false
			}
			labels[label] -= count
		}

		for _, count := range labels {
			if count != 0 {
				return false
			}
		}

		delete(pathLabelCounts, v.Path)
	}

	return len(pathLabelCounts) == 0
}
