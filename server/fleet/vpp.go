package fleet

import "time"

type VPPAppID struct {
	// AdamID is a unique identifier assigned to each app in
	// the App Store, this value is managed by Apple.
	AdamID      string              `db:"adam_id" json:"app_store_id"`
	Platform    AppleDevicePlatform `db:"platform" json:"platform"`
	SelfService bool                `db:"self_service" json:"self_service"`
}

// VPPApp represents a VPP (Volume Purchase Program) application,
// this is used by Apple MDM to manage applications via Apple
// Business Manager.
type VPPApp struct {
	VPPAppID
	// BundleIdentifier is the unique bundle identifier of the
	// Application.
	BundleIdentifier string `db:"bundle_identifier" json:"bundle_identifier"`
	// IconURL is the URL of this App icon
	IconURL string `db:"icon_url" json:"icon_url"`
	// Name is the user-facing name of this app.
	Name string `db:"name" json:"name"`
	// LatestVersion is the latest version of this app.
	LatestVersion string `db:"latest_version" json:"latest_version"`
	TeamID        *uint  `db:"-" json:"-"`
	TitleID       uint   `db:"title_id" json:"-"`

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
