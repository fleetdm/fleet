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
	whereIdx := strings.Index(rest, " WHERE ")
	if whereIdx < 0 {
		return existsQuery
	}
	table := rest[:whereIdx]
	conditions := rest[whereIdx+len(" WHERE "):]
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
	if whereIdx := strings.Index(rest, " WHERE "); whereIdx >= 0 {
		return rest[:whereIdx]
	}
	if space := strings.IndexByte(rest, ' '); space >= 0 {
		return rest[:space]
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
