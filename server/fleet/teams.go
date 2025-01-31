package fleet

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"golang.org/x/text/unicode/norm"
)

const (
	RoleAdmin        = "admin"
	RoleMaintainer   = "maintainer"
	RoleObserver     = "observer"
	RoleObserverPlus = "observer_plus"
	RoleGitOps       = "gitops"
	TeamNameNoTeam   = "No team"
	TeamNameAllTeams = "All teams"
)

const (
	ReservedNameAllTeams = "All teams"
	ReservedNameNoTeam   = "No team"
)

// IsReservedTeamName checks if the name provided is a reserved team name
func IsReservedTeamName(name string) bool {
	normalizedName := norm.NFC.String(name)
	return normalizedName == ReservedNameAllTeams || normalizedName == ReservedNameNoTeam
}

type TeamPayload struct {
	Name               *string              `json:"name"`
	Description        *string              `json:"description"`
	Secrets            []*EnrollSecret      `json:"secrets"`
	WebhookSettings    *TeamWebhookSettings `json:"webhook_settings"`
	Integrations       *TeamIntegrations    `json:"integrations"`
	MDM                *TeamPayloadMDM      `json:"mdm"`
	HostExpirySettings *HostExpirySettings  `json:"host_expiry_settings"`
	// Note AgentOptions must be set by a separate endpoint.
}

// TeamPayloadMDM is a distinct struct than TeamMDM because in ModifyTeam we
// need to be able which part of the MDM config was provided in the request,
// so the fields are pointers to structs.
type TeamPayloadMDM struct {
	EnableDiskEncryption optjson.Bool `json:"enable_disk_encryption"`

	// MacOSUpdates defines the OS update settings for macOS devices.
	MacOSUpdates *AppleOSUpdateSettings `json:"macos_updates"`
	// IOSUpdates defines the OS update settings for iOS devices.
	IOSUpdates *AppleOSUpdateSettings `json:"ios_updates"`
	// IPadOSUpdates defines the OS update settings for iPadOS devices.
	IPadOSUpdates *AppleOSUpdateSettings `json:"ipados_updates"`
	// WindowsUpdates defines the OS update settings for Windows devices.
	WindowsUpdates *WindowsUpdates `json:"windows_updates"`

	MacOSSetup *MacOSSetup `json:"macos_setup"`
}

// Team is the data representation for the "Team" concept (group of hosts and
// group of users that can perform operations on those hosts).
type Team struct {
	// Directly in DB

	// ID is the database ID.
	ID       uint    `json:"id" db:"id"`
	Filename *string `json:"gitops_filename,omitempty" db:"filename"`
	// CreatedAt is the timestamp of the label creation.
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	// Name is the human friendly name of the team.
	Name string `json:"name" db:"name"`
	// Description is an optional description for the team.
	Description string     `json:"description" db:"description"`
	Config      TeamConfig `json:"-" db:"config"` // see json.MarshalJSON/UnmarshalJSON implementations

	// Derived from JOINs

	// UserCount is the count of users with explicit roles on this team.
	UserCount int `json:"user_count" db:"user_count"`
	// Users is the users that have a role on this team.
	Users []TeamUser `json:"users,omitempty"`
	// UserCount is the count of hosts assigned to this team.
	HostCount int `json:"host_count" db:"host_count"`
	// Hosts are the hosts assigned to the team.
	Hosts []Host `json:"hosts,omitempty"`
	// Secrets is the enroll secrets valid for this team.
	Secrets []*EnrollSecret `json:"secrets,omitempty"`
}

