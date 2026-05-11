package winoffice

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/io"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/utils"
)

// getLatestBulletin returns the most recent Windows Office bulletin (based on the date in the
// filename) contained in 'vulnPath'.
func getLatestBulletin(vulnPath string) (*BulletinFile, error) {
	fs := io.NewFSClient(vulnPath)

	files, err := fs.WinOfficeBulletin()
	if err != nil {
		return nil, err
	}

	if len(files) == 0 {
		return nil, nil
	}

	sort.Slice(files, func(i, j int) bool { return files[j].Before(files[i]) })
	filePath := filepath.Join(vulnPath, files[0].String())

	payload, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var bulletin BulletinFile
	if err := json.Unmarshal(payload, &bulletin); err != nil {
		return nil, err
	}

	return &bulletin, nil
}

// parseOfficeVersion parses a Windows Office version string like "16.0.19725.20204"
// and returns the build prefix and build suffix.
func parseOfficeVersion(version string) (buildPrefix, buildSuffix string, err error) {
	if !strings.HasPrefix(version, OfficeVersionPrefix) {
		return "", "", fmt.Errorf("invalid Office version prefix: %s", version)
	}
	parts := strings.Split(version, ".")
	if len(parts) < 4 {
		return "", "", fmt.Errorf("invalid Office version format: %s", version)
	}
	return parts[2], parts[3], nil
}

// compareBuildSuffix compares two build suffixes numerically.
// Returns -1 if a < b, 0 if a == b, 1 if a > b.
func compareBuildSuffix(a, b string) int {
	// Pad to same length for proper string comparison
	maxLen := max(len(a), len(b))
	paddedA := fmt.Sprintf("%0*s", maxLen, a)
	paddedB := fmt.Sprintf("%0*s", maxLen, b)

	if paddedA < paddedB {
		return -1
	}
	if paddedA > paddedB {
		return 1
	}
	return 0
}

// CheckVersion returns all CVEs affecting the given Office version.
// This is the main entry point for vulnerability matching.
func CheckVersion(version string, bulletin *BulletinFile) []fleet.SoftwareVulnerability {
	if bulletin == nil {
		return nil
	}
	software := &fleet.Software{Version: version}
	return collectVulnerabilities(software, bulletin)
}

// collectVulnerabilities finds all CVEs that affect the given software based on the bulletin.
func collectVulnerabilities(
	software *fleet.Software,
	bulletin *BulletinFile,
) []fleet.SoftwareVulnerability {
	buildPrefix, buildSuffix, err := parseOfficeVersion(software.Version)
	if err != nil {
		// Invalid version format, skip
		return nil
	}

	// Look up version branch from build prefix
	versionBranch, ok := bulletin.BuildPrefixes[buildPrefix]
	if !ok {
		// Unknown build prefix, might be very old or very new
		return nil
	}

	// Get security updates for this version
	versionBulletin, ok := bulletin.Versions[versionBranch]
	if !ok {
		// No data for this version
		return nil
	}

	var vulns []fleet.SoftwareVulnerability
	for _, update := range versionBulletin.SecurityUpdates {
		// Parse the fixed build
		fixedPrefix, fixedSuffix, err := parseOfficeVersion(update.ResolvedInVersion)
		if err != nil {
			continue
		}

		// Check if fix is for a different build prefix.
		// This happens when a version branch has multiple build prefixes over time
		// (e.g., LTSC 2024 with prefixes 17928 and 17932).
		// Only vulnerable if the fix's prefix is NEWER than the host's prefix.
		if fixedPrefix != buildPrefix {
			// Compare prefixes numerically - host is only vulnerable if fix prefix > host prefix
			fixedPrefixNum, err := strconv.Atoi(fixedPrefix)
			if err != nil {
				continue
			}
			buildPrefixNum, err := strconv.Atoi(buildPrefix)
			if err != nil {
				continue
			}
			if fixedPrefixNum > buildPrefixNum {
				resolvedVersion := update.ResolvedInVersion
				vulns = append(vulns, fleet.SoftwareVulnerability{
					SoftwareID:        software.ID,
					CVE:               update.CVE,
					ResolvedInVersion: &resolvedVersion,
				})
			}
			continue
		}

		// Same version branch - compare build suffixes
		// If host build suffix < fixed build suffix, the host is vulnerable
		if compareBuildSuffix(buildSuffix, fixedSuffix) < 0 {
			resolvedVersion := update.ResolvedInVersion
			vulns = append(vulns, fleet.SoftwareVulnerability{
				SoftwareID:        software.ID,
				CVE:               update.CVE,
				ResolvedInVersion: &resolvedVersion,
			})
		}
	}

	return vulns
}

