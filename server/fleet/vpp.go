package fleet

import (
	"encoding/json"
	"fmt"
	"slices"
	"time"

	"github.com/fleetdm/fleet/v4/server/variables"
	"howett.net/plist"
)

type VPPAppID struct {
	// AdamID is a unique identifier assigned to each app in
	// the App Store, this value is managed by Apple.
	AdamID   string                    `db:"adam_id" json:"app_store_id"`
	Platform InstallableDevicePlatform `db:"platform" json:"platform"`
}

func (v VPPAppID) String() string {
	return fmt.Sprintf(`%s_%s`, v.AdamID, v.Platform)
}

// VPPAppTeam contains extra metadata injected by fleet
type VPPAppTeam struct {
	VPPAppID

	AppTeamID   uint `db:"id" json:"-"`
	SelfService bool `db:"self_service" json:"self_service"`
	// VersionName is the name of this version of the app on the fleet. Empty means DefaultAppStoreAppVersionName.
	VersionName string `db:"version_name" json:"-"`

	// InstallDuringSetup is either the stored value of that flag for the VPP app
	// or the value to set to that VPP app when batch-setting it. When used to
	// set the value, if nil it will keep the currently saved value (or default
	// to false), while if not nil, it will update the flag's value in the DB.
	InstallDuringSetup *bool `db:"install_during_setup" json:"-"`
	// LabelsIncludeAny are the names of labels associated with this app. If a host has any of
	// these labels, the app is in scope for that host. If this field is set, other label fields
	// cannot be set.
	LabelsIncludeAny []string `json:"labels_include_any"`
	// LabelsExcludeAny are the names of labels associated with this app. If a host has any of
	// these labels, the app is out of scope for that host. If this field is set, other label fields
	// cannot be set.
	LabelsExcludeAny []string `json:"labels_exclude_any"`
	// LabelsIncludeAll are the names of labels associated with this app. If a host has all of
	// these labels, the app is in scope for that host. If this field is set, other label fields
	// cannot be set.
	LabelsIncludeAll []string `json:"labels_include_all"`
	// ValidatedLabels are the labels (either include any/all or exclude any) that have been validated by
	// Fleet as being valid labels. This field is only used internally.
	ValidatedLabels *LabelIdentsWithScope `json:"-"`
	// AddAutoInstallPolicy indicates whether or not we should create an auto-install policy for
	// this VPP app on VPP app add to Fleet.
	AddAutoInstallPolicy bool `json:"-"`
	// AddedAt is when the VPP app was added to the team
	AddedAt     time.Time `db:"added_at" json:"created_at"`
	Categories  []string  `json:"categories"`
	CategoryIDs []uint    `json:"-"`
	// AddedAutomaticInstallPolicy is the auto-install policy that can be
	// automatically created when a VPP app is added to Fleet. This field should be set after VPP
	// app creation if AddAutoInstallPolicy is true.
	AddedAutomaticInstallPolicy *Policy `json:"-"`
	DisplayName                 *string `json:"display_name"`
	// Configuration is the managed app configuration payload.
	// JSON for Android, XML for iOS / iPadOS.
	Configuration       []byte  `json:"configuration,omitempty"`
	AutoUpdateEnabled   *bool   `db:"auto_update_enabled" json:"-"`
	AutoUpdateStartTime *string `db:"auto_update_window_start" json:"-"`
	AutoUpdateEndTime   *string `db:"auto_update_window_end" json:"-"`
}

func (v VPPAppTeam) GetPlatform() string {
	return string(v.Platform)
}

func (v VPPAppTeam) GetAppStoreID() string {
	return v.AdamID
}

