package fleet

import "time"

// MaintainedApp represets an app in the Fleet library of maintained apps,
// as stored in the fleet_library_apps table.
type MaintainedApp struct {
	ID               uint                `json:"id" db:"id"`
	Name             string              `json:"name" db:"name"`
	Token            string              `json:"-" db:"token"`
	Version          string              `json:"version" db:"version"`
	Platform         AppleDevicePlatform `json:"platform" db:"platform"`
	InstallerURL     string              `json:"-" db:"installer_url"`
	SHA256           string              `json:"-" db:"sha256"`
	BundleIdentifier string              `json:"-" db:"bundle_identifier"`

	// InstallScript and UninstallScript are not stored directly in the table, they
	// must be filled via a JOIN on script_contents. On insert/update/upsert, these
	// fields are used to provide the content of those scripts.
	InstallScript   string `json:"install_script" db:"install_script"`
	UninstallScript string `json:"uninstall_script" db:"uninstall_script"`
	// UpdatedAt is the timestamp when the fleet maintained app data was last updated.
	UpdatedAt *time.Time `json:"-" db:"updated_at"`
}

// AuthzType implements authz.AuthzTyper.
func (s *MaintainedApp) AuthzType() string {
	return "maintained_app"
}
