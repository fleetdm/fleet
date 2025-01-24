package fleet

import (
	"fmt"
	"time"
)

type VPPAppID struct {
	// AdamID is a unique identifier assigned to each app in
	// the App Store, this value is managed by Apple.
	AdamID   string              `db:"adam_id" json:"app_store_id"`
	Platform AppleDevicePlatform `db:"platform" json:"platform"`
}

// VPPAppTeam contains extra metadata injected by fleet
type VPPAppTeam struct {
	VPPAppID

	SelfService bool `db:"self_service" json:"self_service"`

	// InstallDuringSetup is either the stored value of that flag for the VPP app
	// or the value to set to that VPP app when batch-setting it. When used to
	// set the value, if nil it will keep the currently saved value (or default
	// to false), while if not nil, it will update the flag's value in the DB.
	InstallDuringSetup *bool    `db:"install_during_setup" json:"-"`
	LabelsIncludeAny   []string `json:"labels_include_any"`
	LabelsExcludeAny   []string `json:"labels_exclude_any"`
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
	// TeamID is used for authorization, it must be json serialized to be available
	// to the rego script. We don't set it outside authorization anyway, so it
	// won't render otherwise.
	TeamID  *uint `db:"-" json:"team_id,omitempty"`
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
	IconURL       *string              `db:"icon_url" json:"icon_url"`
	Status        *VPPAppStatusSummary `db:"-" json:"status"`
	SelfService   bool                 `db:"self_service" json:"self_service"`
	// only filled by GetVPPAppMetadataByTeamAndTitleID
	VPPAppsTeamsID uint `db:"vpp_apps_teams_id" json:"-"`
	// AutomaticInstallPolicies is the list of policies that trigger automatic
	// installation of this software.
	AutomaticInstallPolicies []AutomaticInstallPolicy `json:"automatic_install_policies" db:"-"`
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
