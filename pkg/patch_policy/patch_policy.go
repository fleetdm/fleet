package patch_policy

import (
	"errors"
	"fmt"
	"strings"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

type PolicyData struct {
	Name        string
	Platform    string
	Description string
	Resolution  string
	Query       string
	ExistsQuery string
	Version     string
}

const (
	// notExistsStart is prepended to the exists query body; version_compare is appended
	// to the same WHERE clause (inside NOT EXISTS), matching the pre-#45647 generator.
	notExistsStart = "SELECT 1 WHERE NOT EXISTS ("
	existsPrefix   = "SELECT 1 FROM "
)

var (
	ErrWrongPlatform = errors.New("platform should be darwin or windows")
	ErrNoExistsQuery = errors.New("exists query was not provided")
)

// GenerateQueryForManifest wraps the "exists" query to create a patch policy query.
func GenerateQueryForManifest(p PolicyData) (string, error) {
	if p.ExistsQuery == "" {
		return "", ErrNoExistsQuery
	}

	suffix, err := versionCompareSuffix(p.Platform, p.ExistsQuery, p.Version)
	if err != nil {
		return "", err
	}

	before, _ := strings.CutSuffix(p.ExistsQuery, ";")
	before = strings.TrimSpace(before)
	if strings.Contains(before, " OR ") {
		before = parenthesizeWhereClause(before)
	}

	return notExistsStart + before + suffix, nil
}

// parenthesizeWhereClause wraps the WHERE body in parens when it contains OR so that
// the trailing AND version_compare(...) binds to the full predicate, not just the
// right-hand side of OR (SQL precedence: AND > OR).
func parenthesizeWhereClause(existsQuery string) string {
	if !strings.HasPrefix(existsQuery, existsPrefix) {
		return existsQuery
	}
	rest := strings.TrimPrefix(existsQuery, existsPrefix)
	table, conditions, found := strings.Cut(rest, " WHERE ")
	if !found {
		return existsQuery
	}
	if !strings.Contains(conditions, " OR ") {
		return existsQuery
	}
	return existsPrefix + table + " WHERE (" + conditions + ")"
}

func versionCompareSuffix(platform, existsQuery, version string) (string, error) {
	column, err := versionCompareColumn(platform, existsQuery)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(" AND version_compare(%s, '%s') < 0);", column, version), nil
}

func versionCompareColumn(platform, existsQuery string) (string, error) {
	switch platform {
	case "darwin":
		return "bundle_short_version", nil
	case "windows":
		if tableFromExistsQuery(existsQuery) == "file" {
			return "file_version", nil
		}
		return "version", nil
	default:
		return "", ErrWrongPlatform
	}
}

func tableFromExistsQuery(existsQuery string) string {
	trimmed, _ := strings.CutSuffix(strings.TrimSpace(existsQuery), ";")
	if !strings.HasPrefix(trimmed, existsPrefix) {
		return ""
	}
	rest := strings.TrimPrefix(trimmed, existsPrefix)
	if table, _, found := strings.Cut(rest, " WHERE "); found {
		return table
	}
	if table, _, found := strings.Cut(rest, " "); found {
		return table
	}
	return strings.TrimSpace(rest)
}

// GenerateFromInstaller creates a patch policy with all fields from an installer
func GenerateFromInstaller(p PolicyData, installer *fleet.SoftwareInstaller) (*PolicyData, error) {
	// use the patch policy query from the app manifest if available
	query := installer.PatchQuery

	if p.Description == "" {
		p.Description = "Outdated software might introduce security vulnerabilities or compatibility issues."
	}

	if p.Resolution == "" {
		p.Resolution = "Install the latest version from self-service."
	}

	switch installer.Platform {
	case "darwin":
		if p.Name == "" {
			p.Name = fmt.Sprintf("macOS - %s up to date", installer.SoftwareTitle)
		}
		if installer.PatchQuery == "" {
			query = defaultMacOSQuery(installer.BundleIdentifier, installer.Version)
		}
	case "windows":
		if p.Name == "" {
			p.Name = fmt.Sprintf("Windows - %s up to date", installer.SoftwareTitle)
		}
		if installer.PatchQuery == "" {
			query = defaultWindowsQuery(installer.SoftwareTitle, installer.Version)
		}
	default:
		return nil, ErrWrongPlatform
	}

	return &PolicyData{Query: query, Platform: installer.Platform, Name: p.Name, Description: p.Description, Resolution: p.Resolution}, nil
}

func defaultMacOSQuery(bundleIdentifier string, version string) string {
	patchTemplate := "SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM apps WHERE bundle_identifier = '%s' AND version_compare(bundle_short_version, '%s') < 0);"
	return fmt.Sprintf(patchTemplate, bundleIdentifier, version)
}

func defaultWindowsQuery(softwareTitle string, version string) string {
	patchTemplate := "SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM programs WHERE name = '%s' AND version_compare(version, '%s') < 0);"
	return fmt.Sprintf(patchTemplate, softwareTitle, version)
}

// GenerateOpenQuery returns a pre-install query that returns a row only when the app is closed.
func GenerateOpenQuery(platform string, bundleIdentifier string, softwareTitle string) string {
	switch platform {
	case "darwin":
		return defaultMacOSOpenQuery(bundleIdentifier)
	case "windows":
		return defaultWindowsOpenQuery(softwareTitle)
	default:
		return ""
	}
}

func defaultMacOSOpenQuery(bundleIdentifier string) string {
	// Resolve the app's install path from its bundle identifier via the apps table, then
	// match any process running from inside that path.
	// alternatives considered:
	// - get processes by name - requires a lot of manual overrides
	// - use the running_apps table - not reliable when run through orbit
	// - use the "app" artifact in the homebrew cask - requires extra code to extract
	openTemplate := "SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM apps a JOIN processes p ON p.path LIKE concat(a.path, '/%%') WHERE a.bundle_identifier = '%s');"
	return fmt.Sprintf(openTemplate, escapeSQLLiteral(bundleIdentifier))
}

func defaultWindowsOpenQuery(softwareTitle string) string {
	if query, ok := windowsOpenQueryOverrides[softwareTitle]; ok {
		windowsOpenQueryPrefix := "SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM processes WHERE LOWER(name) %s);"
		return fmt.Sprintf(windowsOpenQueryPrefix, query)
	}

	// Match a process named "<title>.exe"
	// alternatives considered:
	// - join programs.install_location with processes.path - install_location is unreliable (especially for MSI installers)
	executable := strings.ToLower(softwareTitle) + ".exe"
	openTemplate := "SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM processes WHERE LOWER(name) = '%s');"
	return fmt.Sprintf(openTemplate, escapeSQLLiteral(executable))
}

func escapeSQLLiteral(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

// overrides based on uninstall scripts
var windowsOpenQueryOverrides = map[string]string{ //nolint:gosec // G101 false positive: values are app process names, not credentials
	"1Password":                    "LIKE '1password%'",
	"7-zip":                        "IN ('7zfm.exe','7zg.exe')",
	"Amazon Chime":                 "IN ('amazon chime.exe','chime.exe')",
	"Android Studio":               "= 'studio64.exe'",
	"Beyond Compare":               "= 'bcompare.exe'",
	"CLion":                        "IN ('clion.exe','clion64.exe')",
	"DataGrip":                     "IN ('datagrip.exe','datagrip64.exe')",
	"DataSpell":                    "IN ('dataspell.exe','dataspell64.exe')",
	"DAX Studio":                   "= 'daxstudio.exe'",
	"DBeaverEE":                    "= 'dbeaver.exe'",
	"DBeaverLite":                  "= 'dbeaver.exe'",
	"DBeaverUltimate":              "= 'dbeaver.exe'",
	"Dell Command Update":          "IN ('dellcommandupdate.exe','dcu-cli.exe')",
	"GoLand":                       "IN ('goland.exe','goland64.exe')",
	"Google Antigravity IDE":       "= 'antigravity.exe'",
	"Google Chrome":                "= 'chrome.exe'",
	"IntelliJ IDEA CE":             "IN ('idea.exe','idea64.exe')",
	"IntelliJ IDEA Ultimate":       "IN ('idea.exe','idea64.exe')",
	"JetBrains Toolbox":            "IN ('toolbox.exe','jetbrains-toolbox.exe')",
	"KNIME Analytics Platform":     "= 'knime.exe'",
	"Lenovo Dock Manager":          "= 'dockmgr.exe'",
	"Microsoft Edge":               "= 'msedge.exe'",
	"Microsoft Remote Help":        "= 'remotehelp.exe'",
	"Microsoft Teams":              "IN ('teams.exe','ms-teams.exe')",
	"Microsoft Visual Studio Code": "= 'code.exe'",
	"Node.js":                      "= 'node.exe'",
	"Notion Calendar":              "IN ('cron.exe','notion calendar.exe')",
	"OBS":                          "IN ('obs32.exe','obs64.exe')",
	"Okta Verify":                  "= 'oktaverify.exe'",
	"Ollama":                       "IN ('ollama.exe','ollama app.exe')",
	"OneDrive":                     "LIKE 'onedrive%'",
	"Pale Moon":                    "= 'palemoon.exe'",
	"pgAdmin 4":                    "= 'pgadmin4.exe'",
	"PhpStorm":                     "IN ('phpstorm.exe','phpstorm64.exe')",
	"Plantronics Hub":              "= 'plthub.exe'",
	"Portfolio Performance":        "= 'portfolioperformance.exe'",
	"Power Automate":               "= 'pad.console.host.exe'",
	"PowerShell":                   "= 'pwsh.exe'",
	"ProtonVPN":                    "IN ('proton vpn.exe','protonvpn.exe')",
	"PyCharm Community Edition":    "IN ('pycharm.exe','pycharm64.exe')",
	"PyCharm Professional":         "IN ('pycharm.exe','pycharm64.exe')",
	"Rider":                        "IN ('rider.exe','rider64.exe')",
	"RStudio":                      "IN ('rgui.exe','rsession.exe' 'rstudio.exe')",
	"RubyMine":                     "IN ('rubymine.exe','rubymine64.exe')",
	"RustRover":                    "IN ('rustrover.exe','rustrover64.exe')",
	"Spotify":                      "IN ('spotify.exe','spotifywebhelper.exe')",
	"Sublime Text":                 "= 'sublime_text.exe'",
	"VirtualBox":                   "LIKE 'virtualbox%'",
	"Wacom Tablet":                 "IN ('wacomdesktopcenter.exe','wacom_tablet.exe')",
	"WebStorm":                     "IN ('webstorm.exe','webstorm64.exe')",
	"Windows App":                  "= 'windowsapp.exe'",
}
