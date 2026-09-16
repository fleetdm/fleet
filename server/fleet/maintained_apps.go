package fleet

import (
	"net/http"
	"slices"
)

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
	AutomaticInstallQuery string   `json:"automatic_install_query,omitempty" db:"pre_install_query"` //nolint:apiparamcheck // SQL query for automatic install
	Categories            []string `json:"categories"`
	UpgradeCode           string   `json:"upgrade_code,omitempty" db:"upgrade_code"`
	PatchQuery            string   `json:"-" db:"patch_query"`
	AppOpenQuery          string   `json:"-" db:"app_open_query"`

	// TitleName is the name of the software title this app's installer owns, which is
	// not necessarily Name: a Windows app's title is never renamed when the catalog
	// name changes (the darwin reconcile passes are platform-scoped, and the installer
	// only renames when an upgrade code is present). Windows software ingestion merges
	// onto TitleID, and this is that title's name.
	TitleName string `json:"-" db:"title_name"`
}

// WinMatchPrefixes returns the candidate program-name prefixes for matching a reported
// Windows program name onto this app's software title, longest first so the most
// specific match wins, deduplicated and without blanks.
//
// Windows programs embed the version in programs.name (e.g. "Granola 7.373.2"), so a
// prefix match is the only join key available. All three names are candidates because
// none alone is reliable: osquery reports "CPUID CPU-Z ..." for the app Fleet calls
// "CPU-Z" (only UniqueIdentifier works); some apps.json entries carry a version-bearing
// identifier frozen at the version current when they were added, e.g. "Notion 6.1.0"
// (only Name works); and where the catalog name has drifted from the title, TitleName is
// what inventory most likely reports under, being what the app was called when added.
//
// Returns nothing unless this is a Windows app with a resolved title, since there is
// nothing to merge onto otherwise. Note that this depends on Platform and TitleID being
// populated: a caller loading a partial MaintainedApp must select both.
//
// Given (UniqueIdentifier, Name, TitleName), it returns:
//
//	("Granola", "Granola", "Granola")                  -> ["Granola"]
//	("CPUID CPU-Z", "CPU-Z", "CPU-Z")                   -> ["CPUID CPU-Z", "CPU-Z"]
//	("Notion 6.1.0", "Notion", "Notion")                -> ["Notion 6.1.0", "Notion"]
//	("Zoom", "Zoom Workplace", "Zoom")                  -> ["Zoom Workplace", "Zoom"]
//	("Box", "Box Drive", "Box Drive")                   -> ["Box Drive", "Box"]
//	("", "Granola", "")                                 -> ["Granola"]
//
// The first collapses to one entry because all three names agree. The second and third
// show why all three are kept: only the identifier matches "CPUID CPU-Z 2.16", and only
// the name matches "Notion 7.2.0". The fourth is a catalog rename, where the title still
// carries the older name inventory reports under. Ordering is longest first, so
// "Box Drive 2.0" matches the more specific "Box Drive" rather than "Box".
func (s *MaintainedApp) WinMatchPrefixes() []string {
	if s.Platform != "windows" || s.TitleID == nil {
		return nil
	}

	candidates := []string{s.UniqueIdentifier, s.Name, s.TitleName}

	prefixes := make([]string, 0, len(candidates))
	for _, c := range candidates {
		if c == "" || slices.Contains(prefixes, c) {
			continue
		}
		prefixes = append(prefixes, c)
	}
	slices.SortStableFunc(prefixes, func(a, b string) int { return len(b) - len(a) })
	return prefixes
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
