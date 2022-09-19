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
		hostIDs, err := ds.HostIDsByOSID(ctx, os.ID, offset, hostsBatchSize)
		offset += len(hostIDs)

		if err != nil {
			return nil, err
		}

		if len(hostIDs) == 0 {
			break
		}

		for _, hostID := range hostIDs {
			updates, err := ds.ListWindowsUpdatesByHostID(ctx, hostID)
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

				vulns = append(vulns, fleet.OSVulnerability{OSID: os.ID, HostID: hostID, CVE: cve})
			}
		}
	}

	return vulns, nil
}

func patched(
	b *msrc.SecurityBulletin,
	v msrc.Vulnerability,
	productIDs map[string]bool,
	updates []fleet.WindowsUpdate,
) bool {
	// check if any update directly remediates this vulnerability,
	// as this will be much faster than walking the linked list of vendor fixes
	for _, u := range updates {
		if v.RemediatedBy[u.KBID] {
			return true
		}
	}

	for _, u := range updates {
		for KBID := range v.RemediatedBy {
			fix := b.VendorFixes[KBID]
			if !utils.ProductIDsIntersect(fix.ProductIDs, productIDs) {
				continue
			}

			if b.WalkVendorFixes(KBID, func(kbID uint) bool {
				return kbID == u.KBID
			}) {
				return true
			}
		}
	}

	return false
}

func loadBulletin(os fleet.OperatingSystem, vulnPath string) (*msrc.SecurityBulletin, error) {
	product := msrc.NewProductFromFullName(os.Name)
	fileName := io.FileName(product.Name(), time.Now())

	latest, err := utils.LatestFile(fileName, vulnPath)
	if err != nil {
		return nil, err
	}

	return msrc.UnmarshalBulletin(latest)
}
