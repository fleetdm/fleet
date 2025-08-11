package service

import (
	"context"

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
	svc.authz.SkipAuthorization(ctx)
	// Implementation here
	return []*fleet.CertificateAuthoritySummary{{ID: 1, Name: "CA1", Type: "digicert"}}, nil
}

type getCertificateAuthorityRequest struct {
	ID uint `url:"id"`
}

type getCertificateAuthorityResponse struct {
	*fleet.CertificateAuthorityResult
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
		CertificateAuthorityResult: certAuth,
	}, nil
}

func (svc *Service) GetCertificateAuthority(ctx context.Context, id uint) (*fleet.CertificateAuthorityResult, error) {
	svc.authz.SkipAuthorization(ctx)
	// Implementation here
	return &fleet.CertificateAuthorityResult{
		ID:                            id,
		Type:                          "digicert",
		Name:                          "Example DigiCert CA",
		URL:                           "https://example.com",
		APIToken:                      "example-token",
		ProfileID:                     "profile-id",
		CertificateCommonName:         "example.com",
		CertificateUserPrincipalNames: []string{"user@example.com"},
	}, nil
}
