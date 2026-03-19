package osv

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	feednvd "github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/cvefeed/nvd"
)

var ErrUnsupportedPlatform = errors.New("unsupported platform")

// IsPlatformSupported returns true if the given platform is supported by OSV
func IsPlatformSupported(platform string) bool {
	return strings.ToLower(platform) == "ubuntu"
}

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

// Analyze scans all hosts for vulnerabilities based on the OSV artifacts for their platform
// Returns new vulnerabilities found
func Analyze(
	ctx context.Context,
	ds fleet.Datastore,
	ver fleet.OSVersion,
	vulnPath string,
	collectVulns bool,
	logger *slog.Logger,
) ([]fleet.SoftwareVulnerability, error) {
	if !IsPlatformSupported(ver.Platform) {
		return nil, ErrUnsupportedPlatform
	}

	artifact, err := loadOSVArtifact(ctx, ver, vulnPath, logger)
	if err != nil {
		return nil, fmt.Errorf("loading OSV artifact: %w", err)
	}

	// Get all hosts with this OS version
	hostIDs, err := ds.HostIDsByOSVersion(ctx, ver, 0, 10000)
	if err != nil {
		return nil, fmt.Errorf("getting host IDs: %w", err)
	}

	if len(hostIDs) == 0 {
		logger.DebugContext(ctx, "no hosts found for os version", "platform", ver.Platform, "version", ver.Version)
		return nil, nil
	}

	logger.DebugContext(ctx, "processing hosts for osv analysis", "platform", ver.Platform, "version", ver.Version, "host_count", len(hostIDs))

	allVulns := make(map[string]fleet.SoftwareVulnerability)

	for _, hostID := range hostIDs {
		software, err := ds.ListSoftwareForVulnDetection(ctx, fleet.VulnSoftwareFilter{
			HostID: &hostID,
		})
		if err != nil {
			return nil, fmt.Errorf("listing software for host %d: %w", hostID, err)
		}

		vulns := matchSoftwareToOSV(software, artifact)

		for _, v := range vulns {
			key := v.Key()
			allVulns[key] = v
		}
	}

	vulnsList := make([]fleet.SoftwareVulnerability, 0, len(allVulns))
	for _, v := range allVulns {
		vulnsList = append(vulnsList, v)
	}

	source := fleet.UbuntuOSVSource
	newVulns, err := ds.InsertSoftwareVulnerabilities(ctx, vulnsList, source)
	if err != nil {
		return nil, fmt.Errorf("inserting software vulnerabilities: %w", err)
	}

	if !collectVulns {
		return nil, nil
	}

	return newVulns, nil
}

// loadOSVArtifact loads the full OSV artifact for the given Ubuntu version
func loadOSVArtifact(ctx context.Context, ver fleet.OSVersion, _ /*artifactPath*/ string, logger *slog.Logger) (*OSVArtifact, error) {
	// Extract Ubuntu version (e.g., "22.04.8 LTS" -> "2204")
	ubuntuVer := extractUbuntuVersion(ver.Version)
	if ubuntuVer == "" {
		return nil, fmt.Errorf("could not extract Ubuntu version from %s", ver.Version)
	}

	// TODO:: #41571
	artifactsPath := "/Users/ksykulev/projects/fleet-main/cmd/osv-processor/test-artifacts-final"

	// TODO:: Figure out how to see last run time
	// If one exists, use the delta file else use the full artifact file.

	// Find the latest OSV artifact file for this version
	// Pattern: osv-ubuntu-2204-YYYY-MM-DD.json.gz
	pattern := fmt.Sprintf("osv-ubuntu-%s-*.json.gz", ubuntuVer)
	matches, err := filepath.Glob(filepath.Join(artifactsPath, pattern))
	if err != nil {
		return nil, fmt.Errorf("globbing for OSV artifacts: %w", err)
	}

	var fullArtifacts []string
	// Pattern for deltas: osv-ubuntu-2204-delta-YYYY-MM-DD.json.gz
	for _, match := range matches {
		if !strings.Contains(filepath.Base(match), "-delta-") {
			fullArtifacts = append(fullArtifacts, match)
		}
	}

	if len(fullArtifacts) == 0 {
		return nil, fmt.Errorf("no OSV artifact found for Ubuntu %s in %s", ubuntuVer, artifactsPath)
	}

	latestFile := fullArtifacts[len(fullArtifacts)-1]

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

	logger.DebugContext(ctx, "loaded osv artifact",
		"file", filepath.Base(latestFile),
		"ubuntu_version", artifact.UbuntuVersion,
		"total_packages", artifact.TotalPackages,
		"total_cves", artifact.TotalCVEs)

	return &artifact, nil
}

// matchSoftwareToOSV matches software against OSV vulnerabilities
func matchSoftwareToOSV(software []fleet.Software, artifact *OSVArtifact) []fleet.SoftwareVulnerability {
	var result []fleet.SoftwareVulnerability

	for _, sw := range software {
		packageName := sw.Name

		// Kernel package mapping: Only linux-image-* packages (actual kernel binaries)
		// should be checked against kernel CVEs, not headers/modules/tools
		if strings.HasPrefix(sw.Name, "linux-image-") || strings.HasPrefix(sw.Name, "linux-signed-image-") {
			packageName = "linux"
		}

		vulns, ok := artifact.Vulnerabilities[packageName]
		if !ok {
			continue
		}

		// For kernel packages, we need to normalize the version
		// osquery: 5.15.0-94-generic -> 5.15.0-94
		// OSV: 5.15.0-94.104
		isKernelPackage := packageName == "linux"

		for _, vuln := range vulns {
			if isVulnerable(sw.Version, vuln, isKernelPackage) {
				result = append(result, fleet.SoftwareVulnerability{
					SoftwareID: sw.ID,
					CVE:        vuln.CVE,
				})
			}
		}
	}

	return result
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
