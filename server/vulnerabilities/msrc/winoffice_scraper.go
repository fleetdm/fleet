package msrc

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strings"

	msrcapps "github.com/fleetdm/fleet/v4/server/vulnerabilities/msrc/apps"
)

const (
	// WinOfficeSecurityUpdatesURL is the Microsoft Learn page with Windows Office security updates.
	// This covers Microsoft 365 Apps, Office LTSC 2024/2021, and Office 2019.
	WinOfficeSecurityUpdatesURL = "https://learn.microsoft.com/en-us/officeupdates/microsoft365-apps-security-updates"
)

// WinOfficeVersionBranch represents a supported Windows Office version branch.
// Windows Office versions follow a YYMM pattern (e.g., 2602 = February 2026).
type WinOfficeVersionBranch struct {
	Version     string // e.g., "2602" (YYMM format)
	BuildPrefix string // e.g., "19725" (first part of build number)
	FullBuild   string // e.g., "19725.20172" (complete build number)
}

// WinOfficeSecurityRelease represents a single Windows Office security update release.
type WinOfficeSecurityRelease struct {
	Date     string                   // e.g., "March 10, 2026"
	Branches []WinOfficeVersionBranch // All supported version branches with their fixed builds
	CVEs     []string
}

// WinOfficeBulletin contains parsed Windows Office security data optimized for vulnerability matching.
// This covers Microsoft 365 Apps, Office LTSC 2024/2021, and Office 2019 on Windows.
type WinOfficeBulletin struct {
	// BuildPrefixToVersion maps build prefix to version branch (e.g., "19725" -> "2602")
	BuildPrefixToVersion map[string]string

	// CVEToFixedBuilds maps CVE to fixed builds per version branch
	// e.g., "CVE-2026-26107" -> {"2602": "19725.20172", "2512": "19530.20260"}
	CVEToFixedBuilds map[string]map[string]string

	// SupportedVersions lists currently supported version branches (from most recent update)
	SupportedVersions []string
}

// ScrapeWinOfficeSecurityUpdates fetches and parses the Office security updates page
func ScrapeWinOfficeSecurityUpdates(client *http.Client) ([]WinOfficeSecurityRelease, error) {
	req, err := http.NewRequest("GET", WinOfficeSecurityUpdatesURL, nil)
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

	return parseWinOfficeSecurityMarkdown(resp.Body)
}