// VPPApp represents a VPP (Volume Purchase Program) application,
// this is used by Apple MDM to manage applications via Apple
// Business Manager.
type VPPApp struct {
	VPPAppTeam
	// BundleIdentifier is the unique bundle identifier of the
	// Application.
	BundleIdentifier string `db:"bundle_identifier" json:"bundle_identifier"`
	// IconURL is the URL of this App icon
	IconURL string `db:"icon_url" json:"icon_url"`
	// Name is the user-facing name of this app.
	Name string `db:"name" json:"name"`
	// LatestVersion is the latest version of this app.
	LatestVersion string `db:"latest_version" json:"latest_version"`
	// CountryCode is the App Store storefront country (lowercase ISO 3166-1
	// alpha-2 such as "us", "de") that this app is "anchored" to. It is set
	// on the first add of the (adam_id, platform) and re-used for all future
	// metadata fetches so the displayed name/icon/version stay consistent
	// regardless of which team's token triggers the fetch.
	CountryCode string `db:"country_code" json:"-"`
	// TeamID is used for authorization, it must be json serialized to be available
	// to the rego script. We don't set it outside authorization anyway, so it
	// won't render otherwise.
	TeamID  *uint `db:"-" json:"team_id,omitempty" renameto:"fleet_id"`
	TitleID uint  `db:"title_id" json:"-"`

	CreatedAt time.Time `db:"created_at" json:"-"`
	UpdatedAt time.Time `db:"updated_at" json:"-"`
}

// AuthzType implements authz.AuthzTyper.
func (v *VPPApp) AuthzType() string {
	return "installable_entity"
}

// VPPAppStoreApp contains the field required by the get software title
// endpoint to represent an App Store app (VPP app).
type VPPAppStoreApp struct {
	VPPAppID
	Name          string               `db:"name" json:"name"`
	LatestVersion string               `db:"latest_version" json:"latest_version"`
	IconURL       *string              `db:"icon_url" json:"-"`
	Status        *VPPAppStatusSummary `db:"-" json:"status"`
	SelfService   bool                 `db:"self_service" json:"self_service"`
	// only filled by GetVPPAppMetadataByTeamAndTitleID
	VPPAppsTeamsID uint `db:"vpp_apps_teams_id" json:"-"`

	VersionName         string  `db:"version_name" json:"-"`
	AutoUpdateEnabled   *bool   `db:"auto_update_enabled" json:"-"`
	AutoUpdateStartTime *string `db:"auto_update_window_start" json:"-"`
	AutoUpdateEndTime   *string `db:"auto_update_window_end" json:"-"`
	// AutomaticInstallPolicies is the list of policies that trigger automatic
	// installation of this software.
	AutomaticInstallPolicies []AutomaticInstallPolicy `json:"automatic_install_policies" db:"-"`
	// LabelsIncludeAny is the list of "include any" labels for this app store app (if not nil).
	LabelsIncludeAny []SoftwareScopeLabel `json:"labels_include_any" db:"labels_include_any"`
	// LabelsExcludeAny is the list of "exclude any" labels for this app store app (if not nil).
	LabelsExcludeAny []SoftwareScopeLabel `json:"labels_exclude_any" db:"labels_exclude_any"`
	// LabelsIncludeAll is the list of "include all" labels for this app store app (if not nil).
	LabelsIncludeAll []SoftwareScopeLabel `json:"labels_include_all" db:"labels_include_all"`
	// BundleIdentifier is the bundle identifier for this app.
	BundleIdentifier string `json:"-" db:"bundle_identifier"`
	// AddedAt is when the VPP app was added to the team
	AddedAt time.Time `db:"added_at" json:"created_at"`
	// Categories is the list of categories to which this software belongs: e.g. "Productivity",
	// "Browsers", etc.
	Categories  []string `json:"categories"`
	DisplayName string   `json:"display_name"`
	// Configuration is the managed app configuration payload.
	// JSON for Android, XML for iOS / iPadOS.
	Configuration json.RawMessage `json:"configuration,omitempty"`
}

