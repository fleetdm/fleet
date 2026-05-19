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

const existsQueryPrefix = "SELECT 1 FROM "

var (
	ErrWrongPlatform = errors.New("platform should be darwin or windows")
	ErrNoExistsQuery = errors.New("exists query was not provided")
)

// GenerateQueryForManifest wraps the "exists" query to create a patch policy query.
// The exists query must be of the form: SELECT 1 FROM <table> WHERE <conditions>;
// The patched query moves version_compare into the NOT EXISTS subquery WHERE clause.
func GenerateQueryForManifest(p PolicyData) (string, error) {
	if p.ExistsQuery == "" {
		return "", ErrNoExistsQuery
	}

	var versionCompare string
	switch p.Platform {
	case "darwin":
		versionCompare = fmt.Sprintf("version_compare(bundle_short_version, '%s') < 0", p.Version)
	case "windows":
		versionCompare = fmt.Sprintf("version_compare(version, '%s') < 0", p.Version)
	default:
		return "", ErrWrongPlatform
	}

	before, _ := strings.CutSuffix(p.ExistsQuery, ";")
	before = strings.TrimSpace(before)
	if !strings.HasPrefix(before, existsQueryPrefix) {
		return "", fmt.Errorf("exists query must start with %q", existsQueryPrefix)
	}

	rest := strings.TrimPrefix(before, existsQueryPrefix)
	whereIdx := strings.Index(rest, " WHERE ")
	if whereIdx < 0 {
		return "", errors.New("exists query must contain a WHERE clause")
	}

	table := rest[:whereIdx]
	conditions := rest[whereIdx+len(" WHERE "):]
	// Parenthesize OR conditions so AND binds to version_compare before OR.
	if strings.Contains(conditions, " OR ") {
		conditions = "(" + conditions + ")"
	}

	return fmt.Sprintf(
		"SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM %s WHERE %s AND %s);",
		table, conditions, versionCompare,
	), nil
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
