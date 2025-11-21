package service

import (
	"context"
	"errors"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

type updateCertificateStatusRequest struct {
	CertificateTemplateID                   string `json:"certificate_template_id"`
	Status string `json:"status"`
}

type updateCertificateStatusResponse struct {
	Err                    error  `json:"error,omitempty"`
}

func (r updateCertificateStatusResponse) Error() error  { return r.Err }	

func updateCertificateStatusEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req, ok := request.(updateCertificateStatusRequest)
	if !ok {
		return nil, errors.New("invalid request")
	}

	err := svc.UpdateCertificateStatus(ctx, req.CertificateTemplateID, req.Status)
	if err != nil {
		return updateCertificateStatusResponse{Err: err}, nil
	}

	return updateCertificateStatusResponse{}, nil
}

func (svc *Service) UpdateCertificateStatus(ctx context.Context, certificateTemplateID, status string) error {
	return nil
	// return svc.ds.UpdateCertificateStatus(ctx, certificateTemplateID, status)
}