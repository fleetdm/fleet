package vulnerabilities

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/vuln_centos"
	kitlog "github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
)

// centosPostProcessing performs processing over the list of vulnerable rpm packages
// and removes the vulnerabilities where the CVEs are known to be fixed.
func centosPostProcessing(
	ctx context.Context,
	ds fleet.Datastore,
	db *sql.DB,
	logger kitlog.Logger,
	config config.FleetConfig,
) error {
	centOSPkgs, err := vuln_centos.LoadCentOSFixedCVEs(ctx, db, logger)
	if err != nil {
		return fmt.Errorf("failed to fetch CentOS packages: %w", err)
	}
	level.Info(logger).Log("centosPackages", len(centOSPkgs))
	if len(centOSPkgs) == 0 {
		return nil
	}

	rpmVulnerable, err := ds.ListVulnerableSoftwareBySource(ctx, "rpm_packages")
	if err != nil {
		return fmt.Errorf("failed to list vulnerable software: %w", err)
	}
	level.Info(logger).Log("vulnerable rpm_packages", len(rpmVulnerable))
	if len(rpmVulnerable) == 0 {
		return nil
	}

	var fixedCVEs []fleet.SoftwareVulnerability
	var softwareCount int
	for _, software := range rpmVulnerable {
		if software.Vendor != "CentOS" {
			continue
		}
		pkgFixedCVEs, ok := centOSPkgs[vuln_centos.CentOSPkg{
			Name:    software.Name,
			Version: software.Version,
			Release: software.Release,
			Arch:    software.Arch,
		}]
		if !ok {
			continue
		}
		var cves []string
		for _, vulnerability := range software.Vulnerabilities {
			if _, ok := pkgFixedCVEs[vulnerability.CVE]; ok {
				cves = append(cves, vulnerability.CVE)
				fixedCVEs = append(fixedCVEs, fleet.SoftwareVulnerability{
					CPEID: software.CPEID,
					CVE:   vulnerability.CVE,
				})
			}
		}
		if len(cves) > 0 {
			softwareCount++

			level.Debug(logger).Log(
				"msg", "fixedCVEs",
				"software", fmt.Sprintf(
					"%s-%s-%s.%s",
					software.Name, software.Version, software.Release, software.Arch,
				),
				"softwareCPE", software.CPEID,
				"cves", fmt.Sprintf("%v", cves),
			)
		}
	}

	level.Info(logger).Log(
		"msg", "CentOS fixed CVEs",
		"fixedCVEsCount", len(fixedCVEs),
		"distinctSoftwareCount", softwareCount,
	)

	if err := ds.DeleteVulnerabilitiesByCPECVE(ctx, fixedCVEs); err != nil {
		return fmt.Errorf("failed to delete fixed vulnerabilities: %w", err)
	}
	return nil
}
