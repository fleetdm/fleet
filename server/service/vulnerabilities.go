package service

import (
	"context"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

var freeValidVulnSortColumns = []string{
	"cve",
	"host_count",
	"host_count_updated_at",
	"created_at",
}

type listVulnerabilitiesRequest struct {
	fleet.VulnListOptions
}

type listVulnerabilitiesResponse struct {
	Vulnerabilities []fleet.VulnerabilityWithMetadata `json:"vulnerabilities"`
	Count           int                               `json:"count"`
	CountsUpdatedAt time.Time                         `json:"counts_updated_at"`
	Meta            *fleet.PaginationMetadata         `json:"meta,omitempty"`
	Err             error                             `json:"error,omitempty"`
}

func (r listVulnerabilitiesResponse) error() error { return r.Err }

func listVulnerabilitiesEndpoint(ctx context.Context, req interface{}, svc fleet.Service) (errorer, error) {
	request := req.(*listVulnerabilitiesRequest)
	vulns, err := svc.ListVulnerabilities(ctx, request.VulnListOptions)
	if err != nil {
		return listVulnerabilitiesResponse{Err: err}, nil
	}

	return listVulnerabilitiesResponse{Vulnerabilities: vulns}, nil
}

func (svc *Service) ListVulnerabilities(ctx context.Context, opt fleet.VulnListOptions) ([]fleet.VulnerabilityWithMetadata, error) {
	if err := svc.authz.Authorize(ctx, &fleet.AuthzSoftwareInventory{
		TeamID: opt.TeamID,
	}, fleet.ActionRead); err != nil {
		return nil, err
	}

	if len(opt.ValidSortColumns) == 0 {
		opt.ValidSortColumns = freeValidVulnSortColumns
	}
	
	if !opt.HasValidSortColumn() {
		return nil, badRequest("invalid order key")
	}

	return svc.ds.ListVulnerabilities(ctx, opt)
}
