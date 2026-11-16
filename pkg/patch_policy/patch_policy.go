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
	// ParenthesizedBuild marks a macOS app whose bundle_short_version carries its
	// build in parentheses (e.g. Ableton Live's "12.4.16 (2026-10-30_e33287a1c7)")
	// and whose manifest version is the release alone.
	ParenthesizedBuild bool
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

	suffix, err := versionCompareSuffix(p.Platform, p.ExistsQuery, p.Version, p.ParenthesizedBuild)
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

func versionCompareSuffix(platform, existsQuery, version string, parenthesizedBuild bool) (string, error) {
	column, err := versionCompareColumn(platform, existsQuery)
	if err != nil {
		return "", err
	}
	if platform == "darwin" && (parenthesizedBuild || strings.Contains(version, "(")) {
		column, version = stripParenthesizedBuild(column, version)
	}
	return fmt.Sprintf(" AND %s < 0);", versionCompare(column, version)), nil
}

// stripParenthesizedBuild compares only the release before a parenthesized build
// ("7.2.2 (88465)" and "2.4.3(2EEA)" become "7.2.2" and "2.4.3"). version_compare
// doesn't split on spaces or parentheses, and a build kept as a dotted segment would
// rank "5.8(0920)" above "5.8.2". Unlike split, regex_match returns an empty string,
// not NULL, for an empty version; version_compare rejects NULL and fails the query.
func stripParenthesizedBuild(column, version string) (string, string) {
	if i := strings.IndexAny(version, " ("); i > 0 {
		version = version[:i]
	}
	return fmt.Sprintf("regex_match(%s, '^[^ (]*', 0)", column), version
}

// versionCompare returns a version_compare call for column against version. osquery
// doesn't treat "+" as a delimiter, so a "+build" segment compares by string length
// ("0.4.25+10" sorts above "0.4.26+4"); make "+" a dot on both sides instead.
func versionCompare(column, version string) string {
	if strings.Contains(version, "+") {
		return fmt.Sprintf("version_compare(REPLACE(%s, '+', '.'), '%s')", column, escapeSQLLiteral(strings.ReplaceAll(version, "+", ".")))
	}
	return fmt.Sprintf("version_compare(%s, '%s')", column, escapeSQLLiteral(version))
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
	column := "bundle_short_version"
	if strings.Contains(version, "(") {
		column, version = stripParenthesizedBuild(column, version)
	}
	patchTemplate := "SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM apps WHERE bundle_identifier = '%s' AND %s < 0);"
	return fmt.Sprintf(patchTemplate, escapeSQLLiteral(bundleIdentifier), versionCompare(column, version))
}

func defaultWindowsQuery(softwareTitle string, version string) string {
	patchTemplate := "SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM programs WHERE name = '%s' AND %s < 0);"
	return fmt.Sprintf(patchTemplate, escapeSQLLiteral(softwareTitle), versionCompare("version", version))
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
	// Resolve the app's install path and CFBundleExecutable from its bundle identifier via
	// the apps table, then match only the app's own executable. Matching any process inside
	// the bundle's subtree reports apps with in-bundle login items or background helpers
	// (e.g. 1Password's browser helper) as permanently open. Bundles without a
	// CFBundleExecutable report as closed, matching what the subtree join did for them.
	// alternatives considered:
	// - get processes by name - requires a lot of manual overrides
	// - use the running_apps table - not reliable when run through orbit
	// - use the "app" artifact in the homebrew cask - requires extra code to extract
	openTemplate := "SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM apps a JOIN processes p ON p.path = concat(a.path, '/Contents/MacOS/', a.bundle_executable) WHERE a.bundle_identifier = '%s' AND a.bundle_executable != '');"
	return fmt.Sprintf(openTemplate, escapeSQLLiteral(bundleIdentifier))
}

func defaultWindowsOpenQuery(softwareTitle string) string {
	// Match a process named "<title>.exe". Apps whose process is named differently set
	// open_process_names or open_process_name_prefix in their winget input manifest.
	// alternatives considered:
	// - join programs.install_location with processes.path - install_location is unreliable (especially for MSI installers)
	executable := strings.ToLower(softwareTitle) + ".exe"
	openTemplate := "SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM processes WHERE LOWER(name) = '%s');"
	return fmt.Sprintf(openTemplate, escapeSQLLiteral(executable))
}

// windowsARMHostQuery matches Windows hosts whose native CPU is ARM64. osquery reports
// cpu_type "ARM" there, natively or under x64 emulation, and no other Windows CPU type
// starts with "ARM".
const windowsARMHostQuery = "SELECT 1 FROM system_info WHERE cpu_type LIKE 'ARM%'"

// ScopeToWindowsARM wraps a Windows policy query so it passes on hosts that are not
// ARM64. An arm64 installer can't run on x64, so its automatic-install and patch
// policies must not fail there; x64 installers run on ARM through emulation and need
// no scoping. Apply it after GenerateQueryForManifest, which parses the exists query.
func ScopeToWindowsARM(query string) string {
	query, _ = strings.CutSuffix(strings.TrimSpace(query), ";")
	if query == "" {
		return ""
	}
	return fmt.Sprintf("SELECT 1 WHERE EXISTS (%s) OR NOT EXISTS (%s);", query, windowsARMHostQuery)
}

func escapeSQLLiteral(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}