func (t Team) MarshalJSON() ([]byte, error) {
	// The reason for not embedding TeamConfig above, is that it also implements sql.Scanner/Valuer.
	// We do not want it be promoted to the parent struct, because it causes issues when using sqlx for scanning.
	// Also need to implement json.Marshaler/Unmarshaler on each type that embeds Team so because it will be promoted
	// to the parent struct.
	x := struct {
		ID          uint            `json:"id"`
		CreatedAt   time.Time       `json:"created_at"`
		Name        string          `json:"name"`
		Description string          `json:"description"`
		TeamConfig                  // inline this using struct embedding
		UserCount   int             `json:"user_count"`
		Users       []TeamUser      `json:"users,omitempty"`
		HostCount   int             `json:"host_count"`
		Hosts       []HostResponse  `json:"hosts,omitempty"`
		Secrets     []*EnrollSecret `json:"secrets,omitempty"`
	}{
		ID:          t.ID,
		CreatedAt:   t.CreatedAt,
		Name:        t.Name,
		Description: t.Description,
		TeamConfig:  t.Config,
		UserCount:   t.UserCount,
		Users:       t.Users,
		HostCount:   t.HostCount,
		Hosts:       HostResponsesForHostsCheap(t.Hosts),
		Secrets:     t.Secrets,
	}

	return json.Marshal(x)
}

func (t *Team) UnmarshalJSON(b []byte) error {
	var x struct {
		ID          uint            `json:"id"`
		CreatedAt   time.Time       `json:"created_at"`
		Name        string          `json:"name"`
		Description string          `json:"description"`
		TeamConfig                  // inline this using struct embedding
		UserCount   int             `json:"user_count"`
		Users       []TeamUser      `json:"users,omitempty"`
		HostCount   int             `json:"host_count"`
		Hosts       []Host          `json:"hosts,omitempty"`
		Secrets     []*EnrollSecret `json:"secrets,omitempty"`
	}

	if err := json.Unmarshal(b, &x); err != nil {
		return err
	}

	if !x.MDM.MacOSSetup.EnableReleaseDeviceManually.Valid {
		x.MDM.MacOSSetup.EnableReleaseDeviceManually = optjson.SetBool(false)
	}
	*t = Team{
		ID:          x.ID,
		CreatedAt:   x.CreatedAt,
		Name:        x.Name,
		Description: x.Description,
		Config:      x.TeamConfig,
		UserCount:   x.UserCount,
		Users:       x.Users,
		HostCount:   x.HostCount,
		Hosts:       x.Hosts,
		Secrets:     x.Secrets,
	}

	return nil
}

type TeamConfig struct {
	// AgentOptions is the options for osquery and Orbit.
	AgentOptions       *json.RawMessage      `json:"agent_options,omitempty"`
	HostExpirySettings HostExpirySettings    `json:"host_expiry_settings"`
	WebhookSettings    TeamWebhookSettings   `json:"webhook_settings"`
	Integrations       TeamIntegrations      `json:"integrations"`
	Features           Features              `json:"features"`
	MDM                TeamMDM               `json:"mdm"`
	Scripts            optjson.Slice[string] `json:"scripts,omitempty"`
	Software           *SoftwareSpec         `json:"software,omitempty"`
}

type TeamWebhookSettings struct {
	// HostStatusWebhook can be nil to match the TeamSpec webhook settings
	HostStatusWebhook      *HostStatusWebhookSettings     `json:"host_status_webhook"`
	FailingPoliciesWebhook FailingPoliciesWebhookSettings `json:"failing_policies_webhook"`
}

type TeamSpecSoftwareAsset struct {
	Path string `json:"path"`
}

type TeamSpecAppStoreApp struct {
	AppStoreID  string `json:"app_store_id"`
	SelfService bool   `json:"self_service"`
}

type TeamMDM struct {
	EnableDiskEncryption bool                  `json:"enable_disk_encryption"`
	MacOSUpdates         AppleOSUpdateSettings `json:"macos_updates"`
	IOSUpdates           AppleOSUpdateSettings `json:"ios_updates"`
	IPadOSUpdates        AppleOSUpdateSettings `json:"ipados_updates"`
	WindowsUpdates       WindowsUpdates        `json:"windows_updates"`
	MacOSSettings        MacOSSettings         `json:"macos_settings"`
	MacOSSetup           MacOSSetup            `json:"macos_setup"`

	WindowsSettings WindowsSettings `json:"windows_settings"`
	// NOTE: TeamSpecMDM must be kept in sync with TeamMDM.

	/////////////////////////////////////////////////////////////////
	// WARNING: If you add to this struct make sure it's taken into
	// account in the TeamMDM Clone implementation!
	/////////////////////////////////////////////////////////////////
}

