package fleet

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"golang.org/x/text/unicode/norm"
)

const (
	RoleAdmin        = "admin"
	RoleMaintainer   = "maintainer"
	RoleObserver     = "observer"
	RoleObserverPlus = "observer_plus"
	RoleGitOps       = "gitops"
	RoleTechnician   = "technician"
	TeamNameNoTeam   = "No team"
	TeamNameAllTeams = "All teams"
)

const (
	ReservedNameAllTeams = "All teams"
	ReservedNameNoTeam   = "No team"
)

// Display names used in user-facing error messages.
const (
	DisplayNameNoTeam   = "Unassigned"
	DisplayNameAllTeams = "All fleets"
)

// MaxTeamNameLength matches the varchar(255) size of teams.name in MySQL.
// Enforce this before insert/update so callers get an InvalidArgumentError
// instead of a raw "Data too long" MySQL error.
const MaxTeamNameLength = 255

// IsReservedTeamName checks if the name provided is a reserved fleet name (case-insensitive).
// Both old names ("No team", "All teams") and new display names ("Unassigned", "All fleets")
// are reserved to prevent creating teams with any of these names.
func IsReservedTeamName(name string) bool {
	normalizedName := strings.ToLower(norm.NFC.String(name))
	return normalizedName == "no team" || normalizedName == "all teams" ||
		normalizedName == "unassigned" || normalizedName == "all fleets"
}

// IsUnassignedFleetName checks if the name provided is the display name of the "Unassigned" pseudo-fleet, i.e. hosts with no fleet (case-insensitive).
func IsUnassignedFleetName(name string) bool {
	return strings.ToLower(name) == "unassigned"
}

type TeamPayload struct {
	Name               *string              `json:"name"`
	Description        *string              `json:"description"`
	Secrets            []*EnrollSecret      `json:"secrets"`
	WebhookSettings    *TeamWebhookSettings `json:"webhook_settings"`
	Integrations       *TeamIntegrations    `json:"integrations"`
	MDM                *TeamPayloadMDM      `json:"mdm"`
	HostExpirySettings *HostExpirySettings  `json:"host_expiry_settings"`
	Features           *TeamPayloadFeatures `json:"features"`
	// Note AgentOptions must be set by a separate endpoint.
}

// TeamPayloadFeatures is a payload-only subset of Features writable via
// `PATCH /api/v1/fleet/fleets/{id}`. It mirrors the global config's
// `features` shape so admins can use the same JSON path on both endpoints.
//
// Only the sub-fields defined here take effect; the broader Features
// fields (enable_host_users, additional_queries, detail_query_overrides)
// remain settable per-fleet only via the `/spec/fleets` GitOps path.
type TeamPayloadFeatures struct {
	// EnableSoftwareInventory uses optjson.Bool so a key omitted from a
	// PATCH body retains its current stored value (PATCH-merge semantics).
	EnableSoftwareInventory optjson.Bool           `json:"enable_software_inventory"`
	HistoricalData          *HistoricalDataPayload `json:"historical_data"`
}

// HistoricalDataPayload is the per-sub-key partial-PATCH form of
// HistoricalDataSettings. Each field uses optjson.Bool so a sub-key omitted
// from a PATCH body retains its current stored value, while a sub-key
// explicitly set to `false` flips it. Mirrors how MDM.EnableDiskEncryption
// behaves on this endpoint.
type HistoricalDataPayload struct {
	Uptime          optjson.Bool `json:"uptime"`
	Vulnerabilities optjson.Bool `json:"vulnerabilities"`
}

