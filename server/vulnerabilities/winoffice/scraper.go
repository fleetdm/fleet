package winoffice

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"
)

// BulletinMaxAge is the maximum age of releases to include in the bulletin.
// Releases older than this are excluded to limit bulletin size.
const BulletinMaxAge = 3 * 365 * 24 * time.Hour // 3 years

const (
	// SecurityUpdatesURL is the Microsoft Learn page with Windows Office security updates.
	SecurityUpdatesURL = "https://learn.microsoft.com/en-us/officeupdates/microsoft365-apps-security-updates"
)

// VersionBranch represents a supported Windows Office version branch.
// Windows Office versions follow a YYMM pattern (e.g., 2602 = February 2026).
type VersionBranch struct {
	Version     string // e.g., "2602" (YYMM format)
	BuildPrefix string // e.g., "19725" (first part of build number)
	FullBuild   string // e.g., "19725.20172" (complete build number)
}

// SecurityRelease represents a single Windows Office security update release.
type SecurityRelease struct {
	Date     string          // e.g., "March 10, 2026"
	Branches []VersionBranch // All supported version branches with their fixed builds
	CVEs     []string
}

// Bulletin contains parsed Windows Office security data optimized for vulnerability matching.
type Bulletin struct {
	// BuildPrefixToVersion maps build prefix to version branch (e.g., "19725" -> "2602")
	BuildPrefixToVersion map[string]string

	// CVEToResolvedVersions maps CVE to fixed builds per version branch
	// e.g., "CVE-2026-26107" -> {"2602": "19725.20172", "2512": "19530.20260"}
	CVEToResolvedVersions map[string]map[string]string
}

// ScrapeSecurityUpdates fetches and parses the Office security updates page
func ScrapeSecurityUpdates(client *http.Client) ([]SecurityRelease, error) {
	req, err := http.NewRequest("GET", SecurityUpdatesURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	// Request markdown format
	req.Header.Set("Accept", "text/markdown")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching page: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	return parseSecurityMarkdown(resp.Body)
}

// parseSecurityMarkdown parses the markdown content
func parseSecurityMarkdown(r io.Reader) ([]SecurityRelease, error) {
	var releases []SecurityRelease
	var current *SecurityRelease

	// Patterns
	datePattern := regexp.MustCompile(`^## ([A-Z][a-z]+ \d{1,2}, \d{4})`)
	// Matches: "Current Channel: Version 2602 (Build 19725.20172)"
	// Also matches: "Monthly Enterprise Channel: Version 2512 (Build 19530.20260)"
	// Also matches: "Office LTSC 2024 Volume Licensed: Version 2408 (Build 17932.20700)"
	versionPattern := regexp.MustCompile(`([A-Za-z0-9 ]+):\s*Version\s+(\d+)\s+\(Build\s+(\d+)\.(\d+)\)`)
	cvePattern := regexp.MustCompile(`\[CVE-(\d{4}-\d+)\]`)

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()

		// Check for new release date
		if matches := datePattern.FindStringSubmatch(line); matches != nil {
			// Save previous release if exists
			if current != nil && len(current.Branches) > 0 && len(current.CVEs) > 0 {
				releases = append(releases, *current)
			}
			current = &SecurityRelease{Date: matches[1]}
			continue
		}

		if current == nil {
			continue
		}

		// Check for version info (any channel/product) - all versions are on same line
		allMatches := versionPattern.FindAllStringSubmatch(line, -1)
		for _, matches := range allMatches {
			channelOrProduct := strings.TrimSpace(matches[1])
			version := matches[2]
			buildPrefix := matches[3]
			buildSuffix := matches[4]
			fullBuild := buildPrefix + "." + buildSuffix

			// Skip retail versions that duplicate Current Channel
			if strings.Contains(channelOrProduct, "Retail") {
				continue
			}

			// Check if we already have this version branch.
			// Keep the MINIMUM build suffix since that's the lowest build containing
			// the security fix. Any build >= minimum is patched on all channels.
			found := false
			for i, b := range current.Branches {
				if b.Version == version {
					found = true
					if compareBuildVersions(fullBuild, b.FullBuild) < 0 {
						current.Branches[i].BuildPrefix = buildPrefix
						current.Branches[i].FullBuild = fullBuild
					}
					break
				}
			}
			if !found {
				current.Branches = append(current.Branches, VersionBranch{
					Version:     version,
					BuildPrefix: buildPrefix,
					FullBuild:   fullBuild,
				})
			}
		}

		// Check for CVE
		if matches := cvePattern.FindStringSubmatch(line); matches != nil {
			cve := "CVE-" + matches[1]
			current.CVEs = append(current.CVEs, cve)
		}
	}

	// Don't forget the last release
	if current != nil && len(current.Branches) > 0 && len(current.CVEs) > 0 {
		releases = append(releases, *current)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanning: %w", err)
	}

	return releases, nil
}

// parseReleaseDate parses a date string like "March 10, 2026" into a time.Time.
func parseReleaseDate(dateStr string) (time.Time, error) {
	return time.Parse("January 2, 2006", dateStr)
}

// filterRecentReleases returns only releases within the specified duration from now.
func filterRecentReleases(releases []SecurityRelease, maxAge time.Duration) []SecurityRelease {
	cutoff := time.Now().Add(-maxAge)
	var filtered []SecurityRelease
	for _, rel := range releases {
		releaseDate, err := parseReleaseDate(rel.Date)
		if err != nil {
			// If we can't parse the date, include it to be safe
			filtered = append(filtered, rel)
			continue
		}
		if releaseDate.After(cutoff) {
			filtered = append(filtered, rel)
		}
	}
	return filtered
}

