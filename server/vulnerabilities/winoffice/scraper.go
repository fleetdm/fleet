package winoffice

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strings"
)

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

// ScrapeSecurityUpdates fetches and parses the Office security updates page
func ScrapeSecurityUpdates(ctx context.Context, client *http.Client) ([]SecurityRelease, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", SecurityUpdatesURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request with context: %w", err)
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
			releases = appendIfValid(releases, current)
			current = &SecurityRelease{Date: matches[1]}
			continue
		}

		if current == nil {
			continue
		}

		// Check for version info (any channel/product)
		for _, matches := range versionPattern.FindAllStringSubmatch(line, -1) {
			if strings.Contains(matches[1], "Retail") {
				continue
			}
			branch := VersionBranch{
				Version:     matches[2],
				BuildPrefix: matches[3],
				FullBuild:   matches[3] + "." + matches[4],
			}
			current.Branches = addOrUpdateBranch(current.Branches, branch)
		}

		// Check for CVE
		if matches := cvePattern.FindStringSubmatch(line); matches != nil {
			current.CVEs = append(current.CVEs, "CVE-"+matches[1])
		}
	}

	releases = appendIfValid(releases, current)

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanning: %w", err)
	}

	return releases, nil
}

// appendIfValid appends the release to the slice if it has branches and CVEs.
func appendIfValid(releases []SecurityRelease, rel *SecurityRelease) []SecurityRelease {
	if rel != nil && len(rel.Branches) > 0 && len(rel.CVEs) > 0 {
		return append(releases, *rel)
	}
	return releases
}

// addOrUpdateBranch adds a new branch or updates an existing one with the minimum build.
func addOrUpdateBranch(branches []VersionBranch, branch VersionBranch) []VersionBranch {
	for i, b := range branches {
		if b.Version != branch.Version {
			continue
		}
		// Keep the MINIMUM build since that's the lowest build containing the fix
		if compareBuildVersions(branch.FullBuild, b.FullBuild) < 0 {
			branches[i].BuildPrefix = branch.BuildPrefix
			branches[i].FullBuild = branch.FullBuild
		}
		return branches
	}
	return append(branches, branch)
}

// BuildBulletinFile creates a BulletinFile from scraped releases.
func BuildBulletinFile(releases []SecurityRelease) *BulletinFile {
	buildPrefixes := make(map[string]string)
	cveToBuilds := make(map[string]map[string]string) // CVE → version → build
	versions := make(map[string]*VersionBulletin)

	// Identify currently supported versions from the most recent release.
	// Releases are in reverse chronological order (newest first).
	currentVersions := make(map[string]bool)
	if len(releases) > 0 {
		for _, branch := range releases[0].Branches {
			currentVersions[branch.Version] = true
		}
	}

	// Collect all data from releases
	for _, rel := range releases {
		for _, branch := range rel.Branches {
			buildPrefixes[branch.BuildPrefix] = branch.Version
		}
		for _, cve := range rel.CVEs {
			recordCVEFix(cveToBuilds, cve, rel.Branches)
		}
	}

	// Build version-indexed structure with direct fixes
	for cve, fixedBuilds := range cveToBuilds {
		for version, build := range fixedBuilds {
			vb := getOrCreateVersion(versions, version)
			vb.SecurityUpdates = append(vb.SecurityUpdates, SecurityUpdate{
				CVE:               cve,
				ResolvedInVersion: OfficeVersionPrefix + build,
			})
		}
	}

	// Add upgrade paths for deprecated versions (appeared in older releases but not in the latest).
	// These versions need to upgrade to a newer version branch to get fixes.
	deprecatedVersions := findDeprecatedVersions(versions, currentVersions)
	if len(deprecatedVersions) > 0 {
		sortedVersions := sortedVersionKeys(versions)
		for _, version := range deprecatedVersions {
			vb, ok := versions[version]
			if !ok || vb == nil {
				continue
			}
			existingCVEs := make(map[string]bool)
			for _, su := range vb.SecurityUpdates {
				existingCVEs[su.CVE] = true
			}
			addUpgradePaths(vb, version, sortedVersions, cveToBuilds, existingCVEs)
		}
	}

	// Sort for deterministic output
	for _, vb := range versions {
		if vb == nil {
			continue
		}
		sort.Slice(vb.SecurityUpdates, func(i, j int) bool {
			return vb.SecurityUpdates[i].CVE < vb.SecurityUpdates[j].CVE
		})
	}

	return &BulletinFile{
		Version:       1,
		BuildPrefixes: buildPrefixes,
		Versions:      versions,
	}
}

