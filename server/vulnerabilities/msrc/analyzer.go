package msrc

import (
	"context"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	io "github.com/fleetdm/fleet/v4/server/vulnerabilities/msrc/io"
	msrc "github.com/fleetdm/fleet/v4/server/vulnerabilities/msrc/parsed"
	utils "github.com/fleetdm/fleet/v4/server/vulnerabilities/utils"
)

const (
	hostsBatchSize = 500
	vulnBatchSize  = 500
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
	osProduct := msrc.NewProductFromOS(os)
	matchingPIDs := make(map[string]bool)
	for pID, p := range bulletin.Products {
		if p.Matches(osProduct) {
			matchingPIDs[pID] = true
		}
	}

	toInsertSet := make(map[string]fleet.OSVulnerability)
	toDeleteSet := make(map[string]fleet.OSVulnerability)

	var offset int
	for {
		hIDs, err := ds.HostIDsByOSID(ctx, os.ID, offset, hostsBatchSize)
		offset += len(hIDs)

		if err != nil {
			return nil, err
		}

		if len(hIDs) == 0 {
			break
		}

		// Run vulnerability detection for all hosts in this batch (hIDs)
		// and store the results in 'foundInBatch'.
		foundInBatch := make(map[uint][]fleet.OSVulnerability, len(hIDs))
		for _, hID := range hIDs {
			updates, err := ds.ListWindowsUpdatesByHostID(ctx, hID)
			if err != nil {
				return nil, err
			}

			var vs []fleet.OSVulnerability
			for cve, v := range bulletin.Vulnerabities {
				// Check if this vulnerability targets the OS
				if !utils.ProductIDsIntersect(v.ProductIDs, matchingPIDs) {
					continue
				}
				if patched(bulletin, v, matchingPIDs, updates) {
					continue
				}
				vs = append(vs, fleet.OSVulnerability{OSID: os.ID, HostID: hID, CVE: cve})
			}
			foundInBatch[hID] = vs
		}

		osvulns, err := ds.ListOSVulnerabilities(ctx, hIDs)
		if err != nil {
			return nil, err
		}

		existingInBatch := make(map[uint][]fleet.OSVulnerability)
		for _, osv := range osvulns {
			existingInBatch[osv.HostID] = append(existingInBatch[osv.HostID], osv)
		}

		for _, hID := range hIDs {
			insrt, del := utils.VulnsDelta(foundInBatch[hID], existingInBatch[hID])
			for _, i := range insrt {
				toInsertSet[i.Key()] = i
			}
			for _, d := range del {
				toDeleteSet[d.Key()] = d
			}
		}

	}

	err = utils.BatchProcess(toDeleteSet, func(v []fleet.OSVulnerability) error {
		return ds.DeleteOSVulnerabilities(ctx, v)
	}, vulnBatchSize)
	if err != nil {
		return nil, err
	}

	var inserted []fleet.OSVulnerability
	if collectVulns {
		inserted = make([]fleet.OSVulnerability, 0, len(toInsertSet))
	}

	err = utils.BatchProcess(toInsertSet, func(v []fleet.OSVulnerability) error {
		n, err := ds.InsertOSVulnerabilities(ctx, v, fleet.MSRCSource)
		if err != nil {
			return err
		}

		if collectVulns && n > 0 {
			for _, e := range v {
				inserted = append(inserted, e)
			}
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
	b *msrc.SecurityBulletin,
	v msrc.Vulnerability,
	matchingPIDs map[string]bool,
	updates []fleet.WindowsUpdate,
) bool {
	// check if any update directly remediates the vulnerability,
	// this will be much faster than walking the forest of vendor fixes.
	for _, u := range updates {
		if v.RemediatedBy[u.KBID] {
			return true
		}
	}

	for KBID := range v.RemediatedBy {
		fix := b.VendorFixes[KBID]

		// Check if this vendor fix targets the OS
		if !utils.ProductIDsIntersect(fix.ProductIDs, matchingPIDs) {
			continue
		}

		for _, u := range updates {
			if b.KBIDsConnected(KBID, u.KBID) {
				return true
			}
		}
	}

	return false
}

// loadBulletin loads the most recent bulletin for the given os
func loadBulletin(os fleet.OperatingSystem, dir string) (*msrc.SecurityBulletin, error) {
	product := msrc.NewProductFromOS(os)
	fileName := io.FileName(product.Name(), time.Now())

	latest, err := utils.LatestFile(fileName, dir)
	if err != nil {
		return nil, err
	}

	return msrc.UnmarshalBulletin(latest)
}
