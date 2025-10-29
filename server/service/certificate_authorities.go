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

	return createCertificateAuthorityResponse{ID: ca.ID, Name: *ca.Name, Type: fleet.CAType(ca.Type)}, nil
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

type updateCertificateAuthorityRequest struct {
	ID uint `url:"id"`
	fleet.CertificateAuthorityUpdatePayload
}

type updateCertificateAuthorityResponse struct {
	Err error `json:"error,omitempty"`
}

func (r updateCertificateAuthorityResponse) Error() error { return r.Err }

func updateCertificateAuthorityEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*updateCertificateAuthorityRequest)

	err := svc.UpdateCertificateAuthority(ctx, req.ID, req.CertificateAuthorityUpdatePayload)
	if err != nil {
		return &updateCertificateAuthorityResponse{Err: err}, nil
	}

	return &updateCertificateAuthorityResponse{}, nil
}

func (svc *Service) UpdateCertificateAuthority(ctx context.Context, id uint, payload fleet.CertificateAuthorityUpdatePayload) error {
	// skipauth: No authorization check needed due to implementation returning only license error.
	svc.authz.SkipAuthorization(ctx)
	return fleet.ErrMissingLicense
}

type requestCertificateRequest struct {
	fleet.RequestCertificatePayload
}

type requestCertificateResponse struct {
	Certificate string `json:"certificate"`
	Err         error  `json:"error,omitempty"`
}

func (r requestCertificateResponse) Error() error { return r.Err }

func requestCertificateEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*requestCertificateRequest)

	certificate, err := svc.RequestCertificate(ctx, req.RequestCertificatePayload)
	if err != nil {
		return requestCertificateResponse{Err: err}, nil
	}

	return requestCertificateResponse{Certificate: *certificate}, nil
}

func (svc *Service) RequestCertificate(ctx context.Context, p fleet.RequestCertificatePayload) (*string, error) {
	// skipauth: No authorization check needed due to implementation returning only license error.
	svc.authz.SkipAuthorization(ctx)
	return nil, fleet.ErrMissingLicense
}

type batchApplyCertificateAuthoritiesRequest struct {
	CertificateAuthorities fleet.GroupedCertificateAuthorities `json:"certificate_authorities"`
	DryRun                 bool                                `json:"dry_run"`
}

// TODO(hca): do we need to return anything to facilitate logging by the gitops client?
type batchApplyCertificateAuthoritiesResponse struct {
	Err error `json:"error,omitempty"`
}

func (r batchApplyCertificateAuthoritiesResponse) Error() error { return r.Err }

func batchApplyCertificateAuthoritiesEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*batchApplyCertificateAuthoritiesRequest)

	// Call the service method to apply the certificate authorities spec
	err := svc.BatchApplyCertificateAuthorities(ctx, req.CertificateAuthorities, req.DryRun, true)
	if err != nil {
		return &batchApplyCertificateAuthoritiesResponse{Err: err}, nil
	}

	return &batchApplyCertificateAuthoritiesResponse{}, nil
}

func (svc *Service) BatchApplyCertificateAuthorities(ctx context.Context, incoming fleet.GroupedCertificateAuthorities, dryRun bool, viaGitOps bool) error {
	if err := svc.authz.Authorize(ctx, &fleet.CertificateAuthority{}, fleet.ActionWrite); err != nil {
		return err
	}

	if incoming.NDESSCEP == nil && len(incoming.DigiCert) == 0 && len(incoming.CustomScepProxy) == 0 && len(incoming.Hydrant) == 0 {
		return nil
	}

	return fleet.ErrMissingLicense
}

type getCertificateAuthoritiesSpecRequest struct {
	IncludeSecrets bool `query:"include_secrets,optional"`
}

type getCertificateAuthoritiesSpecResponse struct {
	CertificateAuthorities *fleet.GroupedCertificateAuthorities `json:"certificate_authorities"`
	Err                    error                                `json:"error,omitempty"`
}

func (r getCertificateAuthoritiesSpecResponse) Error() error { return r.Err }

func getCertificateAuthoritiesSpecEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*getCertificateAuthoritiesSpecRequest)

	certificateAuthorities, err := svc.GetGroupedCertificateAuthorities(ctx, req.IncludeSecrets)
	if err != nil {
		return &getCertificateAuthoritiesSpecResponse{Err: err}, nil
	}

	return &getCertificateAuthoritiesSpecResponse{CertificateAuthorities: certificateAuthorities}, nil
}

func (svc *Service) GetGroupedCertificateAuthorities(ctx context.Context, includeSecrets bool) (*fleet.GroupedCertificateAuthorities, error) {
	if err := svc.authz.Authorize(ctx, &fleet.CertificateAuthority{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	return svc.ds.GetGroupedCertificateAuthorities(ctx, includeSecrets)
}