// Clone implements cloner for TeamMDM.
func (t *TeamMDM) Clone() (Cloner, error) {
	return t.Copy(), nil
}

// Copy returns a deep copy of the TeamMDM.
func (t *TeamMDM) Copy() *TeamMDM {
	if t == nil {
		return nil
	}

	clone := *t

	// EnableDiskEncryption, MacOSUpdates and MacOSSetup don't have fields that
	// require cloning (all fields are basic value types, no
	// pointers/slices/maps).

	if t.MacOSSettings.CustomSettings != nil {
		clone.MacOSSettings.CustomSettings = make([]MDMProfileSpec, len(t.MacOSSettings.CustomSettings))
		for i, mps := range t.MacOSSettings.CustomSettings {
			clone.MacOSSettings.CustomSettings[i] = *mps.Copy()
		}
	}
	if t.MacOSSettings.DeprecatedEnableDiskEncryption != nil {
		clone.MacOSSettings.DeprecatedEnableDiskEncryption = ptr.Bool(*t.MacOSSettings.DeprecatedEnableDiskEncryption)
	}
	if t.WindowsSettings.CustomSettings.Set {
		windowsSettings := make([]MDMProfileSpec, len(t.WindowsSettings.CustomSettings.Value))
		for i, mps := range t.WindowsSettings.CustomSettings.Value {
			windowsSettings[i] = *mps.Copy()
		}
		clone.WindowsSettings.CustomSettings = optjson.SetSlice(windowsSettings)
	}
	if t.MacOSSetup.Software.Set {
		sw := make([]*MacOSSetupSoftware, len(t.MacOSSetup.Software.Value))
		for i, s := range t.MacOSSetup.Software.Value {
			s := *s
			sw[i] = &s
		}
		clone.MacOSSetup.Software = optjson.SetSlice(sw)
	}
	return &clone
}

type TeamSpecMDM struct {
	EnableDiskEncryption optjson.Bool `json:"enable_disk_encryption"`

	// MacOSUpdates defines the OS update settings for macOS devices.
	MacOSUpdates AppleOSUpdateSettings `json:"macos_updates"`
	// IOSUpdates defines the OS update settings for iOS devices.
	IOSUpdates AppleOSUpdateSettings `json:"ios_updates"`
	// IPadOSUpdates defines the OS update settings for iPadOS devices.
	IPadOSUpdates AppleOSUpdateSettings `json:"ipados_updates"`
	// WindowsUpdates defines the OS update settings for Windows devices.
	WindowsUpdates WindowsUpdates `json:"windows_updates"`

	// A map is used for the macos settings so that we can easily detect if its
	// sub-keys were provided or not in an "apply" call. E.g. if the
	// custom_settings key is specified but empty, then we need to clear the
	// value, but if it isn't provided, we need to leave the existing value
	// unmodified.
	MacOSSettings map[string]interface{} `json:"macos_settings"`
	MacOSSetup    MacOSSetup             `json:"macos_setup"`

	WindowsSettings WindowsSettings `json:"windows_settings"`

	// NOTE: TeamMDM must be kept in sync with TeamSpecMDM.
}

// Scan implements the sql.Scanner interface
func (t *TeamConfig) Scan(val interface{}) error {
	switch v := val.(type) {
	case []byte:
		return json.Unmarshal(v, t)
	case string:
		return json.Unmarshal([]byte(v), t)
	case nil: // sql NULL
		return nil
	default:
		return fmt.Errorf("unsupported type: %T", v)
	}
}

// Value implements the sql.Valuer interface
func (t TeamConfig) Value() (driver.Value, error) {
	// force-save as the default `false` value if not set
	if !t.MDM.MacOSSetup.EnableReleaseDeviceManually.Valid {
		t.MDM.MacOSSetup.EnableReleaseDeviceManually = optjson.SetBool(false)
	}
	return json.Marshal(t)
}

type TeamSummary struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (t Team) AuthzType() string {
	return "team"
}

// TeamUser is a user mapped to a team with a role.
type TeamUser struct {
	// User is the user object. At least ID must be specified for most uses.
	User
	// Role is the role the user has for the team.
	Role string `json:"role" db:"role"`
}

