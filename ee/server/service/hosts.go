package service

import (
	"context"
	"fmt"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

func (svc *Service) GetHost(ctx context.Context, id uint, opts fleet.HostDetailOptions) (*fleet.HostDetail, error) {
	// reuse GetHost, but include premium details
	opts.IncludeCVEScores = true
	opts.IncludePolicies = true
	return svc.Service.GetHost(ctx, id, opts)
}

func (svc *Service) HostByIdentifier(ctx context.Context, identifier string, opts fleet.HostDetailOptions) (*fleet.HostDetail, error) {
	// reuse HostByIdentifier, but include premium options
	opts.IncludeCVEScores = true
	opts.IncludePolicies = true
	return svc.Service.HostByIdentifier(ctx, identifier, opts)
}

func (svc *Service) OSVersions(ctx context.Context, teamID *uint, platform *string, name *string, version *string) (*fleet.OSVersions, error) {
	// resuse OSVersions, but include CVSS Scores in call to ListVulnsByOS

	if err := svc.authz.Authorize(ctx, &fleet.Host{TeamID: teamID}, fleet.ActionList); err != nil {
		return nil, err
	}

	if name != nil && version == nil {
		return nil, &fleet.BadRequestError{Message: "Cannot specify os_name without os_version"}
	}

	if name == nil && version != nil {
		return nil, &fleet.BadRequestError{Message: "Cannot specify os_version without os_name"}
	}

	osVersions, err := svc.ds.OSVersions(ctx, teamID, platform, name, version)
	if err != nil && fleet.IsNotFound(err) {
		// differentiate case where team was added after UpdateOSVersions last ran
		if teamID != nil && *teamID > 0 {
			// most of the time, team should exist so checking here saves unnecessary db calls
			_, err := svc.ds.Team(ctx, *teamID)
			if err != nil {
				return nil, err
			}
		}
		// if team exists but stats have not yet been gathered, return empty JSON array
		osVersions = &fleet.OSVersions{}
	} else if err != nil {
		return nil, err
	}

	for i, os := range osVersions.OSVersions {
		vulns, err := svc.ds.ListVulnsByOS(ctx, os.ID, true)
		if err != nil {
			return nil, err
		}

		if os.Platform == "darwin" {
			osVersions.OSVersions[i].GeneratedCPEs = []string{
				fmt.Sprintf("cpe:2.3:o:apple:macos:%s:*:*:*:*:*:*:*", os.Version),
				fmt.Sprintf("cpe:2.3:o:apple:mac_os_x:%s:*:*:*:*:*:*:*", os.Version),
			}
		}

		for _, vuln := range vulns {
			switch os.Platform {
			case "darwin":
				vuln.DetailsLink = fmt.Sprintf("https://nvd.nist.gov/vuln/detail/%s", vuln.CVE)
			case "windows":
				vuln.DetailsLink = fmt.Sprintf("https://msrc.microsoft.com/update-guide/en-US/vulnerability/%s", vuln.CVE)
			}
			osVersions.OSVersions[i].Vulnerabilities = append(osVersions.OSVersions[i].Vulnerabilities, vuln)
		}
	}

	return osVersions, nil
}
