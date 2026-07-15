package android

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/utils"
)

const vulnBatchSize = 500

// OSVulnStore is the subset of fleet.Datastore needed by the Android analyzer.
type OSVulnStore interface {
	ListOSVulnerabilitiesByOS(ctx context.Context, osID uint) ([]fleet.OSVulnerability, error)
	InsertOSVulnerabilities(ctx context.Context, vulns []fleet.OSVulnerability, source fleet.VulnerabilitySource) (int64, error)
	DeleteOSVulnerabilities(ctx context.Context, vulns []fleet.OSVulnerability) error
}

type ArtifactCache struct {
	version  string
	artifact *AndroidArtifact
}

func NewArtifactCache() *ArtifactCache {
	return &ArtifactCache{}
}

func (c *ArtifactCache) get(majorVersion, vulnPath string) (*AndroidArtifact, error) {
	if c.version == majorVersion && c.artifact != nil {
		return c.artifact, nil
	}
	a, err := loadArtifact(majorVersion, vulnPath)
	if err != nil {
		return nil, err
	}
	c.version = majorVersion
	c.artifact = a
	return a, nil
}

// AndroidVuln mirrors the artifact entry produced by cmd/osv-processor.
type AndroidVuln struct {
	CVE      string `json:"cve"`
	FixedSPL string `json:"fixed_spl"`
	Severity string `json:"severity,omitempty"`
}

// AndroidArtifact is the gzipped JSON artifact produced by osv-processor for a
// single Android major version.
type AndroidArtifact struct {
	SchemaVersion   string        `json:"schema_version"`
	AndroidVersion  string        `json:"android_version"`
	Generated       string        `json:"generated"`
	TotalCVEs       int           `json:"total_cves"`
	Vulnerabilities []AndroidVuln `json:"vulnerabilities"`
}

// Analyze matches a single Android OperatingSystem row against the downloaded
// Android OSV artifact and writes the results to operating_system_vulnerabilities.
//
// The OperatingSystem.Version is formatted as "16 (2026-05-01)" by PR #49272.
// We extract the major version to load the right artifact, and the SPL date to
// determine which CVEs affect the host: if hostSPL < vuln.FixedSPL, the host
// is vulnerable.
func Analyze(
	ctx context.Context,
	ds OSVulnStore,
	os fleet.OperatingSystem,
	vulnPath string,
	collectVulns bool,
	logger *slog.Logger,
	cache *ArtifactCache,
) ([]fleet.OSVulnerability, error) {
	if logger == nil {
		logger = slog.Default()
	}

	majorVersion, hostSPL := parseAndroidVersion(os.Version)
	if majorVersion == "" {
		return nil, nil
	}

	artifact, err := cache.get(majorVersion, vulnPath)
	if err != nil {
		logger.DebugContext(ctx, "no Android OSV artifact found",
			"android_version", majorVersion,
			"err", err)
		return nil, nil
	}

	if len(artifact.Vulnerabilities) == 0 {
		return nil, nil
	}

	// Match: host is vulnerable if its SPL is before the fix's SPL.
	// If the host has no SPL (bare version like "16"), we can't determine
	// vulnerability status, so we skip matching.
	var found []fleet.OSVulnerability
	if hostSPL != "" {
		for _, vuln := range artifact.Vulnerabilities {
			if vuln.FixedSPL == "" {
				continue
			}
			if hostSPL < vuln.FixedSPL {
				found = append(found, fleet.OSVulnerability{
					OSID:              os.ID,
					CVE:               vuln.CVE,
					Source:            fleet.AndroidOSVSource,
					ResolvedInVersion: resolvedVersion(majorVersion, vuln.FixedSPL),
				})
			}
		}
	}

	// Fetch existing vulns and compute delta.
	existing, err := ds.ListOSVulnerabilitiesByOS(ctx, os.ID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "listing existing Android OS vulnerabilities")
	}

	// Filter existing to only our source so we don't interfere with other analyzers.
	var existingAndroid []fleet.OSVulnerability
	for _, v := range existing {
		if v.Source == fleet.AndroidOSVSource {
			existingAndroid = append(existingAndroid, v)
		}
	}

	toInsert, toDelete := utils.VulnsDelta(found, existingAndroid)

	toInsertMap := make(map[string]fleet.OSVulnerability, len(toInsert))
	for _, v := range toInsert {
		toInsertMap[v.Key()] = v
	}
	toDeleteMap := make(map[string]fleet.OSVulnerability, len(toDelete))
	for _, v := range toDelete {
		toDeleteMap[v.Key()] = v
	}

	if err := utils.BatchProcess(toDeleteMap, func(v []fleet.OSVulnerability) error {
		return ds.DeleteOSVulnerabilities(ctx, v)
	}, vulnBatchSize); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "deleting stale Android OS vulnerabilities")
	}

	var inserted []fleet.OSVulnerability
	if collectVulns {
		inserted = make([]fleet.OSVulnerability, 0, len(toInsertMap))
	}

	if err := utils.BatchProcess(toInsertMap, func(v []fleet.OSVulnerability) error {
		n, err := ds.InsertOSVulnerabilities(ctx, v, fleet.AndroidOSVSource)
		if err != nil {
			return err
		}
		if collectVulns && n > 0 {
			inserted = append(inserted, v...)
		}
		return nil
	}, vulnBatchSize); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "inserting Android OS vulnerabilities")
	}

	return inserted, nil
}

// parseAndroidVersion parses the operating_systems.version field for Android.
//
//	"16 (2026-05-01)" -> ("16", "2026-05-01")
//	"16"              -> ("16", "")
//	""                -> ("", "")
func parseAndroidVersion(version string) (majorVersion, spl string) {
	if version == "" {
		return "", ""
	}

	// Look for " (YYYY-MM-DD)" pattern
	if idx := strings.Index(version, " ("); idx > 0 {
		major := version[:idx]
		rest := version[idx+2:]
		if end := strings.Index(rest, ")"); end > 0 {
			return major, rest[:end]
		}
	}

	return version, ""
}

// resolvedVersion formats the resolved-in version for display, e.g.
// "16 (2026-06-01)" — matching the operating_systems.version format.
func resolvedVersion(majorVersion, fixedSPL string) *string {
	s := fmt.Sprintf("%s (%s)", majorVersion, fixedSPL)
	return &s
}

// loadArtifact finds and loads the most recent Android OSV artifact for the
// given major version from the vulnerability directory.
func loadArtifact(majorVersion, vulnPath string) (*AndroidArtifact, error) {
	prefix := fmt.Sprintf("osv-android-%s-", majorVersion)
	pattern := filepath.Join(vulnPath, prefix+"*.json.gz")

	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("globbing Android OSV artifacts: %w", err)
	}
	if len(matches) == 0 {
		return nil, fmt.Errorf("no Android OSV artifact found for version %s", majorVersion)
	}

	// Pick the latest by filename (date is in the name, lexicographic sort works).
	latest := matches[0]
	for _, m := range matches[1:] {
		if m > latest {
			latest = m
		}
	}

	return readArtifact(latest)
}

func readArtifact(path string) (*AndroidArtifact, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return nil, fmt.Errorf("opening gzip reader: %w", err)
	}
	defer gz.Close()

	var artifact AndroidArtifact
	if err := json.NewDecoder(gz).Decode(&artifact); err != nil {
		return nil, fmt.Errorf("decoding Android OSV artifact: %w", err)
	}

	return &artifact, nil
}