// TeamPayloadMDM is a distinct struct than TeamMDM because in ModifyTeam we
// need to be able which part of the MDM config was provided in the request,
// so the fields are pointers to structs.
type TeamPayloadMDM struct {
	EnableDiskEncryption       optjson.Bool `json:"enable_disk_encryption"`
	EnableRecoveryLockPassword optjson.Bool `json:"enable_recovery_lock_password"`
	// RequireBitLockerPIN indicates whether BitLocker PIN is required for Windows devices
	// in order for Fleet to consider them compliant.
	RequireBitLockerPIN optjson.Bool `json:"windows_require_bitlocker_pin"`

	// MacOSUpdates defines the OS update settings for macOS devices.
	MacOSUpdates *AppleOSUpdateSettings `json:"macos_updates"`
	// IOSUpdates defines the OS update settings for iOS devices.
	IOSUpdates *AppleOSUpdateSettings `json:"ios_updates"`
	// IPadOSUpdates defines the OS update settings for iPadOS devices.
	IPadOSUpdates *AppleOSUpdateSettings `json:"ipados_updates"`
	// WindowsUpdates defines the OS update settings for Windows devices.
	WindowsUpdates *WindowsUpdates `json:"windows_updates"`

	MacOSSetup       *MacOSSetup    `json:"macos_setup"`
	HostNameTemplate optjson.String `json:"name_template"`

	// MacOSSettings exposes only the disk encryption surface on the team PATCH endpoint;
	// configuration profiles are managed through their own endpoints.
	MacOSSettings *TeamPayloadMacOSSettings `json:"macos_settings" renameto:"apple_settings"`
	// WindowsSettings exposes only the managed local account and disk encryption surfaces
	// on the team PATCH endpoint; configuration profiles are managed through their own endpoints.
	WindowsSettings *TeamPayloadWindowsSettings `json:"windows_settings"`
	LinuxSettings   *TeamPayloadLinuxSettings   `json:"linux_settings"`
}

// TeamPayloadMacOSSettings is the subset of macos_settings (apple_settings) fields
// settable via the team PATCH endpoint.
type TeamPayloadMacOSSettings struct {
	EnableDiskEncryption          optjson.Bool `json:"enable_disk_encryption"`
	EnableEscrowDiskEncryptionKey optjson.Bool `json:"enable_escrow_disk_encryption_key"`
}

// TeamPayloadWindowsSettings is the subset of windows_settings fields settable via the team PATCH endpoint.
type TeamPayloadWindowsSettings struct {
	EnableManagedLocalAccount optjson.Bool `json:"enable_managed_local_account"`
	// RequireBitLockerPIN is the canonical home of the deprecated top-level
	// windows_require_bitlocker_pin key.
	RequireBitLockerPIN  optjson.Bool `json:"require_bitlocker_pin"`
	EnableDiskEncryption optjson.Bool `json:"enable_disk_encryption"`
}

