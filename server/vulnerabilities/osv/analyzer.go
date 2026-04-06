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
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/utils"
)

const (
	hostsBatchSize = 500
	vulnBatchSize  = 500
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
func Analyze(
	ctx context.Context,
	ds fleet.Datastore,
	ver fleet.OSVersion,
	vulnPath string,
	collectVulns bool,
	logger *slog.Logger,
	date time.Time,
) ([]fleet.SoftwareVulnerability, error) {
	if !IsPlatformSupported(ver.Platform) {
		return nil, ErrUnsupportedPlatform
	}

	artifact, err := loadOSVArtifact(ctx, ver, vulnPath, logger, date)
	if err != nil {
		return nil, fmt.Errorf("loading OSV artifact: %w", err)
	}

	source := fleet.UbuntuOSVSource
	toInsertSet := make(map[string]fleet.SoftwareVulnerability)
	toDeleteSet := make(map[string]fleet.SoftwareVulnerability)
	totalHosts := 0

	// Paginate through all hosts with this OS version
	var offset int
	for {
		hostIDs, err := ds.HostIDsByOSVersion(ctx, ver, offset, hostsBatchSize)
		if err != nil {
			return nil, fmt.Errorf("getting host IDs: %w", err)
		}

		if len(hostIDs) == 0 {
			break
		}

		totalHosts += len(hostIDs)
		offset += hostsBatchSize

		foundInBatch := make(map[uint][]fleet.SoftwareVulnerability)
		for _, hostID := range hostIDs {
			software, err := ds.ListSoftwareForVulnDetection(ctx, fleet.VulnSoftwareFilter{
				HostID: &hostID,
			})
			if err != nil {
				return nil, fmt.Errorf("listing software for host %d: %w", hostID, err)
			}

			foundInBatch[hostID] = matchSoftwareToOSV(software, artifact)
		}

		existingInBatch, err := ds.ListSoftwareVulnerabilitiesByHostIDsSource(ctx, hostIDs, source)
		if err != nil {
			return nil, fmt.Errorf("listing existing vulnerabilities: %w", err)
		}

		for _, hostID := range hostIDs {
			insrt, del := utils.VulnsDelta(foundInBatch[hostID], existingInBatch[hostID])
			for _, i := range insrt {
				toInsertSet[i.Key()] = i
			}
			for _, d := range del {
				toDeleteSet[d.Key()] = d
			}
		}
	}

	if totalHosts == 0 {
		logger.DebugContext(ctx, "no hosts found for os version", "platform", ver.Platform, "version", ver.Version)
		return nil, nil
	}

	logger.DebugContext(ctx, "processed hosts for osv analysis", "platform", ver.Platform, "version", ver.Version, "host_count", totalHosts)

	// Delete stale vulnerabilities
	err = utils.BatchProcess(toDeleteSet, func(v []fleet.SoftwareVulnerability) error {
		return ds.DeleteSoftwareVulnerabilities(ctx, v)
	}, vulnBatchSize)
	if err != nil {
		return nil, fmt.Errorf("deleting stale vulnerabilities: %w", err)
	}

	// Insert new vulnerabilities
	allVulns := make([]fleet.SoftwareVulnerability, 0, len(toInsertSet))
	for _, v := range toInsertSet {
		allVulns = append(allVulns, v)
	}

	newVulns, err := ds.InsertSoftwareVulnerabilities(ctx, allVulns, source)
	if err != nil {
		return nil, fmt.Errorf("inserting software vulnerabilities: %w", err)
	}

	if !collectVulns {
		return nil, nil
	}

	return newVulns, nil
}

// findLatestOSVArtifactForVersion finds the most recent OSV artifact for a specific Ubuntu version
func findLatestOSVArtifactForVersion(vulnPath string, ubuntuVersion string) (string, error) {
	files, err := os.ReadDir(vulnPath)
	if err != nil {
		return "", fmt.Errorf("reading vulnerability path: %w", err)
	}

	// Pattern: osv-ubuntu-2204-YYYY-MM-DD.json.gz
	prefix := fmt.Sprintf("osv-ubuntu-%s-", ubuntuVersion)
	suffix := ".json.gz"

	var latestFile os.DirEntry
	var latestModTime time.Time

	for _, f := range files {
		if f.IsDir() {
			continue
		}

		name := f.Name()
		// Skip delta files, same as downloader.go
		if strings.HasPrefix(name, prefix) && strings.HasSuffix(name, suffix) && !strings.Contains(name, "delta") {
			info, err := f.Info()
			if err != nil {
				continue
			}

			if latestFile == nil || info.ModTime().After(latestModTime) {
				latestFile = f
				latestModTime = info.ModTime()
			}
		}
	}

	if latestFile == nil {
		return "", fmt.Errorf("no OSV artifact found for Ubuntu %s", ubuntuVersion)
	}

	return filepath.Join(vulnPath, latestFile.Name()), nil
}

// loadOSVArtifact loads the full OSV artifact for the given Ubuntu version
func loadOSVArtifact(ctx context.Context, ver fleet.OSVersion, vulnPath string, logger *slog.Logger, date time.Time) (*OSVArtifact, error) {
	// Extract Ubuntu version (e.g., "22.04.8 LTS" -> "2204")
	ubuntuVer := extractUbuntuVersion(ver.Version)
	if ubuntuVer == "" {
		return nil, fmt.Errorf("could not extract Ubuntu version from %s", ver.Version)
	}

	// Try to find date-specific artifact first, fall back to latest if not found
	fileName := osvFilename(ubuntuVer, date)
	artifactFile := filepath.Join(vulnPath, fileName)

	if _, err := os.Stat(artifactFile); err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("checking OSV artifact %s: %w", artifactFile, err)
		}

		artifactFile, err = findLatestOSVArtifactForVersion(vulnPath, ubuntuVer)
		if err != nil {
			return nil, fmt.Errorf("finding OSV artifact for Ubuntu %s: %w", ubuntuVer, err)
		}
	}

	f, err := os.Open(artifactFile)
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
		"file", filepath.Base(artifactFile),
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
				var resolvedIn *string
				if vuln.Fixed != "" {
					resolvedIn = &vuln.Fixed
				}
				result = append(result, fleet.SoftwareVulnerability{
					SoftwareID:        sw.ID,
					CVE:               vuln.CVE,
					ResolvedInVersion: resolvedIn,
				})
			}
		}
	}

	return result
}

// extractUbuntuVersion extracts the numeric version from Ubuntu version strings
// Examples:
//
//	"22.04.8 LTS" -> "2204"
//	"20.04.1 LTS" -> "2004"
//	"23.10" -> "2310"
//	"22.04.1 LTS (Jammy Jellyfish)" -> "2204"
func extractUbuntuVersion(version string) string {
	version = strings.TrimSpace(version)

	// Remove " LTS" and anything after it (like codename)
	if idx := strings.Index(version, " LTS"); idx != -1 {
		version = version[:idx]
	}

	// Remove parentheses
	if idx := strings.Index(version, " ("); idx != -1 {
		version = version[:idx]
	}

	version = strings.TrimSpace(version)

	// Get major.minor.patch
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