const (
	// DefaultAppStoreAppVersionName is the name given to an App Store app version added without a name.
	DefaultAppStoreAppVersionName = "Default version"
	// MaxAppStoreAppVersions is the most versions an App Store app can have on one fleet.
	MaxAppStoreAppVersions = 10
	// MaxAppStoreAppVersionNameLength is the most characters a version name can have, the size of its column.
	MaxAppStoreAppVersionNameLength = 255
)

// AppStoreAppVersion is one version of an App Store app on a fleet, as returned in the app_store_apps list of a software title.
type AppStoreAppVersion struct {
	ID                       uint                      `json:"id"`
	Name                     string                    `json:"name"`
	AppStoreID               string                    `json:"app_store_id"`
	Platform                 InstallableDevicePlatform `json:"platform"`
	Version                  string                    `json:"version"`
	Status                   *VPPAppStatusSummary      `json:"status"`
	SelfService              bool                      `json:"self_service"`
	AutomaticInstallPolicies []AutomaticInstallPolicy  `json:"automatic_install_policies"`
	LabelsIncludeAny         []SoftwareScopeLabel      `json:"labels_include_any"`
	LabelsExcludeAny         []SoftwareScopeLabel      `json:"labels_exclude_any"`
	LabelsIncludeAll         []SoftwareScopeLabel      `json:"labels_include_all"`
	AutoUpdateEnabled        *bool                     `json:"auto_update_enabled"`
	AutoUpdateStartTime      *string                   `json:"auto_update_window_start"`
	AutoUpdateEndTime        *string                   `json:"auto_update_window_end"`
	AddedAt                  time.Time                 `json:"created_at"`
	Categories               []string                  `json:"categories"`
	DisplayName              string                    `json:"display_name"`
	Configuration            json.RawMessage           `json:"configuration,omitempty"`
}

func (v *VPPAppStoreApp) AppStoreAppVersion() AppStoreAppVersion {
	return AppStoreAppVersion{
		ID:                       v.VPPAppsTeamsID,
		Name:                     v.VersionName,
		AppStoreID:               v.AdamID,
		Platform:                 v.Platform,
		Version:                  v.LatestVersion,
		Status:                   v.Status,
		SelfService:              v.SelfService,
		AutomaticInstallPolicies: v.AutomaticInstallPolicies,
		LabelsIncludeAny:         v.LabelsIncludeAny,
		LabelsExcludeAny:         v.LabelsExcludeAny,
		LabelsIncludeAll:         v.LabelsIncludeAll,
		AutoUpdateEnabled:        v.AutoUpdateEnabled,
		AutoUpdateStartTime:      v.AutoUpdateStartTime,
		AutoUpdateEndTime:        v.AutoUpdateEndTime,
		AddedAt:                  v.AddedAt,
		Categories:               v.Categories,
		DisplayName:              v.DisplayName,
		Configuration:            v.Configuration,
	}
}

// VPPAppStatusSummary represents aggregated status metrics for a VPP app.
type VPPAppStatusSummary struct {
	// Installed is the number of hosts that have the VPP app installed.
	Installed uint `json:"installed" db:"installed"`
	// Pending is the number of hosts that have the VPP app pending installation.
	Pending uint `json:"pending" db:"pending"`
	// Failed is the number of hosts that have the VPP app installation failed.
	Failed uint `json:"failed" db:"failed"`
}

type ErrVPPTokenTeamConstraint struct {
	Name string
	ID   *uint
}

func (e ErrVPPTokenTeamConstraint) Error() string {
	return fmt.Sprintf("Error: %q team already has a VPP token. Each team can only have one VPP token.", e.Name)
}

// HostVPPSoftwareInstall represents a VPP software install attempt on a host.
type HostVPPSoftwareInstall struct {
	InstallCommandUUID   string     `db:"command_uuid"`
	InstallCommandAckAt  *time.Time `db:"ack_at"`
	HostID               uint       `db:"host_id"`
	InstallCommandStatus string     `db:"install_command_status"`
	BundleIdentifier     string     `db:"bundle_identifier"`
	RetryCount           int        `db:"retry_count"`
	ExpectedVersion      string     `db:"expected_version"`
}