var teamRoles = map[string]struct{}{
	RoleAdmin:        {},
	RoleObserver:     {},
	RoleMaintainer:   {},
	RoleObserverPlus: {},
	RoleGitOps:       {},
}

var premiumTeamRoles = map[string]struct{}{
	RoleObserverPlus: {},
	RoleGitOps:       {},
}

// ValidTeamRole returns whether the role provided is valid for a team user.
func ValidTeamRole(role string) bool {
	_, ok := teamRoles[role]
	return ok
}

var globalRoles = map[string]struct{}{
	RoleObserver:     {},
	RoleMaintainer:   {},
	RoleAdmin:        {},
	RoleObserverPlus: {},
	RoleGitOps:       {},
}

var premiumGlobalRoles = map[string]struct{}{
	RoleObserverPlus: {},
	RoleGitOps:       {},
}

// ValidGlobalRole returns whether the role provided is valid for a global user.
func ValidGlobalRole(role string) bool {
	_, ok := globalRoles[role]
	return ok
}

// ValidateRole returns nil if the global and team roles combination is a valid
// one within fleet, or a fleet Error otherwise.
func ValidateRole(globalRole *string, teamUsers []UserTeam) error {
	if globalRole == nil || *globalRole == "" {
		if len(teamUsers) == 0 {
			return NewError(ErrNoRoleNeeded, "either global role or team role needs to be defined")
		}
		for _, t := range teamUsers {
			if !ValidTeamRole(t.Role) {
				return NewErrorf(ErrNoRoleNeeded, "invalid team role: %s", t.Role)
			}
		}
		return nil
	}

	if len(teamUsers) > 0 {
		return NewError(ErrNoRoleNeeded, "Cannot specify both Global Role and Team Roles")
	}

	if !ValidGlobalRole(*globalRole) {
		return NewErrorf(ErrNoRoleNeeded, "invalid global role: %s", *globalRole)
	}

	return nil
}

// ValidateUserRoles verifies the roles to be applied to a new or existing user.
//
// Argument createNew sets whether the user is being created (true) or is being modified (false).
func ValidateUserRoles(createNew bool, payload UserPayload, license LicenseInfo) error {
	var teamUsers_ []UserTeam
	if payload.Teams != nil {
		teamUsers_ = *payload.Teams
	}
	if err := ValidateRole(payload.GlobalRole, teamUsers_); err != nil {
		return err
	}
	premiumRolesPresent := false
	gitOpsRolePresent := false
	if payload.GlobalRole != nil {
		if *payload.GlobalRole == RoleGitOps {
			gitOpsRolePresent = true
		}
		if _, ok := premiumGlobalRoles[*payload.GlobalRole]; ok {
			premiumRolesPresent = true
		}
	}
	for _, teamUser := range teamUsers_ {
		if teamUser.Role == RoleGitOps {
			gitOpsRolePresent = true
		}
		if _, ok := premiumTeamRoles[teamUser.Role]; ok {
			premiumRolesPresent = true
		}
	}
	if !license.IsPremium() && premiumRolesPresent {
		return ErrMissingLicense
	}
	if gitOpsRolePresent &&
		// New user is not API only.
		((createNew && (payload.APIOnly == nil || !*payload.APIOnly)) ||
			// Removing API only status from existing user.
			(!createNew && payload.APIOnly != nil && !*payload.APIOnly)) {
		return NewErrorf(ErrAPIOnlyRole, "role GitOps can only be set for API only users")
	}

	return nil
}

// TeamFilter is the filtering information passed to the datastore for queries
// that may be filtered by team.
type TeamFilter struct {
	// User is the user to filter by.
	User *User
	// IncludeObserver determines whether to include teams the user is an observer on.
	IncludeObserver bool
	// TeamID is the specific team id to filter by. If other criteria are
	// specified, they must met too (e.g. if a User is provided, that team ID
	// must be part of their teams).
	TeamID *uint
}

const (
	TeamKind = "team"
)