// findDeprecatedVersions returns versions that appear in the bulletin but not in the current release.
func findDeprecatedVersions(versions map[string]*VersionBulletin, currentVersions map[string]bool) []string {
	var deprecated []string
	for version := range versions {
		if !currentVersions[version] {
			deprecated = append(deprecated, version)
		}
	}
	return deprecated
}

// sortedVersionKeys returns the keys of versions sorted in ascending order.
func sortedVersionKeys(versions map[string]*VersionBulletin) []string {
	keys := make([]string, 0, len(versions))
	for v := range versions {
		keys = append(keys, v)
	}
	sort.Strings(keys)
	return keys
}

// addUpgradePaths adds CVE fixes pointing to newer versions for deprecated versions.
func addUpgradePaths(
	vb *VersionBulletin,
	version string,
	sortedVersions []string,
	cveToBuilds map[string]map[string]string,
	existingCVEs map[string]bool,
) {
	for cve, fixedBuilds := range cveToBuilds {
		if existingCVEs[cve] {
			continue
		}
		build := findMinimumUpgrade(version, sortedVersions, fixedBuilds)
		if build == "" {
			continue
		}
		vb.SecurityUpdates = append(vb.SecurityUpdates, SecurityUpdate{
			CVE:               cve,
			ResolvedInVersion: OfficeVersionPrefix + build,
		})
	}
}

// findMinimumUpgrade finds the oldest version > current that has a fix for a CVE.
func findMinimumUpgrade(version string, sortedVersions []string, fixedBuilds map[string]string) string {
	for _, v := range sortedVersions {
		if v <= version {
			continue
		}
		if build, ok := fixedBuilds[v]; ok {
			return build
		}
	}
	return ""
}

// recordCVEFix records a CVE fix for each branch, keeping the first seen fix per version.
// Since releases are processed newest-first, this keeps the newest build. In practice,
// each CVE appears only once per version branch (in the release that fixed it).
func recordCVEFix(cveToBuilds map[string]map[string]string, cve string, branches []VersionBranch) {
	if cveToBuilds[cve] == nil {
		cveToBuilds[cve] = make(map[string]string)
	}
	for _, branch := range branches {
		if _, exists := cveToBuilds[cve][branch.Version]; !exists {
			cveToBuilds[cve][branch.Version] = branch.FullBuild
		}
	}
}

// getOrCreateVersion returns the VersionBulletin for a version, creating it if needed.
func getOrCreateVersion(versions map[string]*VersionBulletin, version string) *VersionBulletin {
	if versions[version] == nil {
		versions[version] = &VersionBulletin{}
	}
	return versions[version]
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
		// Pad to same length for proper numeric comparison
		maxLen := max(len(prefixA), len(prefixB))
		prefixA = fmt.Sprintf("%0*s", maxLen, prefixA)
		prefixB = fmt.Sprintf("%0*s", maxLen, prefixB)
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
func FetchBulletin(ctx context.Context, client *http.Client) (*BulletinFile, error) {
	releases, err := ScrapeSecurityUpdates(ctx, client)
	if err != nil {
		return nil, err
	}

	if len(releases) == 0 {
		return nil, errors.New("no releases found")
	}

	return BuildBulletinFile(releases), nil
}
