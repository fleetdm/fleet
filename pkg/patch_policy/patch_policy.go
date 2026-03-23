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
	TitleName   string
	Version     string
}

const versionVariable = "$FMA_VERSION"

var (
	ErrEmptyQuery    = errors.New("query should not be empty")
	ErrWrongPlatform = errors.New("platform should be darwin or windows")
)

// GenerateFromManifest replaces the $FMA_VERSION variable and checks platform
func GenerateFromManifest(p PolicyData) (string, error) {
	// TODO: generate default query here so it will be in all manifests? if so we
	// need to add an empty query check serverside in case an old manifest with no patch query
	// is available
	if p.Query == "" {
		return "", ErrEmptyQuery
	}
	// Version is extracted from the manifest so this should be safe to run as an osquery query
	query := strings.ReplaceAll(p.Query, versionVariable, p.Version)

	switch p.Platform {
	case "darwin":
	case "windows":
	default:
		return "", ErrWrongPlatform
	}

	return query, nil
}

// GenerateFromInstaller creates a patch policy with all fields from an installer
func GenerateFromInstaller(p PolicyData, installer *fleet.SoftwareInstaller) (*PolicyData, error) {
	var query string

	if p.Description == "" {
		p.Description = "Outdated software might introduce security vulnerabilities or compatibility issues."
	}

	if p.Resolution == "" {
		p.Resolution = "Install the latest version from self-service."
	}

	switch p.Platform {
	case "darwin":
		if p.Name == "" {
			p.Name = fmt.Sprintf("macos - %s up to date", installer.SoftwareTitle)
		}
		if installer.PatchQuery == "" {
			// TODO: what if there's an override query? Whoever calls this will have to find out
			query = fmt.Sprintf(
				"SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM apps WHERE bundle_identifier = '%s' AND version_compare(bundle_short_version, '%s') < 0);",
				installer.BundleIdentifier,
				installer.Version,
			)
		}
	case "windows":
		if p.Name == "" {
			p.Name = fmt.Sprintf("windows - %s up to date", installer.SoftwareTitle)
		}
		if installer.PatchQuery == "" {
			// TODO: use upgrade code to improve accuracy?
			query = fmt.Sprintf(
				"SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM programs WHERE name = '%s' AND version_compare(bundle_short_version, '%s') < 0);",
				installer.SoftwareTitle,
				installer.Version,
			)
		}
	default:
		return nil, ErrWrongPlatform
	}

	return &PolicyData{Name: p.Name, Query: query, Description: p.Description, Resolution: p.Resolution, Platform: p.Platform}, nil
}
