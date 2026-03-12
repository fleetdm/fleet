package osv

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	feednvd "github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/cvefeed/nvd"
)

var ErrUnsupportedPlatform = errors.New("unsupported platform")

// OSVArtifact represents the processed OSV data for a specific Ubuntu version
type OSVArtifact struct {
	SchemaVersion   string                        `json:"schema_version"`
	UbuntuVersion   string                        `json:"ubuntu_version"`
	Generated       time.Time                     `json:"generated"`
	TotalCVEs       int                           `json:"total_cves"`
	TotalPackages   int                           `json:"total_packages"`
	Vulnerabilities map[string][]OSVVulnerability `json:"vulnerabilities"`
}

// OSVVulnerability represents a single vulnerability for a package
type OSVVulnerability struct {
	CVE        string    `json:"cve"`
	Published  time.Time `json:"published"`
	Modified   time.Time `json:"modified"`
	Details    string    `json:"details"`
	Introduced string    `json:"introduced"`
	Fixed      string    `json:"fixed,omitempty"`
	Versions   []string  `json:"versions,omitempty"`
}

// Analyze scans all hosts for vulnerabilities based on the OSV artifacts for their platform.
// Returns new vulnerabilities found.
func Analyze(
	ctx context.Context,
	ds fleet.Datastore,
	ver fleet.OSVersion,
	vulnPath string,
	collectVulns bool,
) ([]fleet.SoftwareVulnerability, error) {
	if ver.Platform != "ubuntu" {
		return nil, ErrUnsupportedPlatform
	}

	artifact, err := loadOSVArtifact(ver, vulnPath)
	if err != nil {
		return nil, fmt.Errorf("loading OSV artifact: %w", err)
	}

	fmt.Printf("[OSV DEBUG] Loaded artifact for Ubuntu %s: %d packages, %d total CVEs\n",
		artifact.UbuntuVersion, len(artifact.Vulnerabilities), artifact.TotalCVEs)

	// Get all hosts with this OS version
	hostIDs, err := ds.HostIDsByOSVersion(ctx, ver, 0, 10000)
	if err != nil {
		return nil, fmt.Errorf("getting host IDs: %w", err)
	}

	if len(hostIDs) == 0 {
		return nil, nil
	}

	allVulns := make(map[string]fleet.SoftwareVulnerability)

	for _, hostID := range hostIDs {
		software, err := ds.ListSoftwareForVulnDetection(ctx, fleet.VulnSoftwareFilter{
			HostID: &hostID,
		})
		if err != nil {
			return nil, fmt.Errorf("listing software for host %d: %w", hostID, err)
		}

		fmt.Printf("[OSV DEBUG] Host %d has %d software packages\n", hostID, len(software))

		vulns := matchSoftwareToOSV(software, artifact)

		fmt.Printf("[OSV DEBUG] Host %d matched %d vulnerabilities\n", hostID, len(vulns))

		for _, v := range vulns {
			key := v.Key()
			allVulns[key] = v
		}
	}

	vulnsList := make([]fleet.SoftwareVulnerability, 0, len(allVulns))
	for _, v := range allVulns {
		vulnsList = append(vulnsList, v)
	}

	source := fleet.VulnerabilitySource(7)
	newVulns, err := ds.InsertSoftwareVulnerabilities(ctx, vulnsList, source)
	if err != nil {
		return nil, fmt.Errorf("inserting software vulnerabilities: %w", err)
	}

	if !collectVulns {
		return nil, nil
	}

	return newVulns, nil
}

