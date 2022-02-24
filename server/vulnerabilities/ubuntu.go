package vulnerabilities

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/vuln_ubuntu"
	kitlog "github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
)

// ubuntuPostProcessing performs processing over the list of vulnerable ubuntu packages
// and removes the vulnerabilities where the CVEs are known to be fixed.
func ubuntuPostProcessing(
	ctx context.Context,
	ds fleet.Datastore,
	db *sql.DB,
	logger kitlog.Logger,
	config config.FleetConfig,
) error {
	fixedCVEsByPackage, err := vuln_ubuntu.LoadUbuntuFixedCVEs(ctx, db, logger)
	if err != nil {
		return fmt.Errorf("fetch Ubuntu fixed cves: %w", err)
	}

	level.Info(logger).Log("msg", "Loaded fixed cves", "numFixedPackages", len(fixedCVEsByPackage))
	if len(fixedCVEsByPackage) == 0 {
		return nil
	}

	vulnSoftwares, err := ds.ListVulnerableSoftwareBySource(ctx, "deb_packages")
	if err != nil {
		return fmt.Errorf("list vulnerable software: %w", err)
	}

	level.Info(logger).Log("msg", "List vulnerable software", "numVulnerableSoftware", len(vulnSoftwares))
	if len(vulnSoftwares) == 0 {
		return nil
	}

	var fixedVulns []fleet.SoftwareVulnerability
	var fixedCount int
	for _, sw := range vulnSoftwares {
		if sw.Vendor != "Ubuntu" {
			continue
		}

		fixedCVEs, ok := fixedCVEsByPackage[vuln_ubuntu.Package{
			Name:    sw.Name,
			Version: sw.Version,
		}]
		if !ok {
			continue
		}

		// filter fixed cves
		var cves []string
		for _, vulnerability := range sw.Vulnerabilities {
			if _, ok := fixedCVEs[vulnerability.CVE]; ok {
				cves = append(cves, vulnerability.CVE)
				fixedVulns = append(fixedVulns, fleet.SoftwareVulnerability{
					CPEID: sw.CPEID,
					CVE:   vulnerability.CVE,
				})
			}
		}
		if len(cves) > 0 {
			fixedCount++

			level.Debug(logger).Log(
				"msg", "fixedCVEs",
				"software", fmt.Sprintf("%s-%s", sw.Name, sw.Version),
				"softwareCPE", sw.CPEID,
				"cves", fmt.Sprintf("%v", cves),
			)
		}
	}

	level.Info(logger).Log(
		"msg", "Ubuntu fixed CVEs",
		"fixedCVEsCount", len(fixedVulns),
		"distinctFixedSoftwareCount", fixedCount,
	)

	if err := ds.DeleteVulnerabilitiesByCPECVE(ctx, fixedVulns); err != nil {
		return fmt.Errorf("delete fixed vulnerabilities: %w", err)
	}

	return nil
}
