package fleet

// MaintainedApp represents an app in the Fleet library of maintained apps,
// as stored in the fleet_library_apps table.
type MaintainedApp struct {
	ID               uint   `json:"id" db:"id"`
	Name             string `json:"name" db:"name"`
	Token            string `json:"-" db:"token"`
	Version          string `json:"version,omitempty" db:"version"`
	Platform         string `json:"platform" db:"platform"`
	TitleID          *uint  `json:"software_title_id" db:"software_title_id"`
	InstallerURL     string `json:"url,omitempty" db:"installer_url"`
	SHA256           string `json:"-" db:"sha256"`
	BundleIdentifier string `json:"-" db:"bundle_identifier"`

	// InstallScript and UninstallScript are not stored directly in the table, they
	// must be filled via a JOIN on script_contents. On insert/update/upsert, these
	// fields are used to provide the content of those scripts.
	InstallScript   string `json:"install_script,omitempty" db:"install_script"`
	UninstallScript string `json:"uninstall_script,omitempty" db:"uninstall_script"`
}

// AuthzType implements authz.AuthzTyper.
func (s *MaintainedApp) AuthzType() string {
	return "maintained_app"
}