// BuildBulletin creates a Bulletin from scraped releases.
// Only releases within BulletinMaxAge are included.
func BuildBulletin(releases []SecurityRelease) *Bulletin {
	// Filter to only include recent releases
	releases = filterRecentReleases(releases, BulletinMaxAge)
	bulletin := &Bulletin{
		BuildPrefixToVersion:  make(map[string]string),
		CVEToResolvedVersions: make(map[string]map[string]string),
	}

	// Process all releases to build mappings
	for _, rel := range releases {
		// Build prefix to version mapping
		for _, branch := range rel.Branches {
			bulletin.BuildPrefixToVersion[branch.BuildPrefix] = branch.Version
		}

		// CVE to fixed builds mapping
		for _, cve := range rel.CVEs {
			if bulletin.CVEToResolvedVersions[cve] == nil {
				bulletin.CVEToResolvedVersions[cve] = make(map[string]string)
			}
			for _, branch := range rel.Branches {
				// Only store if not already set (first fix wins)
				if _, exists := bulletin.CVEToResolvedVersions[cve][branch.Version]; !exists {
					bulletin.CVEToResolvedVersions[cve][branch.Version] = branch.FullBuild
				}
			}
		}
	}

	return bulletin
}

// ToBulletinFile converts a Bulletin to BulletinFile format for serialization.
func (b *Bulletin) ToBulletinFile() *BulletinFile {
	// Build version-indexed structure
	versions := make(map[string]*VersionBulletin)

	// Group CVEs by version branch (for versions that have direct fixes)
	for cve, fixedBuilds := range b.CVEToResolvedVersions {
		for version, build := range fixedBuilds {
			if versions[version] == nil {
				versions[version] = &VersionBulletin{}
			}
			versions[version].SecurityUpdates = append(versions[version].SecurityUpdates,
				SecurityUpdate{
					CVE:               cve,
					ResolvedInVersion: "16.0." + build,
				})
		}
	}

	// Get all unique versions we've ever seen
	allVersions := make(map[string]bool)
	for _, version := range b.BuildPrefixToVersion {
		allVersions[version] = true
	}

	// Get sorted list of all versions with fixes (for finding minimum upgrade path)
	var sortedVersionsWithFixes []string
	for version := range allVersions {
		sortedVersionsWithFixes = append(sortedVersionsWithFixes, version)
	}
	sort.Strings(sortedVersionsWithFixes)

	// For each version, add CVEs it doesn't have a direct fix for.
	// Use the fix from the oldest version > this one that has a fix (minimum upgrade path).
	for version := range allVersions {
		// Ensure version entry exists
		if versions[version] == nil {
			versions[version] = &VersionBulletin{}
		}

		// Build set of CVEs this version already has fixes for
		existingCVEs := make(map[string]bool)
		for _, su := range versions[version].SecurityUpdates {
			existingCVEs[su.CVE] = true
		}

		// Add all CVEs this version doesn't have a fix for
		for cve, fixedBuilds := range b.CVEToResolvedVersions {
			if existingCVEs[cve] {
				continue // Already has a fix for this CVE
			}

			// Find fix from oldest version that's newer than this version
			// (minimum upgrade path - can't recommend a downgrade)
			var fixedBuild string
			for _, otherVersion := range sortedVersionsWithFixes {
				if otherVersion <= version {
					continue // Skip versions that aren't an upgrade
				}
				if build, ok := fixedBuilds[otherVersion]; ok {
					fixedBuild = "16.0." + build
					break
				}
			}

			// Skip if no valid upgrade path found (this version is newest with no fix)
			if fixedBuild == "" {
				continue
			}

			// Add CVE with the fix from the upgrade target version
			versions[version].SecurityUpdates = append(versions[version].SecurityUpdates,
				SecurityUpdate{
					CVE:               cve,
					ResolvedInVersion: fixedBuild,
				})
		}
	}

	// Sort security updates within each version for deterministic output
	for _, vb := range versions {
		sort.Slice(vb.SecurityUpdates, func(i, j int) bool {
			return vb.SecurityUpdates[i].CVE < vb.SecurityUpdates[j].CVE
		})
	}

	return &BulletinFile{
		Version:       1,
		BuildPrefixes: b.BuildPrefixToVersion,
		Versions:      versions,
	}
}

// compareBuildVersions compares two build versions like "19725.20172"
// Returns: -1 if a < b, 0 if a == b, 1 if a > b
func compareBuildVersions(a, b string) int {
	partsA := strings.Split(a, ".")
	partsB := strings.Split(b, ".")

	// Compare build prefix (major)
	if len(partsA) >= 1 && len(partsB) >= 1 {
		prefixA := partsA[0]
		prefixB := partsB[0]
		if prefixA < prefixB {
			return -1
		}
		if prefixA > prefixB {
			return 1
		}
	}

	// Compare build suffix (minor)
	if len(partsA) >= 2 && len(partsB) >= 2 {
		suffixA := partsA[1]
		suffixB := partsB[1]
		// Pad to same length for proper comparison
		maxLen := max(len(suffixA), len(suffixB))
		suffixA = fmt.Sprintf("%0*s", maxLen, suffixA)
		suffixB = fmt.Sprintf("%0*s", maxLen, suffixB)
		if suffixA < suffixB {
			return -1
		}
		if suffixA > suffixB {
			return 1
		}
	}

	return 0
}

// FetchBulletin scrapes and builds Office security bulletin
func FetchBulletin(client *http.Client) (*Bulletin, error) {
	releases, err := ScrapeSecurityUpdates(client)
	if err != nil {
		return nil, err
	}

	if len(releases) == 0 {
		return nil, errors.New("no releases found")
	}

	return BuildBulletin(releases), nil
}
