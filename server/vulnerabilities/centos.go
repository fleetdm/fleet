package vulnerabilities

import (
	"context"
	"database/sql"
	"fmt"
	"path"

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
	vulnPath string,
	logger kitlog.Logger,
	config config.FleetConfig,
) error {
	dbPath := path.Join(vulnPath, "cpe.sqlite")
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open cpe database: %w", err)
	}

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
		for _, vulnerability := range software.Vulnerabilities {
			if _, ok := pkgFixedCVEs[vulnerability.CVE]; ok {
				level.Info(logger).Log(
					"msg", "fixed CVE found",
					"softwareName", software.Name,
					"softwareCPE", software.CPE,
					"cve", vulnerability.CVE,
				)
				fixedCVEs = append(fixedCVEs, fleet.SoftwareVulnerability{
					CPE: software.CPE,
					CVE: vulnerability.CVE,
				})
			}
		}
	}

	level.Info(logger).Log("fixedCVEsCount", len(fixedCVEs))

	if err := ds.DeleteVulnerabilitiesByCPECVE(ctx, fixedCVEs); err != nil {
		return fmt.Errorf("failed to delete fixed vulnerabilities: %w", err)
	}
	return nil
}