// TeamPayloadLinuxSettings is the subset of linux_settings fields settable via the team PATCH endpoint.
type TeamPayloadLinuxSettings struct {
	EnableEscrowDiskEncryptionKey optjson.Bool `json:"enable_escrow_disk_encryption_key"`
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

// TeamLite is a subset of Team that only includes columns in the Team table
type TeamLite struct {
	// ID is the database ID.
	ID       uint    `json:"id" db:"id"`
	Filename *string `json:"gitops_filename,omitempty" db:"filename"`
	// CreatedAt is the timestamp of the label creation.
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	// Name is the human friendly name of the team.
	Name string `json:"name" db:"name"`
	// Description is an optional description for the team.
	Description string         `json:"description" db:"description"`
	Config      TeamConfigLite `json:"-" db:"config"`
}

func (t *Team) ToTeamLite() *TeamLite {
	return &TeamLite{
		ID:          t.ID,
		Filename:    t.Filename,
		CreatedAt:   t.CreatedAt,
		Name:        t.Name,
		Description: t.Description,
		Config:      t.Config.ToLite(),
	}
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

	// Fall back to defaults when these keys are missing from the stored config
	// (e.g. a team created before they existed), so the serialized team matches
	// what AppConfig.MarshalJSON serves for the global config.
	if !x.MDM.MacOSSetup.EnableManagedLocalAccount.Valid {
		x.MDM.MacOSSetup.EnableManagedLocalAccount = optjson.SetBool(false)
	}
	if !x.MDM.MacOSSetup.EndUserLocalAccountType.Valid {
		x.MDM.MacOSSetup.EndUserLocalAccountType = optjson.SetString("admin")
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
	if !x.MDM.MacOSSetup.LockEndUserInfo.Valid {
		x.MDM.MacOSSetup.LockEndUserInfo = optjson.SetBool(x.MDM.MacOSSetup.EnableEndUserAuthentication)
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
	AgentOptions       *json.RawMessage    `json:"agent_options,omitempty"`
	HostExpirySettings HostExpirySettings  `json:"host_expiry_settings"`
	WebhookSettings    TeamWebhookSettings `json:"webhook_settings"`
	Integrations       TeamIntegrations    `json:"integrations"`
	MDM                TeamMDM             `json:"mdm"`
	// the below aren't serialized as-is into config JSON column in the teams table
	Features Features              `json:"features"`
	Scripts  optjson.Slice[string] `json:"scripts,omitempty"`
	Software *SoftwareSpec         `json:"software,omitempty"`
}

func (t TeamConfig) ToLite() TeamConfigLite {
	return TeamConfigLite{
		AgentOptions:       t.AgentOptions,
		HostExpirySettings: t.HostExpirySettings,
		WebhookSettings:    t.WebhookSettings,
		Integrations:       t.Integrations,
		MDM:                t.MDM,
	}
}

// TeamConfigLite contains only TeamConfig fields that are available as-is from teams.config JSON
type TeamConfigLite struct {
	// AgentOptions is the options for osquery and Orbit.
	AgentOptions       *json.RawMessage    `json:"agent_options,omitempty"`
	HostExpirySettings HostExpirySettings  `json:"host_expiry_settings"`
	WebhookSettings    TeamWebhookSettings `json:"webhook_settings"`
	Integrations       TeamIntegrations    `json:"integrations"`
	MDM                TeamMDM             `json:"mdm"`
}

type TeamWebhookSettings struct {
	// HostStatusWebhook can be nil to match the TeamSpec webhook settings
	HostStatusWebhook      *HostStatusWebhookSettings     `json:"host_status_webhook"`
	FailingPoliciesWebhook FailingPoliciesWebhookSettings `json:"failing_policies_webhook"`
	// HostActivitiesWebhook is nil when not provided so partial updates and
	// team specs can leave the stored value untouched.
	HostActivitiesWebhook *HostActivitiesWebhookSettings `json:"host_activities_webhook"`
}

// HostActivitiesWebhookSettings is the per-fleet webhook fired when an
// activity linked to one of the fleet's hosts is created. The payload has the
// same format as the global activities webhook (ActivitiesWebhookSettings).
type HostActivitiesWebhookSettings struct {
	Enable         bool   `json:"enable_host_activities_webhook"`
	DestinationURL string `json:"destination_url"`
}

// HostActivitiesWebhookLookup is the subset of Datastore reads needed to
// resolve the host-activities webhooks of the fleets a set of hosts belong to.
type HostActivitiesWebhookLookup interface {
	ListHostsLiteByIDs(ctx context.Context, ids []uint) ([]*Host, error)
	TeamLitesByIDs(ctx context.Context, ids []uint) ([]*TeamLite, error)
}

// HostActivitiesWebhookDelivery is one fleet's resolved host-activities
// webhook destination together with the subset of the activity's hosts that
// belong to that fleet. Payloads are scoped this way so a delivery is always
// exactly one fleet's subscription — it never mixes fleets' host IDs.
type HostActivitiesWebhookDelivery struct {
	DestinationURL string
	HostIDs        []uint
}

// ResolveHostActivitiesWebhooks returns one enabled host-activities webhook delivery per fleet the given hosts belong to.
func ResolveHostActivitiesWebhooks(ctx context.Context, ds HostActivitiesWebhookLookup, hostIDs []uint) ([]HostActivitiesWebhookDelivery, error) {
	if len(hostIDs) == 0 {
		return nil, nil
	}

	hosts, err := ds.ListHostsLiteByIDs(ctx, hostIDs)
	if err != nil {
		return nil, err
	}

	// fleetKeys keeps delivery order deterministic (map iteration is
	// randomized). Key 0 is reserved for "Unassigned" hosts (nil TeamID), so
	// it can't collide with a real fleet.
	fleetKeys := make([]uint, 0, len(hosts))
	hostsByFleet := make(map[uint][]uint)
	for _, host := range hosts {
		var fleetKey uint
		if host.TeamID != nil {
			fleetKey = *host.TeamID
		}
		if _, ok := hostsByFleet[fleetKey]; !ok {
			fleetKeys = append(fleetKeys, fleetKey)
		}
		hostsByFleet[fleetKey] = append(hostsByFleet[fleetKey], host.ID)
	}

	fleets, err := ds.TeamLitesByIDs(ctx, fleetKeys)
	if err != nil {
		return nil, err
	}
	fleetsByID := make(map[uint]*TeamLite, len(fleets))
	for _, f := range fleets {
		fleetsByID[f.ID] = f
	}

	// Resolve each fleet's webhook into its own delivery.
	var deliveries []HostActivitiesWebhookDelivery
	for _, fleetKey := range fleetKeys {
		team, ok := fleetsByID[fleetKey]
		if !ok { // deleted fleet
			continue
		}
		webhook := team.Config.WebhookSettings.HostActivitiesWebhook
		if webhook == nil || !webhook.Enable || webhook.DestinationURL == "" {
			continue
		}
		deliveries = append(deliveries, HostActivitiesWebhookDelivery{
			DestinationURL: webhook.DestinationURL,
			HostIDs:        hostsByFleet[fleetKey],
		})
	}

	return deliveries, nil
}

// DefaultTeam represents the limited team information returned for team ID 0
type DefaultTeam struct {
	ID                uint   `json:"id"`
	Name              string `json:"name"`
	DefaultTeamConfig        // Embedded struct - fields appear at top level in JSON
}

type DefaultTeamConfig struct {
	WebhookSettings DefaultTeamWebhookSettings `json:"webhook_settings"`
	Integrations    DefaultTeamIntegrations    `json:"integrations"`
}

// DefaultTeamWebhookSettings contains webhook settings for team ID 0
type DefaultTeamWebhookSettings struct {
	FailingPoliciesWebhook FailingPoliciesWebhookSettings `json:"failing_policies_webhook"`
	HostActivitiesWebhook  *HostActivitiesWebhookSettings `json:"host_activities_webhook"`
}

// DefaultTeamIntegrations contains only the integrations supported for team ID 0
type DefaultTeamIntegrations struct {
	Jira    []*TeamJiraIntegration    `json:"jira"`
	Zendesk []*TeamZendeskIntegration `json:"zendesk"`
}

type TeamSpecSoftwareAsset struct {
	Path string `json:"path"`
}

type TeamSpecAppStoreApp struct {
	AppStoreID       string   `json:"app_store_id"`
	SelfService      bool     `json:"self_service"`
	LabelsIncludeAny []string `json:"labels_include_any"`
	LabelsExcludeAny []string `json:"labels_exclude_any"`
	LabelsIncludeAll []string `json:"labels_include_all"`
	// Categories is the list of names of software categories associated with this VPP app.
	Categories []string `json:"categories"`
	// InstallDuringSetup indicates whether a package should be incorporated into setup experience;
	// if not supplied (Valid field is false) then the server-side value for setup experience membership
	// is not changed, for compatibility with the old fleetctl apply format
	InstallDuringSetup optjson.Bool          `json:"setup_experience"`
	Icon               TeamSpecSoftwareAsset `json:"icon"`
	Platform           string                `json:"platform"`
	DisplayName        string                `json:"display_name,omitempty"`
	Configuration      TeamSpecSoftwareAsset `json:"configuration"`
	// Auto-update fields for VPP apps
	AutoUpdateEnabled   *bool   `json:"auto_update_enabled,omitempty"`
	AutoUpdateStartTime *string `json:"auto_update_window_start,omitempty"`
	AutoUpdateEndTime   *string `json:"auto_update_window_end,omitempty"`
}

func (spec TeamSpecAppStoreApp) ResolvePaths(baseDir string) TeamSpecAppStoreApp {
	spec.Icon.Path = resolveApplyRelativePath(baseDir, spec.Icon.Path)
	spec.Configuration.Path = resolveApplyRelativePath(baseDir, spec.Configuration.Path)

	return spec
}

type TeamMDM struct {
	EnableDiskEncryption       bool                  `json:"enable_disk_encryption"`
	EnableRecoveryLockPassword bool                  `json:"enable_recovery_lock_password"`
	RequireBitLockerPIN        bool                  `json:"windows_require_bitlocker_pin"`
	MacOSUpdates               AppleOSUpdateSettings `json:"macos_updates"`
	IOSUpdates                 AppleOSUpdateSettings `json:"ios_updates"`
	IPadOSUpdates              AppleOSUpdateSettings `json:"ipados_updates"`
	WindowsUpdates             WindowsUpdates        `json:"windows_updates"`
	MacOSSettings              MacOSSettings         `json:"macos_settings" renameto:"apple_settings"`
	MacOSSetup                 MacOSSetup            `json:"macos_setup" renameto:"setup_experience"`

	WindowsSettings WindowsSettings `json:"windows_settings"`

	AndroidSettings AndroidSettings `json:"android_settings"`

	LinuxSettings LinuxSettings `json:"linux_settings"`

	// HostNameTemplate is the template used to compute a host's display name from
	// host-identity Fleet variables (e.g. $FLEET_VAR_HOST_HARDWARE_SERIAL).
	HostNameTemplate string `json:"name_template"`
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

// MarshalJSON keeps the deprecated flat EnableDiskEncryption toggle virtual:
// every serialization (API responses, teams.config storage) recomputes it as
// the AND of the four per-platform disk encryption settings, and unset
// per-platform settings become explicit booleans (fanned out from the flat
// value when none was ever set). The other unset optjson toggles below default
// to false the same way, so no serialization path emits null for them.
func (t TeamMDM) MarshalJSON() ([]byte, error) {
	t.EnableDiskEncryption = normalizeDiskEncryptionFields(
		t.EnableDiskEncryption,
		&t.MacOSSettings.EnableDiskEncryption,
		&t.MacOSSettings.EnableEscrowDiskEncryptionKey,
		&t.WindowsSettings.EnableDiskEncryption,
		&t.LinuxSettings.EnableEscrowDiskEncryptionKey,
	)
	// keep the deprecated windows_require_bitlocker_pin key in sync with its
	// canonical windows_settings.require_bitlocker_pin home
	if t.WindowsSettings.RequireBitLockerPIN.Valid {
		t.RequireBitLockerPIN = t.WindowsSettings.RequireBitLockerPIN.Value
	} else {
		t.WindowsSettings.RequireBitLockerPIN = optjson.SetBool(t.RequireBitLockerPIN)
	}
	if !t.WindowsSettings.EnableManagedLocalAccount.Valid {
		t.WindowsSettings.EnableManagedLocalAccount = optjson.SetBool(false)
	}
	// the alias type has no methods, so marshaling it avoids infinite recursion
	type alias TeamMDM
	return json.Marshal(alias(t))
}

// UnmarshalJSON fills per-platform disk encryption settings ABSENT from the
// stored document from the flat toggle. This keeps team configs correct even
// if the per-platform keys were dropped from the stored JSON (e.g. re-saved by
// a pre-split server after the fan-out migration ran). A key explicitly
// present (including explicit null) is never overridden here.
func (t *TeamMDM) UnmarshalJSON(b []byte) error {
	// the alias type has no methods, so unmarshaling it avoids infinite recursion
	type alias TeamMDM
	if err := json.Unmarshal(b, (*alias)(t)); err != nil {
		return err
	}
	for _, f := range []*optjson.Bool{
		&t.MacOSSettings.EnableDiskEncryption,
		&t.MacOSSettings.EnableEscrowDiskEncryptionKey,
		&t.WindowsSettings.EnableDiskEncryption,
		&t.LinuxSettings.EnableEscrowDiskEncryptionKey,
	} {
		if !f.Set {
			*f = optjson.SetBool(t.EnableDiskEncryption)
		}
	}
	// the BitLocker PIN's canonical home inherits the deprecated top-level key
	// the same way when absent from the document
	if !t.WindowsSettings.RequireBitLockerPIN.Set {
		t.WindowsSettings.RequireBitLockerPIN = optjson.SetBool(t.RequireBitLockerPIN)
	}
	return nil
}

// DiskEncryptionConfig returns the team's effective per-platform disk
// encryption settings.
func (t *TeamMDM) DiskEncryptionConfig() DiskEncryptionConfig {
	if t == nil {
		return DiskEncryptionConfig{}
	}
	pinRequired := t.RequireBitLockerPIN
	if t.WindowsSettings.RequireBitLockerPIN.Valid {
		pinRequired = t.WindowsSettings.RequireBitLockerPIN.Value
	}
	return DiskEncryptionConfig{
		MacOSEnabled:         t.MacOSSettings.EnableDiskEncryption.Value,
		MacOSEscrowEnabled:   t.MacOSSettings.EnableEscrowDiskEncryptionKey.Value,
		WindowsEnabled:       t.WindowsSettings.EnableDiskEncryption.Value,
		BitLockerPINRequired: pinRequired,
		LinuxEscrowEnabled:   t.LinuxSettings.EnableEscrowDiskEncryptionKey.Value,
	}
}

// Copy returns a deep copy of the TeamMDM.
func (t *TeamMDM) Copy() *TeamMDM {
	if t == nil {
		return nil
	}

	clone := *t

	// EnableDiskEncryption, MacOS/IOS/IPadOS/WindowsUpdates don't have fields that
	// require cloning (all fields are basic value types, no
	// pointers/slices/maps).

	if t.MacOSSettings.CustomSettings != nil {
		clone.MacOSSettings.CustomSettings = make([]MDMProfileSpec, len(t.MacOSSettings.CustomSettings))
		for i, mps := range t.MacOSSettings.CustomSettings {
			clone.MacOSSettings.CustomSettings[i] = *mps.Copy()
		}
	}
	if t.WindowsSettings.CustomSettings.Set {
		windowsSettings := make([]MDMProfileSpec, len(t.WindowsSettings.CustomSettings.Value))
		for i, mps := range t.WindowsSettings.CustomSettings.Value {
			windowsSettings[i] = *mps.Copy()
		}
		clone.WindowsSettings.CustomSettings = optjson.SetSlice(windowsSettings)
	}
	if t.AndroidSettings.CustomSettings.Set {
		androidSettings := make([]MDMProfileSpec, len(t.AndroidSettings.CustomSettings.Value))
		for i, mps := range t.AndroidSettings.CustomSettings.Value {
			androidSettings[i] = *mps.Copy()
		}
		clone.AndroidSettings.CustomSettings = optjson.SetSlice(androidSettings)
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
	EnableDiskEncryption       optjson.Bool `json:"enable_disk_encryption"`
	EnableRecoveryLockPassword optjson.Bool `json:"enable_recovery_lock_password"`
	// RequireBitLockerPIN indicates whether BitLocker PIN is required for Windows devices
	// in order for Fleet to consider them compliant.
	RequireBitLockerPIN optjson.Bool `json:"windows_require_bitlocker_pin"`

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
	MacOSSettings map[string]any `json:"macos_settings" renameto:"apple_settings"`
	MacOSSetup    MacOSSetup     `json:"macos_setup" renameto:"setup_experience"`

	WindowsSettings WindowsSettings `json:"windows_settings"`

	AndroidSettings  AndroidSettings `json:"android_settings"`
	LinuxSettings    LinuxSettings   `json:"linux_settings"`
	HostNameTemplate optjson.String  `json:"name_template"`

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
	if !t.MDM.MacOSSetup.LockEndUserInfo.Valid {
		t.MDM.MacOSSetup.LockEndUserInfo = optjson.SetBool(false)
	}
	if !t.MDM.MacOSSetup.EnableManagedLocalAccount.Valid {
		t.MDM.MacOSSetup.EnableManagedLocalAccount = optjson.SetBool(false)
	}
	if !t.MDM.MacOSSetup.EndUserLocalAccountType.Valid {
		t.MDM.MacOSSetup.EndUserLocalAccountType = optjson.SetString("admin")
	}
	return json.Marshal(t)
}

// Copy creates a deep copy of the TeamConfig
func (t *TeamConfig) Copy() *TeamConfig {
	if t == nil {
		return nil
	}

	clone := *t

	// Deep copy AgentOptions if present
	if t.AgentOptions != nil {
		agentOptionsCopy := make(json.RawMessage, len(*t.AgentOptions))
		copy(agentOptionsCopy, *t.AgentOptions)
		clone.AgentOptions = &agentOptionsCopy
	}

	// Deep copy WebhookSettings
	if t.WebhookSettings.HostStatusWebhook != nil {
		hostStatusCopy := *t.WebhookSettings.HostStatusWebhook
		clone.WebhookSettings.HostStatusWebhook = &hostStatusCopy
	}
	if t.WebhookSettings.HostActivitiesWebhook != nil {
		hostActivitiesCopy := *t.WebhookSettings.HostActivitiesWebhook
		clone.WebhookSettings.HostActivitiesWebhook = &hostActivitiesCopy
	}
	if len(t.WebhookSettings.FailingPoliciesWebhook.PolicyIDs) > 0 {
		clone.WebhookSettings.FailingPoliciesWebhook.PolicyIDs = make([]uint, len(t.WebhookSettings.FailingPoliciesWebhook.PolicyIDs))
		copy(clone.WebhookSettings.FailingPoliciesWebhook.PolicyIDs, t.WebhookSettings.FailingPoliciesWebhook.PolicyIDs)
	}

	// Deep copy integrations
	clone.Integrations = t.Integrations.Copy()

	// Deep copy Features
	clone.Features = *t.Features.Copy()

	// Deep copy all MDM fields (includes macOS/windows custom settings and setup software)
	clone.MDM = *t.MDM.Copy()

	// Do not copy script and software since they will not be stored/cached in the database.
	clone.Scripts = optjson.Slice[string]{}
	clone.Software = nil

	return &clone
}

// Clone implements the Cloner interface for cache support
func (t *TeamConfig) Clone() (Cloner, error) {
	return t.Copy(), nil
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
	RoleTechnician:   {},
	RoleObserverPlus: {},
	RoleGitOps:       {},
}

var premiumTeamRoles = map[string]struct{}{
	RoleTechnician:   {},
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
	RoleTechnician:   {},
	RoleObserverPlus: {},
	RoleGitOps:       {},
}

var premiumGlobalRoles = map[string]struct{}{
	RoleTechnician:   {},
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
			return NewError(ErrNoRoleNeeded, "either global role or fleet role needs to be defined")
		}
		for _, t := range teamUsers {
			if !ValidTeamRole(t.Role) {
				return NewErrorf(ErrNoRoleNeeded, "invalid fleet role: %s", t.Role)
			}
		}
		return nil
	}

	if len(teamUsers) > 0 {
		return NewError(ErrNoRoleNeeded, "Cannot specify both global and fleet-scoped roles")
	}

	if !ValidGlobalRole(*globalRole) {
		return NewErrorf(ErrNoRoleNeeded, "invalid global role: %s", *globalRole)
	}

	return nil
}

// PremiumRolesPresent returns true if the provided globalRole or any
// role in teamRoles is a premium role.
func PremiumRolesPresent(globalRole *string, teamRoles []UserTeam) bool {
	if globalRole != nil {
		if _, ok := premiumGlobalRoles[*globalRole]; ok {
			return true
		}
	}
	for _, teamRole := range teamRoles {
		if _, ok := premiumTeamRoles[teamRole.Role]; ok {
			return true
		}
	}
	return false
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
	// ObserverTeamID, when set, restricts observer-role access to only this team.
	// Used for live queries where observer_can_run is scoped to the query's own team,
	// so that a user who is observer on multiple teams only sees hosts from the query's team.
	// Non-observer roles (admin, maintainer, etc.) are not affected.
	ObserverTeamID *uint
}

func (f TeamFilter) UserCanAccessSelectedTeam() bool {
	if f.TeamID == nil { // this method doesn't make sense if there's no team ID specified
		return false
	}

	return f.User.HasAnyGlobalRole() || f.User.HasAnyRoleInTeam(*f.TeamID)
}

const (
	TeamKind  = "team"
	FleetKind = "fleet"
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
	HostActivitiesWebhook  *HostActivitiesWebhookSettings  `json:"host_activities_webhook"`
}

// TeamSpecIntegrations contains the configuration for external services'
// integrations for a specific team.
type TeamSpecIntegrations struct {
	// If value is nil, we don't want to change the existing value.
	GoogleCalendar *TeamGoogleCalendarIntegration `json:"google_calendar"`
	// ConditionalAccessEnabled indicates whether "Conditional access" is enabled/disabled for the team.
	ConditionalAccessEnabled *bool `json:"conditional_access_enabled"`
}

// TeamSpecsDryRunAssumptions holds the assumptions that are made when applying team specs in dry-run mode.
type TeamSpecsDryRunAssumptions struct {
	WindowsEnabledAndConfigured optjson.Bool `json:"windows_enabled_and_configured,omitempty"`
	AndroidEnabledAndConfigured optjson.Bool `json:"android_enabled_and_configured,omitempty"`
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

	// normalize a local copy so the spec always carries explicit per-platform
	// disk encryption booleans, even for configs stored before the
	// per-platform split.
	mdm := t.Config.MDM
	flat := normalizeDiskEncryptionFields(
		mdm.EnableDiskEncryption,
		&mdm.MacOSSettings.EnableDiskEncryption,
		&mdm.MacOSSettings.EnableEscrowDiskEncryptionKey,
		&mdm.WindowsSettings.EnableDiskEncryption,
		&mdm.LinuxSettings.EnableEscrowDiskEncryptionKey,
	)

	var mdmSpec TeamSpecMDM
	mdmSpec.MacOSUpdates = mdm.MacOSUpdates
	mdmSpec.WindowsUpdates = mdm.WindowsUpdates
	mdmSpec.MacOSSettings = mdm.MacOSSettings.ToMap()
	// assets are only present in ToMap for GitOps request validation; they are
	// not stored on the team config, so keep them out of the generated spec.
	delete(mdmSpec.MacOSSettings, "assets")
	mdmSpec.MacOSSetup = mdm.MacOSSetup
	// emit the deprecated flat toggle only when it agrees with every
	// per-platform setting: the flat toggle wins when provided, so emitting it
	// for a mixed state would reset the per-platform values on re-apply.
	uniformDiskEncryption := true
	for _, v := range []bool{
		mdm.MacOSSettings.EnableDiskEncryption.Value,
		mdm.MacOSSettings.EnableEscrowDiskEncryptionKey.Value,
		mdm.WindowsSettings.EnableDiskEncryption.Value,
		mdm.LinuxSettings.EnableEscrowDiskEncryptionKey.Value,
	} {
		if v != flat {
			uniformDiskEncryption = false
			break
		}
	}
	if uniformDiskEncryption {
		mdmSpec.EnableDiskEncryption = optjson.SetBool(flat)
	}
	mdmSpec.EnableRecoveryLockPassword = optjson.SetBool(mdm.EnableRecoveryLockPassword)
	mdmSpec.WindowsSettings = mdm.WindowsSettings
	mdmSpec.AndroidSettings = mdm.AndroidSettings
	mdmSpec.LinuxSettings = mdm.LinuxSettings

	var webhookSettings TeamSpecWebhookSettings
	if t.Config.WebhookSettings.HostStatusWebhook != nil {
		webhookSettings.HostStatusWebhook = t.Config.WebhookSettings.HostStatusWebhook
	}
	if t.Config.WebhookSettings.HostActivitiesWebhook != nil {
		webhookSettings.HostActivitiesWebhook = t.Config.WebhookSettings.HostActivitiesWebhook
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
