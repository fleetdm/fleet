package msrc

import (
	"context"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	io "github.com/fleetdm/fleet/v4/server/vulnerabilities/msrc/io"
	msrc "github.com/fleetdm/fleet/v4/server/vulnerabilities/msrc/parsed"
	utils "github.com/fleetdm/fleet/v4/server/vulnerabilities/utils"
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

	// Find matching products inside the bulletin
	productIDs := make(map[string]bool)
	for pID, p := range bulletin.Products {
		if p.MatchesOS(os) {
			productIDs[pID] = true
		}
	}

	var vulns []fleet.Vulnerability
	for cve, v := range bulletin.Vulnerabities {
		for pID := range productIDs {
			if v.ProductIDs[pID] {
				// Check if patched
				vulns = append(vulns, fleet.OSVulnerability{OSID: os.ID, CVE: cve})
			}
		}
	}

	return vulns, nil
}

func loadBulletin(os fleet.OperatingSystem, vulnPath string) (*msrc.SecurityBulletin, error) {
	product := msrc.NewProduct(os.Name)
	fileName := io.FileName(product.Name(), time.Now())

	latest, err := utils.LatestFile(fileName, vulnPath)
	if err != nil {
		return nil, err
	}

	return msrc.UnmarshalBulletin(latest)
}