// getStoredVulnerabilities returns all stored vulnerabilities for 'softwareID'.
func getStoredVulnerabilities(
	ctx context.Context,
	ds fleet.Datastore,
	softwareID uint,
) ([]fleet.SoftwareVulnerability, error) {
	storedSoftware, err := ds.SoftwareByID(ctx, softwareID, nil, false, nil)
	if err != nil {
		return nil, err
	}

	var result []fleet.SoftwareVulnerability
	for _, v := range storedSoftware.Vulnerabilities {
		result = append(result, fleet.SoftwareVulnerability{
			SoftwareID: storedSoftware.ID,
			CVE:        v.CVE,
		})
	}
	return result, nil
}

func updateVulnsInDB(
	ctx context.Context,
	ds fleet.Datastore,
	detected []fleet.SoftwareVulnerability,
	existing []fleet.SoftwareVulnerability,
) ([]fleet.SoftwareVulnerability, error) {
	toInsert, toDelete := utils.VulnsDelta(detected, existing)

	// Remove any possible dups
	toInsertSet := make(map[string]fleet.SoftwareVulnerability, len(toInsert))
	for _, i := range toInsert {
		toInsertSet[i.Key()] = i
	}

	if err := ds.DeleteSoftwareVulnerabilities(ctx, toDelete); err != nil {
		return nil, err
	}

	allVulns := make([]fleet.SoftwareVulnerability, 0, len(toInsertSet))
	for _, v := range toInsertSet {
		allVulns = append(allVulns, v)
	}

	return ds.InsertSoftwareVulnerabilities(ctx, allVulns, fleet.WinOfficeSource)
}

// Analyze uses the most recent Windows Office bulletin in 'vulnPath' for detecting
// vulnerabilities on Windows Office apps.
func Analyze(
	ctx context.Context,
	ds fleet.Datastore,
	vulnPath string,
	collectVulns bool,
) ([]fleet.SoftwareVulnerability, error) {
	bulletin, err := getLatestBulletin(vulnPath)
	if err != nil {
		return nil, err
	}

	if bulletin == nil {
		return nil, nil
	}

	// Query for Windows Office software from "programs" source.
	// Use NameMatch/NameExclude to filter at the database level.
	queryParams := fleet.SoftwareIterQueryOptions{
		IncludedSources: []string{"programs"},
		NameMatch:       "Microsoft (365|Office)",
		NameExclude:     "[Cc]ompanion",
	}
	iter, err := ds.AllSoftwareIterator(ctx, queryParams)
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	var vulnerabilities []fleet.SoftwareVulnerability
	for iter.Next() {
		software, err := iter.Value()
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "getting software from iterator")
		}

		detected := collectVulnerabilities(software, bulletin)
		existing, err := getStoredVulnerabilities(ctx, ds, software.ID)
		if err != nil {
			return nil, err
		}

		inserted, err := updateVulnsInDB(ctx, ds, detected, existing)
		if err != nil {
			return nil, err
		}

		if collectVulns {
			vulnerabilities = append(vulnerabilities, inserted...)
		}
	}

	if err := iter.Err(); err != nil {
		return nil, fmt.Errorf("iter: %w", err)
	}

	return vulnerabilities, nil
}
