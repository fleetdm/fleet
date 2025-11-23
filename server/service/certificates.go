package service

import (
	"context"
	"errors"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	hostctx "github.com/fleetdm/fleet/v4/server/contexts/host"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

type updateCertificateStatusRequest struct {
	CertificateTemplateID uint   `url:"id"`
	NodeKey               string `json:"node_key"`
	Status                string `json:"status"`
}

func (r *updateCertificateStatusRequest) hostNodeKey() string {
	return r.NodeKey
}

type updateCertificateStatusResponse struct {
	Err error `json:"error,omitempty"`
}

func (r updateCertificateStatusResponse) Error() error { return r.Err }

func updateCertificateStatusEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req, ok := request.(*updateCertificateStatusRequest)
	if !ok {
		return nil, errors.New("invalid request")
	}

	err := svc.UpdateCertificateStatus(ctx, req.CertificateTemplateID, fleet.OSSettingsStatus(req.Status))
	if err != nil {
		return updateCertificateStatusResponse{Err: err}, nil
	}

	return updateCertificateStatusResponse{}, nil
}

func (svc *Service) UpdateCertificateStatus(ctx context.Context, certificateTemplateID uint, status fleet.OSSettingsStatus) error {
	// this is not a user-authenticated endpoint
	svc.authz.SkipAuthorization(ctx)

	host, ok := hostctx.FromContext(ctx)
	if !ok {
		err := ctxerr.Wrap(ctx, fleet.NewAuthRequiredError("internal error: missing host from request context"))
		return err
	}

	// Validate the status.
	if !status.IsValid() {
		return fleet.NewInvalidArgumentError("status", string(status))
	}

	return svc.ds.UpdateCertificateStatus(ctx, host.UUID, certificateTemplateID, fleet.OSSettingsStatus(status))
}
