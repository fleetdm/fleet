package fleet

import "net/http"

// MaintainedApp represents an app in the Fleet library of maintained apps
type MaintainedApp struct {
	ID                    uint     `json:"id" db:"id"`
	Name                  string   `json:"name" db:"name"`
	Slug                  string   `json:"slug" db:"slug"`
	Version               string   `json:"version,omitempty" db:"version"`
	Platform              string   `json:"platform" db:"platform"`
	TitleID               *uint    `json:"software_title_id" db:"software_title_id"`
	InstallerURL          string   `json:"url,omitempty" db:"url"`
	SHA256                string   `json:"-" db:"storage_id"`
	UniqueIdentifier      string   `json:"-" db:"unique_identifier"`
	InstallScript         string   `json:"install_script,omitempty" db:"install_script"`
	UninstallScript       string   `json:"uninstall_script,omitempty" db:"uninstall_script"`
	AutomaticInstallQuery string   `json:"-" db:"pre_install_query"`
	Categories            []string `json:"categories"`
	UpgradeCode           string   `json:"upgrade_code,omitempty" db:"upgrade_code"`
	PatchQuery            string   `json:"-" db:"patch_query"`
	AppOpenQuery          string   `json:"-" db:"app_open_query"`
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

// MaintainedAppListOptions contains the options for listing Fleet-maintained
// apps. Pagination operates on distinct app names (an app's macOS and Windows
// entries are combined into a single row in the UI), so an app is never split
// across a page boundary. The count, however, is the total number of
// installable apps, with each platform entry counted separately.
type MaintainedAppListOptions struct {
	ListOptions

	// Platform optionally filters to apps that have an entry on the given
	// platform ("darwin" or "windows"); an empty value returns all platforms.
	// This restricts which apps appear (and the count), not which platform rows
	// are returned: every platform entry of a matching app is still included so
	// the UI can render all of an app's platforms.
	Platform string

	// AvailableOnly, when true, returns only apps that have not yet been added
	// to the team (the "Hide added apps" filter). It has no effect when no team
	// is specified, since the added/available distinction is team-scoped.
	AvailableOnly bool
}

// NoMaintainedAppsInDatabaseError is the error type for no Fleet Maintained Apps in the database
type NoMaintainedAppsInDatabaseError struct {
	ErrorWithUUID
}

// Error implements the error interface.
func (e *NoMaintainedAppsInDatabaseError) Error() string {
	return `Fleet was unable to ingest the maintained apps list. Run fleetctl trigger name=maintained_apps to try repopulating the apps list.`
}

// StatusCode implements the go-kit http StatusCoder interface.
func (e *NoMaintainedAppsInDatabaseError) StatusCode() int {
	return http.StatusNotFound
}

func (e *NoMaintainedAppsInDatabaseError) Is(target error) bool {
	return target.Error() == e.Error()
}