type HostVPPSoftwareInstallLite struct {
	InstallCommandUUID string `db:"command_uuid"`
	HostID             uint   `db:"host_id"`
	RetryCount         int    `db:"retry_count"`
}

// HostAndroidVPPSoftwareInstall represents the payload needed to
// insert a VPP software install record for an Android host.
//
// NOTE: Currently only supported for setup experience, to revisit when
// we support Android app installs at-large (as it will then go through
// the upcoming queue). For this reason, user ID and (Fleet-) policy id
// are always null and self-service is always false, while platform is
// always android.
type HostAndroidVPPSoftwareInstall struct {
	HostID            uint   `db:"host_id"`
	AdamID            string `db:"adam_id"`             // for Android, this is the e.g. com.chrome application ID
	CommandUUID       string `db:"command_uuid"`        // uuid of the corresponding android_policy_request row
	AssociatedEventID string `db:"associated_event_id"` // for Android (for the current setup-experience-only approach), we overload this field to store the Android policy version ID
	VPPAppTeamID      uint   `db:"vpp_app_team_id"`
}

const (
	DefaultVPPInstallVerifyTimeout = 10 * time.Minute
	DefaultVPPVerifyRequestDelay   = 5 * time.Second
)

type AppStoreAppUpdatePayload struct {
	// VersionID is the vpp_apps_teams id of the version to update, required when the title has more than one version on the fleet.
	VersionID        *uint
	VersionName      *string
	SelfService      *bool
	LabelsIncludeAny []string
	LabelsExcludeAny []string
	LabelsIncludeAll []string
	Categories       []string
	DisplayName      *string
	Configuration    []byte
	SoftwareAutoUpdateConfig
}

// VPPClientUserStatus is the lifecycle state of a row in vpp_client_users.
type VPPClientUserStatus string

const (
	// VPPClientUserStatusPending means Fleet generated a client_user_id but
	// Apple has not yet acknowledged the registration (or returned an error).
	VPPClientUserStatusPending VPPClientUserStatus = "pending"
	// VPPClientUserStatusRegistered means Apple has returned a userId for the
	// client_user_id, and the row is ready for use in Associate Assets.
	VPPClientUserStatusRegistered VPPClientUserStatus = "registered"
	// VPPClientUserStatusRetired is reserved for future use when a host is
	// unenrolled or its Managed Apple ID changes.
	VPPClientUserStatusRetired VPPClientUserStatus = "retired"
)

// VPPClientUser represents a row in `vpp_client_users` — the mapping between a
// VPP token (location), a Managed Apple ID, and the Fleet-generated
// `clientUserId` we send to Apple's user-scoped VPP endpoints.
type VPPClientUser struct {
	ID             uint                `db:"id" json:"-"`
	VPPTokenID     uint                `db:"vpp_token_id" json:"vpp_token_id"`
	ManagedAppleID string              `db:"managed_apple_id" json:"managed_apple_id"`
	ClientUserID   string              `db:"client_user_id" json:"client_user_id"`
	AppleUserID    *string             `db:"apple_user_id" json:"apple_user_id,omitempty"`
	Status         VPPClientUserStatus `db:"status" json:"status"`
	CreatedAt      time.Time           `db:"created_at" json:"created_at"`
	UpdatedAt      time.Time           `db:"updated_at" json:"updated_at"`
}

// FleetVarsSupportedInAppleAppConfig is the allow-list of Fleet variables that
// can appear in an iOS / iPadOS managed app configuration plist. Subset of the
// variables supported in Apple configuration profiles — credential variables
// (NDES, SCEP, DigiCert) don't fit the InstallApplication command shape.
var FleetVarsSupportedInAppleAppConfig = []FleetVarName{
	FleetVarHostUUID,
	FleetVarHostHardwareSerial,
	FleetVarHostPlatform,
	FleetVarHostEndUserEmailIDP,
	FleetVarHostEndUserIDPUsername,
	FleetVarHostEndUserIDPUsernameLocalPart,
	FleetVarHostEndUserIDPGroups,
	FleetVarHostEndUserIDPDepartment,
	FleetVarHostEndUserIDPFullname,
}

