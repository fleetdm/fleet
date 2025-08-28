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

type applyCertificateAuthoritiesSpecRequest struct {
	CertificateAuthorities fleet.GroupedCertificateAuthorities `json:"certificate_authorities"`
	DryRun                 bool                                `json:"dry_run"`
	// ViaGitOps bool                                              `json:"via_git_ops"`
}

type applyCertificateAuthoritiesSpecResponse struct {
	Err error `json:"error,omitempty"`
}

func (r applyCertificateAuthoritiesSpecResponse) Error() error { return r.Err }

// func (r applyCertificateAuthoritiesSpecResponse) Status() int  { return http.StatusNoContent }

func applyCertificateAuthoritiesSpecEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*applyCertificateAuthoritiesSpecRequest)

	// Call the service method to apply the certificate authorities spec
	err := svc.ApplyCertificateAuthoritiesSpec(ctx, req.CertificateAuthorities, req.DryRun, true)
	if err != nil {
		return &applyCertificateAuthoritiesSpecResponse{Err: err}, nil
	}

	return &applyCertificateAuthoritiesSpecResponse{}, nil
}

// func (svc *Service) ApplyCertificateAuthoritiesSpec(ctx context.Context, spec fleet.CertificateAuthoritiesSpec, dryRun bool, viaGitOps bool) error {
// 	// skipauth: No authorization check needed due to implementation returning only license error.
// 	svc.authz.SkipAuthorization(ctx)
// 	return fleet.ErrMissingLicense
// }

func (svc *Service) ApplyCertificateAuthoritiesSpec(ctx context.Context, incoming fleet.GroupedCertificateAuthorities, dryRun bool, viaGitOps bool) error {
	// skipauth: No authorization check needed due to implementation returning only license error.
	svc.authz.SkipAuthorization(ctx)
	return fleet.ErrMissingLicense
}

// // TODO(hca): Implement this!
// func (svc *Service) recordActivitiesForUpdateCAs(ctx context.Context, incoming *fleet.CertificateAuthoritiesSpec, dryRun bool,
// ) (*appConfigCAStatus, error) {
// 	caStatus, _, err := svc.processCertificateAuthoritiesSpec(ctx, incoming)
// 	if err != nil {
// 		return nil, ctxerr.Wrap(ctx, err, "process certificate authorities spec")
// 	}

// 	if dryRun {
// 		return &caStatus, nil
// 	}

// 	// TODO(hca): Implement datastore updates

// 	switch caStatus.ndes {
// 	case caStatusAdded:
// 		if err := svc.NewActivity(ctx, authz.UserFromContext(ctx), fleet.ActivityAddedNDESSCEPProxy{}); err != nil {
// 			return nil, ctxerr.Wrap(ctx, err, "create activity for added NDES SCEP proxy")
// 		}
// 	case caStatusEdited:
// 		if err := svc.NewActivity(ctx, authz.UserFromContext(ctx), fleet.ActivityEditedNDESSCEPProxy{}); err != nil {
// 			return nil, ctxerr.Wrap(ctx, err, "create activity for edited NDES SCEP proxy")
// 		}
// 	case caStatusDeleted:
// 		// Delete stored password
// 		if err := svc.ds.HardDeleteMDMConfigAsset(ctx, fleet.MDMAssetNDESPassword); err != nil {
// 			return nil, ctxerr.Wrap(ctx, err, "delete NDES SCEP password")
// 		}
// 		if err := svc.NewActivity(ctx, authz.UserFromContext(ctx), fleet.ActivityDeletedNDESSCEPProxy{}); err != nil {
// 			return nil, ctxerr.Wrap(ctx, err, "create activity for deleted NDES SCEP proxy")
// 		}
// 	default:
// 		// No change, no activity.
// 	}

// 	var caAssetsToDelete []string
// 	for caName, status := range caStatus.digicert {
// 		switch status {
// 		case caStatusAdded:
// 			if err := svc.NewActivity(ctx, authz.UserFromContext(ctx), fleet.ActivityAddedDigiCert{Name: caName}); err != nil {
// 				return nil, ctxerr.Wrap(ctx, err, "create activity for added DigiCert CA")
// 			}
// 		case caStatusEdited:
// 			if err := svc.NewActivity(ctx, authz.UserFromContext(ctx), fleet.ActivityEditedDigiCert{Name: caName}); err != nil {
// 				return nil, ctxerr.Wrap(ctx, err, "create activity for edited DigiCert CA")
// 			}
// 		case caStatusDeleted:
// 			if _, nameStillExists := caStatus.customSCEPProxy[caName]; !nameStillExists {
// 				caAssetsToDelete = append(caAssetsToDelete, caName)
// 			}
// 			if err := svc.NewActivity(ctx, authz.UserFromContext(ctx), fleet.ActivityDeletedDigiCert{Name: caName}); err != nil {
// 				return nil, ctxerr.Wrap(ctx, err, "create activity for deleted DigiCert CA")
// 			}
// 		}
// 	}
// 	for caName, status := range caStatus.customSCEPProxy {
// 		switch status {
// 		case caStatusAdded:
// 			if err := svc.NewActivity(ctx, authz.UserFromContext(ctx), fleet.ActivityAddedCustomSCEPProxy{Name: caName}); err != nil {
// 				return nil, ctxerr.Wrap(ctx, err, "create activity for added Custom SCEP Proxy")
// 			}
// 		case caStatusEdited:
// 			if err := svc.NewActivity(ctx, authz.UserFromContext(ctx), fleet.ActivityEditedCustomSCEPProxy{Name: caName}); err != nil {
// 				return nil, ctxerr.Wrap(ctx, err, "create activity for edited Custom SCEP Proxy")
// 			}
// 		case caStatusDeleted:
// 			if _, nameStillExists := caStatus.digicert[caName]; !nameStillExists {
// 				caAssetsToDelete = append(caAssetsToDelete, caName)
// 			}
// 			if err := svc.NewActivity(ctx, authz.UserFromContext(ctx), fleet.ActivityDeletedCustomSCEPProxy{Name: caName}); err != nil {
// 				return nil, ctxerr.Wrap(ctx, err, "create activity for deleted Custom SCEP Proxy")
// 			}
// 		}
// 	}
// 	if len(caAssetsToDelete) > 0 {
// 		err := svc.ds.DeleteCAConfigAssets(ctx, caAssetsToDelete)
// 		if err != nil {
// 			return nil, ctxerr.Wrap(ctx, err, "delete CA config assets")
// 		}
// 	}

// 	return &caStatus, nil
// }