type TeamSpec struct {
	Name     string  `json:"name"`
	Filename *string `json:"gitops_filename,omitempty"`

	// We need to distinguish between the agent_options key being present but
	// "empty" or being absent, as we leave the existing agent options unmodified
	// if it is absent, and we clear it if present but empty.
	//
	// If the agent_options key is not provided, the field will be nil (Go nil).
	// If the agent_options key is present but empty in the YAML, will be set to
	// "null" (JSON null). Otherwise, if the key is present and set, it will be
	// set to the agent options JSON object.
	AgentOptions       json.RawMessage         `json:"agent_options,omitempty"` // marshals as "null" if omitempty is not set
	HostExpirySettings *HostExpirySettings     `json:"host_expiry_settings,omitempty"`
	Secrets            *[]EnrollSecret         `json:"secrets,omitempty"`
	Features           *json.RawMessage        `json:"features"`
	MDM                TeamSpecMDM             `json:"mdm"`
	Scripts            optjson.Slice[string]   `json:"scripts"`
	WebhookSettings    TeamSpecWebhookSettings `json:"webhook_settings"`
	Integrations       TeamSpecIntegrations    `json:"integrations"`
	Software           *SoftwareSpec           `json:"software,omitempty"`
}

type TeamSpecWebhookSettings struct {
	HostStatusWebhook      *HostStatusWebhookSettings      `json:"host_status_webhook"`
	FailingPoliciesWebhook *FailingPoliciesWebhookSettings `json:"failing_policies_webhook"`
}

// TeamSpecIntegrations contains the configuration for external services'
// integrations for a specific team.
type TeamSpecIntegrations struct {
	// If value is nil, we don't want to change the existing value.
	GoogleCalendar *TeamGoogleCalendarIntegration `json:"google_calendar"`
}

// TeamSpecsDryRunAssumptions holds the assumptions that are made when applying team specs in dry-run mode.
type TeamSpecsDryRunAssumptions struct {
	WindowsEnabledAndConfigured optjson.Bool `json:"windows_enabled_and_configured,omitempty"`
}

// TeamSpecFromTeam returns a TeamSpec constructed from the given Team.
func TeamSpecFromTeam(t *Team) (*TeamSpec, error) {
	features, err := json.Marshal(t.Config.Features)
	if err != nil {
		return nil, err
	}
	featuresJSON := json.RawMessage(features)
	var secrets []EnrollSecret
	if len(t.Secrets) > 0 {
		secrets = make([]EnrollSecret, 0, len(t.Secrets))
		for _, secret := range t.Secrets {
			secrets = append(secrets, *secret)
		}
	}
	var agentOptions json.RawMessage
	if t.Config.AgentOptions != nil {
		agentOptions = *t.Config.AgentOptions
	}

	var mdmSpec TeamSpecMDM
	mdmSpec.MacOSUpdates = t.Config.MDM.MacOSUpdates
	mdmSpec.WindowsUpdates = t.Config.MDM.WindowsUpdates
	mdmSpec.MacOSSettings = t.Config.MDM.MacOSSettings.ToMap()
	delete(mdmSpec.MacOSSettings, "enable_disk_encryption")
	mdmSpec.MacOSSetup = t.Config.MDM.MacOSSetup
	mdmSpec.EnableDiskEncryption = optjson.SetBool(t.Config.MDM.EnableDiskEncryption)
	mdmSpec.WindowsSettings = t.Config.MDM.WindowsSettings

	var webhookSettings TeamSpecWebhookSettings
	if t.Config.WebhookSettings.HostStatusWebhook != nil {
		webhookSettings.HostStatusWebhook = t.Config.WebhookSettings.HostStatusWebhook
	}

	var integrations TeamSpecIntegrations
	if t.Config.Integrations.GoogleCalendar != nil {
		integrations.GoogleCalendar = t.Config.Integrations.GoogleCalendar
	}

	return &TeamSpec{
		Name:               t.Name,
		AgentOptions:       agentOptions,
		Features:           &featuresJSON,
		Secrets:            &secrets,
		MDM:                mdmSpec,
		HostExpirySettings: &t.Config.HostExpirySettings,
		WebhookSettings:    webhookSettings,
		Integrations:       integrations,
		Scripts:            t.Config.Scripts,
		Software:           t.Config.Software,
	}, nil
}
