package patch_policy

import (
	"errors"
	"fmt"
	"strings"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

type PolicyData struct {
	Name        string
	Query       string
	Platform    string
	Description string
	Resolution  string
	// Information required for query
	SoftwareTitle    string
	BundleIdentifier string
	Version          string
}

const versionVariable = "$FMA_VERSION"

var (
	ErrWrongPlatform = errors.New("platform should be darwin or windows")
)

// GenerateQueryForManifest either creates a default query or replaces the $FMA_VERSION variable in a given one
func GenerateQueryForManifest(p PolicyData) (string, error) {
	if p.Query != "" {
		// Version is extracted from the manifest so this should be safe to run as an osquery query
		return strings.ReplaceAll(p.Query, versionVariable, p.Version), nil
	}

	switch p.Platform {
	case "darwin":
		if p.Query == "" {
			return defaultMacOSQuery(p.BundleIdentifier, p.Version), nil
		}
	case "windows":
		if p.Query == "" {
			return defaulWindowsQuery(p.SoftwareTitle, p.Version), nil
		}
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
			query = defaulWindowsQuery(installer.SoftwareTitle, installer.Version)
		}
	default:
		return nil, ErrWrongPlatform
	}

	return &PolicyData{Query: query, Platform: installer.Platform, Name: p.Name, Description: p.Description, Resolution: p.Resolution}, nil
}

func defaulWindowsQuery(softwareTitle, version string) string {
	// TODO: use upgrade code to improve accuracy?
	return fmt.Sprintf(
		"SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM programs WHERE name = '%s' AND version_compare(version, '%s') < 0);",
		softwareTitle,
		version,
	)
}

func defaultMacOSQuery(bundleIdentifier, version string) string {
	return fmt.Sprintf(
		"SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM apps WHERE bundle_identifier = '%s' AND version_compare(bundle_short_version, '%s') < 0);",
		bundleIdentifier,
		version,
	)
}
