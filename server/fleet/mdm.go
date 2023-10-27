package fleet

import (
	"context"
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
	pkgURL.Path = "/api/latest/fleet/mdm/apple/bootstrap"
	pkgURL.RawQuery = fmt.Sprintf("token=%s", bp.Token)
	return pkgURL.String(), nil
}

// MDMAppleEULA represents an EULA (End User License Agreement) file.
type MDMAppleEULA struct {
	Name      string    `json:"name"`
	Bytes     []byte    `json:"bytes"`
	Token     string    `json:"token"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

func (e MDMAppleEULA) AuthzType() string {
	return "mdm_apple"
}

// ExpectedMDMProfile represents an MDM profile that is expected to be installed on a host.
type ExpectedMDMProfile struct {
	Identifier string `db:"identifier"`
	// EarliestInstallDate is the earliest updated_at of all team profiles with the same checksum.
	// It is used to assess the case where a host has installed a profile with the identifier
	// expected by the host's current team, but the host's install_date is earlier than the
	// updated_at expected by the host's current. This can happen, for example, if a host is
	// transferred to a team created after the host installed the profile. To avoid treating this as
	// a missing profile, we use the earliest_updated_at of all profiles with the same checksum.
	// Ideally, we would simply compare the checksums of the installed and expected profiles, but
	// the checksums are not available in the osquery profiles table.
	EarliestInstallDate time.Time `db:"earliest_install_date"`
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
	ProfileIdentifier string `db:"profile_identifier"`
	Retries           uint   `db:"retries"`
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
	// Status is the command status. One of Acknowledged, Error, or NotNow.
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
}
