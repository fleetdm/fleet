package service

import (
	"context"
	"net/http"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

type listCertificateAuthoritiesRequest struct{}

type listCertificateAuthoritiesResponse struct {
	CertificateAuthorities []*fleet.CertificateAuthoritySummary `json:"certificate_authorities"`
	Err                    error                                `json:"error,omitempty"`
}

func (r listCertificateAuthoritiesResponse) Error() error { return r.Err }

func listCertificateAuthoritiesEndpoint(ctx context.Context, req any, svc fleet.Service) (fleet.Errorer, error) {
	certAuths, err := svc.ListCertificateAuthorities(ctx)
	if err != nil {
		return listCertificateAuthoritiesResponse{Err: err}, nil
	}
	return listCertificateAuthoritiesResponse{
		CertificateAuthorities: certAuths,
	}, nil
}

func (svc *Service) ListCertificateAuthorities(ctx context.Context) ([]*fleet.CertificateAuthoritySummary, error) {
	// skipauth: No authorization check needed due to implementation returning only license error.
	svc.authz.SkipAuthorization(ctx)
	return nil, fleet.ErrMissingLicense
}

type getCertificateAuthorityRequest struct {
	ID uint `url:"id"`
}

type getCertificateAuthorityResponse struct {
	*fleet.CertificateAuthority
	Err error `json:"error,omitempty"`
}

func (r getCertificateAuthorityResponse) Error() error { return r.Err }

func getCertificateAuthorityEndpoint(ctx context.Context, request any, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*getCertificateAuthorityRequest)
	certAuth, err := svc.GetCertificateAuthority(ctx, req.ID)
	if err != nil {
		return getCertificateAuthorityResponse{Err: err}, nil
	}
	return getCertificateAuthorityResponse{
		CertificateAuthority: certAuth,
	}, nil
}

func (svc *Service) GetCertificateAuthority(ctx context.Context, id uint) (*fleet.CertificateAuthority, error) {
	// skipauth: No authorization check needed due to implementation returning only license error.
	svc.authz.SkipAuthorization(ctx)
	return nil, fleet.ErrMissingLicense
}

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

type deleteCertificateAuthorityRequest struct {
	ID uint `url:"id"`
}

type deleteCertificateAuthorityResponse struct {
	Err error `json:"error,omitempty"`
}

func (r deleteCertificateAuthorityResponse) Error() error { return r.Err }
func (r deleteCertificateAuthorityResponse) Status() int  { return http.StatusNoContent }

func deleteCertificateAuthorityEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*deleteCertificateAuthorityRequest)

	err := svc.DeleteCertificateAuthority(ctx, req.ID)
	if err != nil {
		return &deleteCertificateAuthorityResponse{Err: err}, nil
	}

	return &deleteCertificateAuthorityResponse{}, nil
}

func (svc *Service) DeleteCertificateAuthority(ctx context.Context, certificateAuthorityID uint) error {
	// skipauth: No authorization check needed due to implementation returning only license error.
	svc.authz.SkipAuthorization(ctx)
	return fleet.ErrMissingLicense
}

type applyCertificateAuthoritiesSpecRequest struct {
	Spec fleet.CertificateAuthoritiesSpec `json:"spec"`
}

type applyCertificateAuthoritiesSpecResponse struct {
	Err error `json:"error,omitempty"`
}

func (r applyCertificateAuthoritiesSpecResponse) Error() error { return r.Err }

// func (r applyCertificateAuthoritiesSpecResponse) Status() int  { return http.StatusNoContent }

func applyCertificateAuthoritiesSpecEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*applyCertificateAuthoritiesSpecRequest)

	// Call the service method to apply the certificate authorities spec
	err := svc.ApplyCertificateAuthoritiesSpec(ctx, req.Spec)
	if err != nil {
		return &applyCertificateAuthoritiesSpecResponse{Err: err}, nil
	}

	return &applyCertificateAuthoritiesSpecResponse{}, nil
}

func (svc *Service) ApplyCertificateAuthoritiesSpec(ctx context.Context, spec fleet.CertificateAuthoritiesSpec) error {
	// skipauth: No authorization check needed due to implementation returning only license error.
	svc.authz.SkipAuthorization(ctx)
	return fleet.ErrMissingLicense
}