// loadOSVArtifact loads the OSV artifact for the given Ubuntu version
func loadOSVArtifact(ver fleet.OSVersion, _ string) (*OSVArtifact, error) {
	// Extract Ubuntu version (e.g., "22.04.8 LTS" -> "2204")
	ubuntuVer := extractUbuntuVersion(ver.Version)
	if ubuntuVer == "" {
		return nil, fmt.Errorf("could not extract Ubuntu version from %s", ver.Version)
	}

	// Hardcoded path for POC
	artifactsPath := "/Users/ksykulev/projects/fleet-main/cmd/osv-processor/test-artifacts-final"

	// Find the latest OSV artifact file for this version
	// Pattern: osv-ubuntu-2204-YYYY-MM-DD.json.gz
	pattern := fmt.Sprintf("osv-ubuntu-%s-*.json.gz", ubuntuVer)
	matches, err := filepath.Glob(filepath.Join(artifactsPath, pattern))
	if err != nil {
		return nil, fmt.Errorf("globbing for OSV artifacts: %w", err)
	}

	var fullArtifacts []string
	for _, match := range matches {
		if !strings.Contains(filepath.Base(match), "-delta-") {
			fullArtifacts = append(fullArtifacts, match)
		}
	}

	if len(fullArtifacts) == 0 {
		return nil, fmt.Errorf("no OSV artifact found for Ubuntu %s in %s", ubuntuVer, artifactsPath)
	}

	latestFile := fullArtifacts[len(fullArtifacts)-1]

	fmt.Printf("[OSV DEBUG] Loading artifact file: %s\n", filepath.Base(latestFile))

	f, err := os.Open(latestFile)
	if err != nil {
		return nil, fmt.Errorf("opening OSV artifact: %w", err)
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return nil, fmt.Errorf("creating gzip reader: %w", err)
	}
	defer gz.Close()

	var artifact OSVArtifact
	if err := json.NewDecoder(gz).Decode(&artifact); err != nil {
		return nil, fmt.Errorf("decoding OSV artifact: %w", err)
	}

	return &artifact, nil
}

// extractUbuntuVersion extracts the numeric version from Ubuntu version strings
// Examples: "22.04.8 LTS" -> "2204", "20.04.1 LTS" -> "2004"
func extractUbuntuVersion(version string) string {
	// Remove " LTS" suffix if present
	version = strings.TrimSuffix(version, " LTS")
	version = strings.TrimSpace(version)

	parts := strings.Split(version, ".")
	if len(parts) < 2 {
		return ""
	}

	return parts[0] + parts[1]
}

// matchSoftwareToOSV matches software against OSV vulnerabilities
func matchSoftwareToOSV(software []fleet.Software, artifact *OSVArtifact) []fleet.SoftwareVulnerability {
	var result []fleet.SoftwareVulnerability

	matchedPackages := 0
	checkedPackages := 0
	linuxPackagesChecked := 0

	for _, sw := range software {
		checkedPackages++

		if strings.HasPrefix(sw.Name, "linux") {
			linuxPackagesChecked++
			if linuxPackagesChecked <= 5 {
				fmt.Printf("[OSV DEBUG] Linux package found: %s (version: %s)\n", sw.Name, sw.Version)
			}
		}

		packageName := sw.Name

		// Kernel package mapping: Only linux-image-* packages (actual kernel binaries)
		// should be checked against kernel CVEs, not headers/modules/tools
		if strings.HasPrefix(sw.Name, "linux-image-") || strings.HasPrefix(sw.Name, "linux-signed-image-") {
			packageName = "linux"
		}

		vulns, ok := artifact.Vulnerabilities[packageName]
		if !ok {
			if checkedPackages <= 20 {
				fmt.Printf("[OSV DEBUG] No match for package: %s (version: %s, source: %s)\n",
					sw.Name, sw.Version, sw.Source)
			}
			continue
		}

		matchedPackages++

		isLinuxPackage := strings.HasPrefix(sw.Name, "linux")
		if isLinuxPackage {
			fmt.Printf("[OSV DEBUG] MATCHED LINUX package: %s (version: %s) mapped to '%s' has %d potential CVEs\n",
				sw.Name, sw.Version, packageName, len(vulns))
		}

		if matchedPackages <= 10 && !isLinuxPackage {
			fmt.Printf("[OSV DEBUG] MATCHED package: %s (version: %s) has %d potential CVEs\n",
				sw.Name, sw.Version, len(vulns))
		}

		vulnerableCount := 0

		// For kernel packages, we need to normalize the version
		// osquery: 5.15.0-94-generic -> 5.15.0-94
		// OSV: 5.15.0-94.104
		isKernelPackage := packageName == "linux"

		for _, vuln := range vulns {
			var vulnerable bool
			if matchedPackages == 1 {
				vulnerable = isVulnerableDebug(sw.Version, vuln, isKernelPackage)
			} else {
				vulnerable = isVulnerable(sw.Version, vuln, isKernelPackage)
			}

			if vulnerable {
				result = append(result, fleet.SoftwareVulnerability{
					SoftwareID: sw.ID,
					CVE:        vuln.CVE,
				})
				vulnerableCount++
			}
		}

		if matchedPackages <= 10 {
			fmt.Printf("[OSV DEBUG]   -> After version check: %d actually vulnerable\n", vulnerableCount)
		}
	}

	fmt.Printf("[OSV DEBUG] Checked %d packages, matched %d packages, found %d vulnerabilities (linux packages checked: %d)\n",
		checkedPackages, matchedPackages, len(result), linuxPackagesChecked)

	return result
}

