package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

func listCertificateAuthoritiesEndpoint(ctx context.Context, req any, svc fleet.Service) (fleet.Errorer, error) {
	certAuths, err := svc.ListCertificateAuthorities(ctx)
	if err != nil {
		return fleet.ListCertificateAuthoritiesResponse{Err: err}, nil
	}
	return fleet.ListCertificateAuthoritiesResponse{
		CertificateAuthorities: certAuths,
	}, nil
}

func (svc *Service) ListCertificateAuthorities(ctx context.Context) ([]*fleet.CertificateAuthoritySummary, error) {
	// skipauth: No authorization check needed due to implementation returning only license error.
	svc.authz.SkipAuthorization(ctx)
	return nil, fleet.ErrMissingLicense
}

func getCertificateAuthorityEndpoint(ctx context.Context, request any, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.GetCertificateAuthorityRequest)
	certAuth, err := svc.GetCertificateAuthority(ctx, req.ID)
	if err != nil {
		return fleet.GetCertificateAuthorityResponse{Err: err}, nil
	}
	return fleet.GetCertificateAuthorityResponse{
		CertificateAuthority: certAuth,
	}, nil
}

func (svc *Service) GetCertificateAuthority(ctx context.Context, id uint) (*fleet.CertificateAuthority, error) {
	// skipauth: No authorization check needed due to implementation returning only license error.
	svc.authz.SkipAuthorization(ctx)
	return nil, fleet.ErrMissingLicense
}

func createCertificateAuthorityEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.CreateCertificateAuthorityRequest)

	ca, err := svc.NewCertificateAuthority(ctx, req.CertificateAuthorityPayload)
	if err != nil {
		return fleet.CreateCertificateAuthorityResponse{Err: err}, nil
	}

	return fleet.CreateCertificateAuthorityResponse{ID: ca.ID, Name: *ca.Name, Type: fleet.CAType(ca.Type)}, nil
}

func (svc *Service) NewCertificateAuthority(ctx context.Context, p fleet.CertificateAuthorityPayload) (*fleet.CertificateAuthority, error) {
	// skipauth: No authorization check needed due to implementation returning only license error.
	svc.authz.SkipAuthorization(ctx)
	return nil, fleet.ErrMissingLicense
}

func deleteCertificateAuthorityEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.DeleteCertificateAuthorityRequest)

	err := svc.DeleteCertificateAuthority(ctx, req.ID)
	if err != nil {
		return &fleet.DeleteCertificateAuthorityResponse{Err: err}, nil
	}

	return &fleet.DeleteCertificateAuthorityResponse{}, nil
}

func (svc *Service) DeleteCertificateAuthority(ctx context.Context, certificateAuthorityID uint) error {
	// skipauth: No authorization check needed due to implementation returning only license error.
	svc.authz.SkipAuthorization(ctx)
	return fleet.ErrMissingLicense
}

func updateCertificateAuthorityEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.UpdateCertificateAuthorityRequest)

	err := svc.UpdateCertificateAuthority(ctx, req.ID, req.CertificateAuthorityUpdatePayload)
	if err != nil {
		return &fleet.UpdateCertificateAuthorityResponse{Err: err}, nil
	}

	return &fleet.UpdateCertificateAuthorityResponse{}, nil
}

func (svc *Service) UpdateCertificateAuthority(ctx context.Context, id uint, payload fleet.CertificateAuthorityUpdatePayload) error {
	// skipauth: No authorization check needed due to implementation returning only license error.
	svc.authz.SkipAuthorization(ctx)
	return fleet.ErrMissingLicense
}

func requestCertificateEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.RequestCertificateRequest)

	certificate, err := svc.RequestCertificate(ctx, req.RequestCertificatePayload)
	if err != nil {
		return fleet.RequestCertificateResponse{Err: err}, nil
	}

	return fleet.RequestCertificateResponse{Certificate: *certificate}, nil
}

func (svc *Service) RequestCertificate(ctx context.Context, p fleet.RequestCertificatePayload) (*string, error) {
	// skipauth: No authorization check needed due to implementation returning only license error.
	svc.authz.SkipAuthorization(ctx)
	return nil, fleet.ErrMissingLicense
}

// TODO(hca): do we need to return anything to facilitate logging by the gitops client?

func batchApplyCertificateAuthoritiesEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.BatchApplyCertificateAuthoritiesRequest)

	// Call the service method to apply the certificate authorities spec
	err := svc.BatchApplyCertificateAuthorities(ctx, req.CertificateAuthorities, req.DryRun, true)
	if err != nil {
		return &fleet.BatchApplyCertificateAuthoritiesResponse{Err: err}, nil
	}

	return &fleet.BatchApplyCertificateAuthoritiesResponse{}, nil
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

func getCertificateAuthoritiesSpecEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.GetCertificateAuthoritiesSpecRequest)

	certificateAuthorities, err := svc.GetGroupedCertificateAuthorities(ctx, req.IncludeSecrets)
	if err != nil {
		return &fleet.GetCertificateAuthoritiesSpecResponse{Err: err}, nil
	}

	return &fleet.GetCertificateAuthoritiesSpecResponse{CertificateAuthorities: certificateAuthorities}, nil
}

func (svc *Service) GetGroupedCertificateAuthorities(ctx context.Context, includeSecrets bool) (*fleet.GroupedCertificateAuthorities, error) {
	if err := svc.authz.Authorize(ctx, &fleet.CertificateAuthority{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	return svc.ds.GetGroupedCertificateAuthorities(ctx, includeSecrets)
}
