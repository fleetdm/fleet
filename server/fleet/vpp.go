package fleet

import "time"

// VPPApp represents a VPP (Volume Purchase Program) application,
// this is used by Apple MDM to manage applications via Apple
// Bussines Manager.
type VPPApp struct { // TODO(JVE): do we need the team id here?
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
