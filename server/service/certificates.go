package service

import (
	"context"
	"errors"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

type updateCertificateStatusRequest struct {
	CertificateTemplateID                   string `url:"id"`
	NodeKey 							    string `json:"node_key"`
	Status string `json:"status"`
}

func (r *updateCertificateStatusRequest) hostNodeKey() string {
	return r.NodeKey
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