// normalizeKernelVersion extracts the base kernel version for matching
// Example: "5.15.0-94-generic" -> "5.15.0-94"
func normalizeKernelVersion(version string) string {
	parts := strings.Split(version, "-")
	if len(parts) >= 2 {
		return parts[0] + "-" + parts[1]
	}
	return version
}

// kernelVersionMatches checks if a kernel version matches an OSV version
// Handles both exact matches and prefix matches
// osquery: "5.15.0-94-generic" normalizes to "5.15.0-94"
// OSV: "5.15.0-94.104" starts with "5.15.0-94"
func kernelVersionMatches(softwareVersion, osvVersion string) bool {
	normalized := normalizeKernelVersion(softwareVersion)

	if normalized == osvVersion {
		return true
	}

	if strings.HasPrefix(osvVersion, normalized+".") {
		return true
	}

	return false
}

// isVulnerable checks if a software version is vulnerable based on OSV data
func isVulnerable(softwareVersion string, vuln OSVVulnerability, isKernelPackage bool) bool {
	if len(vuln.Versions) > 0 {
		for _, v := range vuln.Versions {
			if isKernelPackage {
				if kernelVersionMatches(softwareVersion, v) {
					return true
				}
			} else {
				if softwareVersion == v {
					return true
				}
			}
		}
		return false
	}

	// No explicit versions list - use range-based matching
	introduced := vuln.Introduced
	if introduced == "" {
		introduced = "0"
	}

	if introduced != "0" {
		cmp := feednvd.SmartVerCmp(softwareVersion, introduced)
		if cmp == -1 { // softwareVersion < introduced
			return false
		}
	}

	if vuln.Fixed != "" {
		cmp := feednvd.SmartVerCmp(softwareVersion, vuln.Fixed)
		if cmp != -1 { // softwareVersion >= fixed (not vulnerable)
			return false
		}
	}

	return true
}

var debugVersionCheckCount = 0

// isVulnerableDebug is a debug wrapper around isVulnerable
func isVulnerableDebug(softwareVersion string, vuln OSVVulnerability, isKernelPackage bool) bool {
	result := isVulnerable(softwareVersion, vuln, isKernelPackage)

	debugVersionCheckCount++
	if debugVersionCheckCount <= 5 {
		hasVersions := len(vuln.Versions) > 0
		normalized := ""
		if isKernelPackage {
			normalized = normalizeKernelVersion(softwareVersion)
		}
		fmt.Printf("[OSV DEBUG]     Version check: %s (normalized: %s, is_kernel: %v) vs CVE %s (introduced: %s, fixed: %s, has_versions_list: %v, versions_count: %d) -> %v\n",
			softwareVersion, normalized, isKernelPackage, vuln.CVE, vuln.Introduced, vuln.Fixed, hasVersions, len(vuln.Versions), result)
	}

	return result
}
