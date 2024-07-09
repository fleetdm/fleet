package fleet

import "time"

// VPPApp represents a VPP (Volume Purchase Program) application,
// this is used by Apple MDM to manage applications via Apple
// Bussines Manager.
type VPPApp struct {
	// AdamID is a unique identifier assigned to each app in
	// the App Store, this value is managed by Apple.
	AdamID string `db:"adam_id"`
	// AvailableCount keeps track of how many licenses are
	// available for the specific software, this value is
	// managed by Apple and tracked in the DB as a helper.
	//
	// TODO(roberto): could we omit this and rely on API errors
	// from Apple instead? seems safer unless we really need to
	// display this value in the API.
	AvailableCount uint `db:"available_count"`
	// BundleIdentifier is the unique bundle identifier of the
	// Application.
	BundleIdentifier string `db:"bundle_identifier"`
	// IconURL is the URL of this App icon
	IconURL string `db:"icon_url"`
	// Name is the user-facing name of this app.
	Name string `db:"name"`

	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

// AuthzType implements authz.AuthzTyper.
func (v *VPPApp) AuthzType() string {
	return "installable_entity"
}

// TODO(mna): It might be possible to merge this with the VPPApp struct above,
// but since it will evolve via the other PRs implemented in parallel, I'll
// create a distinct struct and we'll see at integration time.
type VPPAppStoreApp struct {
	AppStoreID    string               `db:"adam_id" json:"app_store_id"`
	Name          string               `db:"name" json:"name"`
	LatestVersion string               `db:"version" json:"latest_version"`
	Status        *VPPAppStatusSummary `db:"-" json:"status"`
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
