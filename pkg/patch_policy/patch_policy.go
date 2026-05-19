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
	templateStart      = "SELECT 1 WHERE NOT EXISTS ("
	templateEndDarwin  = " AND version_compare(bundle_short_version, '%s') < 0);"
	templateEndWindows = " AND version_compare(version, '%s') < 0);"
)

var (
	ErrWrongPlatform   = errors.New("platform should be darwin or windows")
	ErrNoExistsQuery   = errors.New("exists query was not provided")
	ErrNoWhereInExists = errors.New("exists query is missing a WHERE clause")
)

// GenerateQueryForManifest wraps the "exists" query to create a patch policy query.
// The WHERE body of the exists query is wrapped in parentheses so that any OR in
// the body binds before the appended AND version_compare(...) clause (SQL operator
// precedence: AND > OR).
func GenerateQueryForManifest(p PolicyData) (string, error) {
	if p.ExistsQuery == "" {
		return "", ErrNoExistsQuery
	}
	before, _ := strings.CutSuffix(p.ExistsQuery, ";")
	// Escape any literal '%' in the exists query (e.g. SQL LIKE patterns)
	// so fmt.Sprintf doesn't interpret them as format verbs.
	before = strings.ReplaceAll(before, "%", "%%")

	const whereKeyword = " WHERE "
	idx := strings.Index(before, whereKeyword)
	if idx < 0 {
		return "", ErrNoWhereInExists
	}
	wrapped := before[:idx+len(whereKeyword)] + "(" + before[idx+len(whereKeyword):] + ")"

	switch p.Platform {
	case "darwin":
		return fmt.Sprintf(templateStart+wrapped+templateEndDarwin, p.Version), nil
	case "windows":
		return fmt.Sprintf(templateStart+wrapped+templateEndWindows, p.Version), nil
	}
	return "", ErrWrongPlatform
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