// ValidateAppleAppConfiguration validates a managed app configuration payload
// for an iOS or iPadOS InstallApplication command. The payload must be an XML
// plist whose root element is a <dict>, and any Fleet variable tokens used in
// string values or keys must be drawn from FleetVarsSupportedInAppleAppConfig.
// Empty input is allowed — callers decide whether to store or clear.
func ValidateAppleAppConfiguration(config []byte) error {
	if len(config) == 0 {
		return nil
	}

	var root map[string]any
	format, err := plist.Unmarshal(config, &root)
	if err != nil {
		return NewInvalidArgumentError("configuration", fmt.Sprintf("invalid plist: %s", err))
	}
	// Apple's MDM InstallApplication only accepts XML plist for the
	// Configuration dict; reject binary, OpenStep and GNUStep formats up front
	// so admins don't get surprised when devices reject the install.
	if format != plist.XMLFormat {
		return NewInvalidArgumentError("configuration", "configuration must be an XML plist")
	}

	// Raw-bytes scan catches structural bypasses (duplicate keys, trailing
	// siblings) invisible to the decoded tree. The decoded-tree walk below
	// catches XML-entity-encoded tokens the regex won't match.
	for _, name := range variables.Find(string(config)) {
		if !slices.Contains(FleetVarsSupportedInAppleAppConfig, FleetVarName(name)) {
			return NewInvalidArgumentError("configuration", fmt.Sprintf("unsupported variable $FLEET_VAR_%s", name))
		}
	}
	if name, ok := findUnsupportedFleetVar(root); ok {
		return NewInvalidArgumentError("configuration", fmt.Sprintf("unsupported variable $FLEET_VAR_%s", name))
	}
	return nil
}

// findUnsupportedFleetVar walks v's keys and string values and returns the
// first $FLEET_VAR_* token that is not in FleetVarsSupportedInAppleAppConfig.
// The second return is false when every referenced variable is allowed.
func findUnsupportedFleetVar(v any) (string, bool) {
	switch t := v.(type) {
	case string:
		for _, name := range variables.Find(t) {
			if !slices.Contains(FleetVarsSupportedInAppleAppConfig, FleetVarName(name)) {
				return name, true
			}
		}
	case map[string]any:
		for k, val := range t {
			if name, ok := findUnsupportedFleetVar(k); ok {
				return name, ok
			}
			if name, ok := findUnsupportedFleetVar(val); ok {
				return name, ok
			}
		}
	case []any:
		for _, val := range t {
			if name, ok := findUnsupportedFleetVar(val); ok {
				return name, ok
			}
		}
	}
	return "", false
}

// VPPInstallReleaseInfo carries the per-install data the cancel path needs to
// decide whether (and how) to release a previously-reserved VPP license seat
// for a canceled install. AssociatedEventID is set only when this install was
// the one that called AssociateAssets; HasOtherActiveInstall is true when
// another non-canceled, non-failed install for the same (host, adam_id) is
// still in flight and would still need the seat.
type VPPInstallReleaseInfo struct {
	AdamID                string
	AssociatedEventID     string
	HasOtherActiveInstall bool
}

type DuplicateStringGroup struct {
	// Indices in the provided input slice
	Indices []int
}

// HostAppStoreAppVersion is the App Store app version a host gets for a software title: the first-added version
// the host is in label scope for, or the first-added version with InScope false when the host is in scope for none.
type HostAppStoreAppVersion struct {
	VPPAppTeamID uint   `db:"id"`
	AdamID       string `db:"adam_id"`
	TitleID      uint   `db:"title_id"`
	Name         string `db:"name"`
	InScope      bool   `db:"in_scope"`
}
