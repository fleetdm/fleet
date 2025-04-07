package msrc

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/io"
	msrc "github.com/fleetdm/fleet/v4/server/vulnerabilities/msrc/parsed"
	utils "github.com/fleetdm/fleet/v4/server/vulnerabilities/utils"
	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

const (
	vulnBatchSize = 500
)

func Analyze(
	ctx context.Context,
	ds fleet.Datastore,
	os fleet.OperatingSystem,
	vulnPath string,
	collectVulns bool,
	logger kitlog.Logger,
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
		// Check if the OS is vulnerable to the vulnerability by referencing the OS kernel version
		isVuln, riv := isOSVulnerable(os.KernelVersion, bulletin, v, matchingPIDs, cve, logger)
		if isVuln {
			found = append(found, fleet.OSVulnerability{
				OSID:              os.ID,
				CVE:               cve,
				Source:            fleet.MSRCSource,
				ResolvedInVersion: ptr.String(riv),
			})
		}
	}

	// Fetch all stored vulnerabilities for the current batch
	osVulns, err := ds.ListOSVulnerabilitiesByOS(ctx, os.ID)
	if err != nil {
		return nil, err
	}
	var existing []fleet.OSVulnerability
	existing = append(existing, osVulns...)

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

// isOSVulnerable returns true if the OS is vulnerable to the given vulnerability.
// If the OS is vulnerable, the function returns the version in which the vulnerability
// was resolved.
func isOSVulnerable(
	osKernel string,
	b *msrc.SecurityBulletin,
	v msrc.Vulnerability,
	matchingPIDs map[string]bool,
	cve string,
	logger kitlog.Logger,
) (isVulnerable bool, resolvedInVersion string) {
	for KBID := range v.RemediatedBy {
		fix := b.VendorFixes[KBID]

		// Check if the MSRC FixedBuild version targets the OS version
		if !utils.ProductIDsIntersect(fix.ProductIDs, matchingPIDs) {
			continue
		}

		for _, build := range fix.FixedBuilds {
			// An empty FixBuild is a bug in the MSRC feed, last
			// seen around apr-2021.  Ignoring it to avoid false
			// positive vulnerabilities.
			if build == "" {
				continue
			}

			fixedBuild, feedParts, err := getBuildNumber(build)
			if err != nil {
				level.Debug(logger).Log("msg", "invalid msrc feed version", "cve", cve, "err", err)
				continue
			}

			osBuild, osParts, err := getBuildNumber(osKernel)
			if err != nil {
				continue
			}

			// skip if the product version number does not match
			// ie. 10.0.22000.X vs 10.0.22631.X
			if isProductVersionMismatch(feedParts, osParts) {
				continue
			}

			if osBuild < fixedBuild {
				return true, build
			}
		}
	}

	return
}

func isProductVersionMismatch(feedVersion, osVersion []string) bool {
	for i := 0; i < 3; i++ {
		if feedVersion[i] != osVersion[i] {
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

// getBuildNumber expects a version string in the format "10.0.22000.194" and
// returns the final part (build number) as an integer and the parts as a slice of strings.
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
