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
)

func Analyze(
	ctx context.Context,
	ds fleet.Datastore,
	os fleet.OperatingSystem,
	vulnPath string,
	collectVulns bool,
) ([]fleet.Vulnerability, error) {
	bulletin, err := loadBulletin(os, vulnPath)
	if err != nil {
		return nil, err
	}

	osProduct := msrc.NewProductFromOS(os)

	// Find matching products inside the bulletin
	productIDs := make(map[string]bool)
	for pID, p := range bulletin.Products {
		if p.Matches(osProduct) {
			productIDs[pID] = true
		}
	}

	var offset int
	var vulns []fleet.Vulnerability

	for {
		hIDs, err := ds.HostIDsByOSID(ctx, os.ID, offset, hostsBatchSize)
		offset += len(hIDs)

		if err != nil {
			return nil, err
		}

		if len(hIDs) == 0 {
			break
		}

		for _, hID := range hIDs {
			updates, err := ds.ListWindowsUpdatesByHostID(ctx, hID)
			if err != nil {
				return nil, err
			}

			for cve, v := range bulletin.Vulnerabities {
				if !utils.ProductIDsIntersect(v.ProductIDs, productIDs) {
					continue
				}

				if patched(bulletin, v, productIDs, updates) {
					continue
				}

				vulns = append(vulns, fleet.OSVulnerability{OSID: os.ID, HostID: hID, CVE: cve})
			}
		}
	}

	return vulns, nil
}

// patched returns true if the vulnerability (v) is patched by the any of the provided Windows
// updates.
func patched(
	b *msrc.SecurityBulletin,
	v msrc.Vulnerability,
	productIDs map[string]bool,
	updates []fleet.WindowsUpdate,
) bool {
	// check if any update directly remediates the vulnerability,
	// this will be much faster than walking the linked list of vendor fixes.
	for _, u := range updates {
		if v.RemediatedBy[u.KBID] {
			return true
		}
	}

	for KBID := range v.RemediatedBy {
		fix := b.VendorFixes[KBID]

		if !utils.ProductIDsIntersect(fix.ProductIDs, productIDs) {
			continue
		}

		for _, u := range updates {
			if b.Connected(KBID, u.KBID) {
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
