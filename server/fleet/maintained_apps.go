package fleet

// MaintainedApp represents an app in the Fleet library of maintained apps,
// as stored in the fleet_library_apps table.
type MaintainedApp struct {
	ID               uint   `json:"id" db:"id"`
	Name             string `json:"name" db:"name"`
	Slug             string `json:"slug" db:"slug"`
	Version          string `json:"version,omitempty"`
	Platform         string `json:"platform" db:"platform"`
	TitleID          *uint  `json:"software_title_id" db:"software_title_id"`
	InstallerURL     string `json:"url,omitempty"`
	SHA256           string `json:"-"`
	UniqueIdentifier string `json:"-" db:"unique_identifier"`
	InstallScript    string `json:"install_script,omitempty"`
	UninstallScript  string `json:"uninstall_script,omitempty"`
}

func (s *MaintainedApp) Source() string {
	if s.Platform == "windows" {
		return "programs"
	}

	return "apps"
}

func (s *MaintainedApp) BundleIdentifier() string {
	if s.Platform == "windows" {
		return ""
	}

	return s.UniqueIdentifier
}

// AuthzType implements authz.AuthzTyper.
func (s *MaintainedApp) AuthzType() string {
	return "maintained_app"
}
