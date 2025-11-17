package service

import (
	"context"
	"strconv"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

type applyCertificateSpecsRequest struct {
	Specs []*fleet.CertificateRequestSpec `json:"specs"`
}

type applyCertificateSpecsResponse struct {
	Err error `json:"error,omitempty"`
}

func (r applyCertificateSpecsResponse) Error() error { return r.Err }

func applyCertificateSpecsEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*applyCertificateSpecsRequest)
	err := svc.ApplyCertificateSpecs(ctx, req.Specs)
	if err != nil {
		return applyCertificateSpecsResponse{Err: err}, nil
	}
	return applyCertificateSpecsResponse{}, nil
}

func (svc *Service) ApplyCertificateSpecs(ctx context.Context, specs []*fleet.CertificateRequestSpec) error {
	// TODO: What is the right authorization here?
	// svc.authz.Authorize(ctx, &fleet.Certificate{TeamID: tmID}, fleet.ActionWrite) ?
	if err := svc.authz.Authorize(ctx, &fleet.Team{}, fleet.ActionWrite); err != nil {
		return err
	}

	var certificates []*fleet.Certificate
	for _, spec := range specs {
		var teamID uint
		if spec.Team != "" {
			parsed, err := strconv.ParseUint(spec.Team, 10, 0)
			if err != nil {
				return err
			}
			teamID = uint(parsed)
		}

		cert := &fleet.Certificate{
			Name:                   spec.Name,
			CertificateAuthorityID: spec.CertificateAuthorityId,
			SubjectName:            spec.SubjectName,
			TeamID:                 teamID,
		}
		certificates = append(certificates, cert)
	}

	return svc.ds.BatchUpsertCertificateTemplates(ctx, certificates)
}