// parseWinOfficeSecurityMarkdown parses the markdown content
func parseWinOfficeSecurityMarkdown(r io.Reader) ([]WinOfficeSecurityRelease, error) {
	var releases []WinOfficeSecurityRelease
	var current *WinOfficeSecurityRelease

	// Patterns
	datePattern := regexp.MustCompile(`^## ([A-Z][a-z]+ \d{1,2}, \d{4})`)
	// Matches: "Current Channel: Version 2602 (Build 19725.20172)"
	// Also matches: "Monthly Enterprise Channel: Version 2512 (Build 19530.20260)"
	// Also matches: "Office LTSC 2024 Volume Licensed: Version 2408 (Build 17932.20700)"
	// Note: All versions appear on a single line, so we use FindAllStringSubmatch
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
			current = &WinOfficeSecurityRelease{Date: matches[1]}
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
			// (Office 2024 Retail and Office 2021 Retail have same builds as Current Channel)
			if strings.Contains(channelOrProduct, "Retail") {
				continue
			}

			// Check if we already have this version branch (avoid duplicates)
			found := false
			for _, b := range current.Branches {
				if b.Version == version {
					found = true
					break
				}
			}
			if !found {
				current.Branches = append(current.Branches, WinOfficeVersionBranch{
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

// BuildWinOfficeBulletin creates an WinOfficeBulletin from scraped releases
func BuildWinOfficeBulletin(releases []WinOfficeSecurityRelease) *WinOfficeBulletin {
	bulletin := &WinOfficeBulletin{
		BuildPrefixToVersion: make(map[string]string),
		CVEToFixedBuilds:     make(map[string]map[string]string),
	}

	// Process all releases to build mappings
	for _, rel := range releases {
		// Build prefix to version mapping
		for _, branch := range rel.Branches {
			bulletin.BuildPrefixToVersion[branch.BuildPrefix] = branch.Version
		}

		// CVE to fixed builds mapping
		for _, cve := range rel.CVEs {
			if bulletin.CVEToFixedBuilds[cve] == nil {
				bulletin.CVEToFixedBuilds[cve] = make(map[string]string)
			}
			for _, branch := range rel.Branches {
				// Only store if not already set (first fix wins)
				if _, exists := bulletin.CVEToFixedBuilds[cve][branch.Version]; !exists {
					bulletin.CVEToFixedBuilds[cve][branch.Version] = branch.FullBuild
				}
			}
		}
	}

	// Get supported versions from most recent release
	if len(releases) > 0 {
		for _, branch := range releases[0].Branches {
			bulletin.SupportedVersions = append(bulletin.SupportedVersions, branch.Version)
		}
	}

	return bulletin
}

// ConvertWinOfficeToAppBulletin converts an WinOfficeBulletin to AppBulletinFile format
func ConvertWinOfficeToAppBulletin(bulletin *WinOfficeBulletin) *msrcapps.AppBulletinFile {
	// Group CVEs by version branch for the products list
	// We'll create one "product" per version branch
	products := make(map[string]*msrcapps.AppBulletin)

	for cve, fixedBuilds := range bulletin.CVEToFixedBuilds {
		for version, fixedBuild := range fixedBuilds {
			if products[version] == nil {
				products[version] = &msrcapps.AppBulletin{
					ProductID: version, // Use version as product ID for now
					Product:   fmt.Sprintf("Microsoft 365 Apps (Version %s)", version),
				}
			}
			products[version].SecurityUpdates = append(products[version].SecurityUpdates, msrcapps.SecurityUpdate{
				CVE:          cve,
				FixedVersion: "16.0." + fixedBuild,
			})
		}
	}

	// Convert to slice and sort
	var productList []msrcapps.AppBulletin
	for _, p := range products {
		// Sort security updates by CVE for deterministic output
		sort.Slice(p.SecurityUpdates, func(i, j int) bool {
			return p.SecurityUpdates[i].CVE < p.SecurityUpdates[j].CVE
		})
		productList = append(productList, *p)
	}
	sort.Slice(productList, func(i, j int) bool {
		return productList[i].ProductID < productList[j].ProductID
	})

	return &msrcapps.AppBulletinFile{
		Products: productList,
	}
}

// MatchHostVersion determines if a host is vulnerable to a CVE
// Returns: (isVulnerable, fixedVersion, error)
// fixedVersion is empty if not vulnerable or if version is unsupported
func (b *WinOfficeBulletin) MatchHostVersion(hostVersion string, cve string) (bool, string, error) {
	// Parse host version: "16.0.19628.20204" -> buildPrefix="19628", buildSuffix="20204"
	parts := strings.Split(hostVersion, ".")
	if len(parts) < 4 {
		return false, "", fmt.Errorf("invalid version format: %s", hostVersion)
	}

	buildPrefix := parts[2]
	buildSuffix := parts[3]
	hostBuild := buildPrefix + "." + buildSuffix

	// Look up version branch from build prefix
	versionBranch, ok := b.BuildPrefixToVersion[buildPrefix]
	if !ok {
		// Unknown build prefix - might be very old or very new
		return false, "", fmt.Errorf("unknown build prefix: %s", buildPrefix)
	}

	// Check if this CVE has a fix for this version branch
	fixedBuilds, ok := b.CVEToFixedBuilds[cve]
	if !ok {
		// CVE not found - not vulnerable (or CVE doesn't affect Office)
		return false, "", nil
	}

	fixedBuild, ok := fixedBuilds[versionBranch]
	if !ok {
		// No fix for this version branch
		// Check if the branch is still supported
		supported := false
		for _, v := range b.SupportedVersions {
			if v == versionBranch {
				supported = true
				break
			}
		}
		if !supported {
			// Unsupported version - vulnerable, must upgrade
			return true, "", nil
		}
		// Supported but no fix listed - CVE might not affect this version
		return false, "", nil
	}

	// Compare builds
	vulnerable := compareBuildVersions(hostBuild, fixedBuild) < 0
	if vulnerable {
		return true, "16.0." + fixedBuild, nil
	}
	return false, "", nil
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
		maxLen := len(suffixA)
		if len(suffixB) > maxLen {
			maxLen = len(suffixB)
		}
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

// FetchWinOfficeBulletin scrapes and builds Office security bulletin
func FetchWinOfficeBulletin(client *http.Client) (*WinOfficeBulletin, error) {
	releases, err := ScrapeWinOfficeSecurityUpdates(client)
	if err != nil {
		return nil, err
	}

	if len(releases) == 0 {
		return nil, errors.New("no releases found")
	}

	return BuildWinOfficeBulletin(releases), nil
}
