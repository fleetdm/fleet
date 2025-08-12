package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

type createCertificateAuthorityRequest struct {
	fleet.CertificateAuthorityPayload
}

type createCertificateAuthorityResponse struct {
	ID   uint         `json:"id,omitempty"`
	Name string       `json:"name,omitempty"`
	Type fleet.CAType `json:"type,omitempty"`
	Err  error        `json:"error,omitempty"`
}

func (r createCertificateAuthorityResponse) Error() error { return r.Err }

func createCertificateAuthorityEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*createCertificateAuthorityRequest)

	ca, err := svc.NewCertificateAuthority(ctx, req.CertificateAuthorityPayload)
	if err != nil {
		return createCertificateAuthorityResponse{Err: err}, nil
	}

	return createCertificateAuthorityResponse{ID: ca.ID, Name: ca.Name, Type: fleet.CAType(ca.Type)}, nil
}

func (svc *Service) NewCertificateAuthority(ctx context.Context, p fleet.CertificateAuthorityPayload) (*fleet.CertificateAuthority, error) {
	// skipauth: No authorization check needed due to implementation returning only license error.
	svc.authz.SkipAuthorization(ctx)
	return nil, fleet.ErrMissingLicense
}
