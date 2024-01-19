package msrc

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/io"
	msrc "github.com/fleetdm/fleet/v4/server/vulnerabilities/msrc/parsed"
	utils "github.com/fleetdm/fleet/v4/server/vulnerabilities/utils"
)

const (
	hostsBatchSize = 500
	vulnBatchSize = 500
)

func Analyze(
	ctx context.Context,
	ds fleet.Datastore,
	os fleet.OperatingSystem,
	vulnPath string,
	collectVulns bool,
) ([]fleet.OSVulnerability, error) {
	bulletin, err := loadBulletin(os, vulnPath)
	if err != nil {
		return nil, err
	}

	// Find matching products inside the bulletin
	matchingPIDs := make(map[string]bool)
	pID, err := bulletin.Products.GetMatchForOS(ctx, os)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "Analyzing MSRC vulnerabilities")
	}
	if pID != "" {
		matchingPIDs[pID] = true
	}

	if len(matchingPIDs) == 0 {
		return nil, nil
	}

	toInsert := make(map[string]fleet.OSVulnerability)
	toDelete := make(map[string]fleet.OSVulnerability)

	// Run vulnerability detection for all hosts in this batch (hIDs)
	// and store the results in 'found'.
	var found []fleet.OSVulnerability

	for cve, v := range bulletin.Vulnerabities {
		// Check if this vulnerability targets the OS
		if !utils.ProductIDsIntersect(v.ProductIDs, matchingPIDs) {
			continue
		}
		// Check if the vulnerability is patched by referencing the OS version number
		if patched(os, bulletin, v, matchingPIDs) {
			continue
		}
		found = append(found, fleet.OSVulnerability{OSID: os.ID, CVE: cve})
	}

	// Fetch all stored vulnerabilities for the current batch
	osVulns, err := ds.ListOSVulnerabilitiesByOS(ctx, os.ID)
	if err != nil {
		return nil, err
	}
	var existing []fleet.OSVulnerability
	for _, osv := range osVulns {
		existing = append(existing, osv)
	}

	// Compute what needs to be inserted/deleted for this batch
	insrt, del := utils.VulnsDelta(found, existing)
	for _, i := range insrt {
		toInsert[i.Key()] = i
	}
	for _, d := range del {
		toDelete[d.Key()] = d
	}

	err = utils.BatchProcess(toDelete, func(v []fleet.OSVulnerability) error {
		return ds.DeleteOSVulnerabilities(ctx, v)
	}, vulnBatchSize)
	if err != nil {
		return nil, err
	}

	var inserted []fleet.OSVulnerability
	if collectVulns {
		inserted = make([]fleet.OSVulnerability, 0, len(toInsert))
	}

	err = utils.BatchProcess(toInsert, func(v []fleet.OSVulnerability) error {
		n, err := ds.InsertOSVulnerabilities(ctx, v, fleet.MSRCSource)
		if err != nil {
			return err
		}

		if collectVulns && n > 0 {
			inserted = append(inserted, v...)
		}

		return nil
	}, vulnBatchSize)
	if err != nil {
		return nil, err
	}

	return inserted, nil
}

// patched returns true if the vulnerability (v) is patched by the any of the provided Windows
// updates.
func patched(
	os fleet.OperatingSystem,
	b *msrc.SecurityBulletin,
	v msrc.Vulnerability,
	matchingPIDs map[string]bool,
) bool {
	for KBID := range v.RemediatedBy {
		fix := b.VendorFixes[KBID]

		// Check if this vendor fix targets the OS
		if !utils.ProductIDsIntersect(fix.ProductIDs, matchingPIDs) {
			continue
		}

		// An empty FixBuild is a bug in the MSRC feed, last
		// seen around apr-2021.  Ignoring it to avoid false
		// positive vulnerabilities.
		if fix.FixedBuild == "" {
			continue
		}

		isGreater, err := winBuildVersionGreaterOrEqual(fix.FixedBuild, os.KernelVersion)
		if err != nil {
			continue
		}
		if isGreater {
			return true
		}
	}

	return false
}

// loadBulletin loads the most recent bulletin for the given os
func loadBulletin(os fleet.OperatingSystem, dir string) (*msrc.SecurityBulletin, error) {
	product := msrc.NewProductFromOS(os)
	fileName := io.MSRCFileName(product.Name(), time.Now())

	latest, err := utils.LatestFile(fileName, dir)
	if err != nil {
		return nil, err
	}

	return msrc.UnmarshalBulletin(latest)
}

func winBuildVersionGreaterOrEqual(feed, os string) (bool, error) {
	if feed == "" {
		return false, errors.New("empty feed version")
	}

	feedBuild, feedParts, err := getBuildNumber(feed)
	if err != nil {
		return false, fmt.Errorf("invalid feed version: %w", err)
	}

	osBuild, osParts, err := getBuildNumber(os)
	if err != nil {
		return false, fmt.Errorf("invalid os version: %w", err)
	}

	for i := 0; i < 3; i++ {
		if feedParts[i] != osParts[i] {
			return false, fmt.Errorf("comparing different product versions: %s, %s", feed, os)
		}
	}

	return osBuild >= feedBuild, nil
}

func getBuildNumber(version string) (int, []string, error) {
	if version == "" {
		return 0, nil, fmt.Errorf("empty version string %s", version)
	}

	parts := strings.Split(version, ".")
	if len(parts) != 4 {
		return 0, nil, fmt.Errorf("parts count mismatch %s", version)
	}

	build, err := strconv.Atoi(parts[3])
	if err != nil {
		return 0, nil, fmt.Errorf("unable to parse build number %s", version)
	}

	return build, parts, nil
}
