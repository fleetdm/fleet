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
				if v.TargetsAny(productIDs) && !v.PatchedBy(updates) {
					vulns = append(vulns, fleet.OSVulnerability{OSID: os.ID, HostID: hostID, CVE: cve})
				}
			}
		}

		offset += len(hostIDs)
	}

	return vulns, nil
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
