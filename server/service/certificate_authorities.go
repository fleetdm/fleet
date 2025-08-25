package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"

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
	Spec   fleet.CertificateAuthoritiesSpec `json:"spec"`
	DryRun bool                             `json:"dry_run"`
	// ViaGitOps bool                          `json:"via_git_ops"`
}

type applyCertificateAuthoritiesSpecResponse struct {
	Err error `json:"error,omitempty"`
}

func (r applyCertificateAuthoritiesSpecResponse) Error() error { return r.Err }

// func (r applyCertificateAuthoritiesSpecResponse) Status() int  { return http.StatusNoContent }

func applyCertificateAuthoritiesSpecEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*applyCertificateAuthoritiesSpecRequest)

	// Call the service method to apply the certificate authorities spec
	err := svc.ApplyCertificateAuthoritiesSpec(ctx, req.Spec, req.DryRun, true)
	if err != nil {
		return &applyCertificateAuthoritiesSpecResponse{Err: err}, nil
	}

	return &applyCertificateAuthoritiesSpecResponse{}, nil
}

func (svc *Service) ApplyCertificateAuthoritiesSpec(ctx context.Context, spec fleet.CertificateAuthoritiesSpec, dryRun bool, viaGitOps bool) error {
	// skipauth: No authorization check needed due to implementation returning only license error.
	svc.authz.SkipAuthorization(ctx)
	// return fleet.ErrMissingLicense

	if !viaGitOps {
		// TODO: use fleet.InvalidArgumentError
		return errors.New("certificate authorities spec can only be applied with gitops")
	}

	incoming := fleet.GroupedCertificateAuthorities{}

	// validate
	ops, err := preprocessCAs(ctx, svc, incoming)
	if err != nil {
		return err
	}

	if dryRun {
		// TODO: something?
	}

	// call datastore methods with ops
	fmt.Println(ops)

	// // TODO(hca): record activities based on ops
	// if err := svc.recordActivitiesUpdateCAs(ctx, ops); err != nil {
	// 	return err
	// }

	return nil
}

// TODO(hca): there's probably a more elegant way to implement this, but this will do for now while
// we sort out the details
type caBatchOperations struct {
	add    []*fleet.CertificateAuthorityPayload
	delete []*fleet.CertificateAuthorityPayload
	update []*fleet.CertificateAuthorityPayload
}

func preprocessCAs(ctx context.Context, svc *Service, incoming fleet.GroupedCertificateAuthorities) (*caBatchOperations, error) {
	var invalid *fleet.InvalidArgumentError

	// check for invalid or duplicate names
	allNames := make(map[string]struct{})
	for _, ca := range incoming.DigiCert {
		ca.Name = fleet.Preprocess(ca.Name)
		validateCAName(ca.Name, "digicert", allNames, invalid)
	}
	for _, ca := range incoming.CustomScepProxy {
		ca.Name = fleet.Preprocess(ca.Name)
		validateCAName(ca.Name, "custom_scep_proxy", allNames, invalid)
	}
	for _, ca := range incoming.Hydrant {
		ca.Name = fleet.Preprocess(ca.Name)
		validateCAName(ca.Name, "hydrant", allNames, invalid)
	}

	current, err := svc.ds.GetGroupedCertificateAuthorities(ctx, true)
	if err != nil {
		return nil, err
	}

	// process ndes
	ndesOps, err := preprocessNDESSCEPProxyCA(ctx, svc, incoming.NDESSCEP, current.NDESSCEP, invalid)
	if err != nil {
		invalid.Append("certificate_authorities.ndes_scep_proxy", err.Error()) // TODO: how to wrap this?
	}

	// process digicert
	digicertOps, err := preprocessDigicertCAs(ctx, svc, incoming.DigiCert, current.DigiCert, invalid)
	if err != nil {
		invalid.Append("certificate_authorities.digicert", err.Error()) // TODO: how to wrap this?
		// return nil, err
	}

	// process custom scep proxy
	customSCEPProxyOps, err := preprocessCustomSCEPProxyCAs(ctx, svc, incoming.CustomScepProxy, current.CustomScepProxy, invalid)
	if err != nil {
		// return nil, err
		invalid.Append("certificate_authorities.custom_scep_proxy", err.Error()) // TODO: how to wrap this?
	}

	if invalid != nil {
		return nil, invalid // TODO: confirm this return makes sense
	}

	ops := &caBatchOperations{
		add:    make([]*fleet.CertificateAuthorityPayload, len(digicertOps.add)+len(customSCEPProxyOps.add)),
		delete: make([]*fleet.CertificateAuthorityPayload, len(digicertOps.delete)+len(customSCEPProxyOps.delete)),
		update: make([]*fleet.CertificateAuthorityPayload, len(digicertOps.update)+len(customSCEPProxyOps.update)),
	}

	// populate operations
	ops.add = append(ops.add, ndesOps.add...)
	ops.delete = append(ops.delete, ndesOps.delete...)
	ops.update = append(ops.update, ndesOps.update...)

	ops.add = append(ops.add, digicertOps.add...)
	ops.delete = append(ops.delete, digicertOps.delete...)
	ops.update = append(ops.update, digicertOps.update...)

	ops.add = append(ops.add, customSCEPProxyOps.add...)
	ops.delete = append(ops.delete, customSCEPProxyOps.delete...)
	ops.update = append(ops.update, customSCEPProxyOps.update...)

	fmt.Printf("%+v\n", digicertOps)
	fmt.Printf("%+v\n", customSCEPProxyOps)

	return nil, nil
}

func preprocessNDESSCEPProxyCA(ctx context.Context, svc *Service, incoming *fleet.NDESSCEPProxyCA, current *fleet.NDESSCEPProxyCA, invalid *fleet.InvalidArgumentError) (*caBatchOperations, error) {
	if incoming == nil && current == nil {
		// do nothing
		return &caBatchOperations{}, invalid
	}

	if incoming == nil {
		// delete current
		return &caBatchOperations{
			delete: []*fleet.CertificateAuthorityPayload{
				{NDESSCEPProxy: current},
			},
		}, invalid
	}

	// preprocess ndes (but do not preprocess password)
	incoming.URL = fleet.Preprocess(incoming.URL)
	incoming.AdminURL = fleet.Preprocess(incoming.AdminURL)
	incoming.Username = fleet.Preprocess(incoming.Username)

	if len(incoming.Password) == 0 || incoming.Password == fleet.MaskedPassword {
		invalid.Append("certificate_authorities.ndes_scep_proxy.password",
			"NDES SCEP proxy password must be set")
	}

	if current != nil && incoming.URL == current.URL && incoming.AdminURL == current.AdminURL && incoming.Username == current.Username && incoming.Password == current.Password {
		// If all fields are identical, we can skip further validation
		return &caBatchOperations{}, invalid
	}

	if err := svc.scepConfigService.ValidateNDESSCEPAdminURL(ctx, *incoming); err != nil {
		invalid.Append("certificate_authorities.ndes_scep_proxy", err.Error())
	}
	if err := svc.scepConfigService.ValidateSCEPURL(ctx, incoming.URL); err != nil {
		invalid.Append("certificate_authorities.ndes_scep_proxy.url", err.Error())
	}

	payload := &fleet.CertificateAuthorityPayload{NDESSCEPProxy: incoming}
	if current == nil {
		return &caBatchOperations{
			add: []*fleet.CertificateAuthorityPayload{payload},
		}, invalid
	}

	return &caBatchOperations{
		update: []*fleet.CertificateAuthorityPayload{payload},
	}, invalid
}

func preprocessDigicertCAs(ctx context.Context, svc *Service, incoming []fleet.DigiCertCA, current []fleet.DigiCertCA, invalid *fleet.InvalidArgumentError) (*caBatchOperations, error) {
	ops := &caBatchOperations{
		add:    make([]*fleet.CertificateAuthorityPayload, 0),
		delete: make([]*fleet.CertificateAuthorityPayload, 0),
		update: make([]*fleet.CertificateAuthorityPayload, 0),
	}

	if len(incoming) > 0 && len(svc.config.Server.PrivateKey) == 0 {
		invalid.Append("certificate_authorities.digicert",
			"Cannot encrypt DigiCert API token. Missing required private key. Learn how to configure the private key here: https://fleetdm."+
				"com/learn-more-about/fleet-server-private-key")
		return nil, invalid
	}

	incomingByName := make(map[string]*fleet.DigiCertCA)
	for _, ca := range incoming {
		if _, ok := incomingByName[ca.Name]; ok {
			// TODO: error for duplicate incoming CA? but this is already handled by the caller?
		}

		validateCACN(ca.CertificateCommonName, invalid)
		validateSeatID(ca.CertificateSeatID, invalid)
		validateUserPrincipalNames(ca.CertificateUserPrincipalNames, invalid)
		ca.URL = fleet.Preprocess(ca.URL)
		ca.ProfileID = fleet.Preprocess(ca.ProfileID)
		if len(ca.APIToken) == 0 || ca.APIToken == fleet.MaskedPassword {
			invalid.Append("certificate_authorities.digicert.api_token",
				fmt.Sprintf("DigiCert API token must be set on CA: %s", ca.Name))
		}
		incomingByName[ca.Name] = &ca // TODO(hca): confirm range pointer semantics
	}

	currentByName := make(map[string]*fleet.DigiCertCA)
	for _, ca := range current {
		// TODO(hca): confirm range pointer semantics; confirm duplicate detection not necessary
		currentByName[ca.Name] = &ca
		// if this CA isn't in the incoming list, we should delete it
		if _, ok := incomingByName[ca.Name]; !ok {
			ops.delete = append(ops.delete, &fleet.CertificateAuthorityPayload{
				DigiCert: &ca,
			})
		}
	}

	for name, ca := range incomingByName {
		payload := &fleet.CertificateAuthorityPayload{DigiCert: ca}
		var skipVerify bool
		if currentCA, ok := currentByName[name]; ok {
			ops.update = append(ops.update, payload) // update existing CA
			skipVerify = ca.StrictEquals(currentCA)  // if the incoming CA is identical to the current CA, we can skip further verification
		} else {
			ops.add = append(ops.add, payload) // add new CA
		}

		if u, err := url.ParseRequestURI(ca.URL); err != nil {
			invalid.Append("certificate_authorities.digicert.url", err.Error())
			skipVerify = true // skip further verification
		} else if u.Scheme != "https" && u.Scheme != "http" {
			invalid.Append("certificate_authorities.digicert.url", "digicert URL must be https or http")
			skipVerify = true // skip further verification
		}

		if skipVerify {
			continue
		}

		if err := svc.digiCertService.VerifyProfileID(ctx, *ca); err != nil {
			invalid.Append("certificate_authorities.digicert.profile_id",
				fmt.Sprintf("Could not verify DigiCert profile ID %s for CA %s: %s", ca.ProfileID, ca.Name, err))
		}
	}

	return ops, nil
}

func preprocessCustomSCEPProxyCAs(ctx context.Context, svc *Service, incoming []fleet.CustomSCEPProxyCA, current []fleet.CustomSCEPProxyCA, invalid *fleet.InvalidArgumentError) (*caBatchOperations, error) {
	ops := &caBatchOperations{
		add:    make([]*fleet.CertificateAuthorityPayload, 0),
		delete: make([]*fleet.CertificateAuthorityPayload, 0),
		update: make([]*fleet.CertificateAuthorityPayload, 0),
	}

	if len(incoming) > 0 && len(svc.config.Server.PrivateKey) == 0 {
		invalid.Append("certificate_authorities.custom_scep_proxy",
			"Cannot encrypt SCEP challenge. Missing required private key. Learn how to configure the private key here: "+
				"https://fleetdm.com/learn-more-about/fleet-server-private-key")
		return nil, invalid
	}

	incomingByName := make(map[string]*fleet.CustomSCEPProxyCA)
	for _, ca := range incoming {
		if _, ok := incomingByName[ca.Name]; ok {
			// TODO: error for duplicate incoming CA?
		}
		incomingByName[ca.Name] = &ca

	}

	currentByName := make(map[string]*fleet.CustomSCEPProxyCA)
	for _, ca := range current {
		// TODO(hca): confirm range pointer semantics; confirm duplicate detection not necessary
		currentByName[ca.Name] = &ca
		// if this CA isn't in the incoming list, we should delete it
		if _, ok := incomingByName[ca.Name]; !ok {
			ops.delete = append(ops.delete, &fleet.CertificateAuthorityPayload{
				CustomSCEPProxy: &ca,
			})
		}
	}

	for name, ca := range incomingByName {
		var skipVerify bool
		payload := &fleet.CertificateAuthorityPayload{CustomSCEPProxy: ca}
		if _, ok := currentByName[name]; ok {
			ops.update = append(ops.update, payload) // update existing CA
		} else {
			ops.add = append(ops.add, payload) // add new CA
		}

		if len(ca.Challenge) == 0 || ca.Challenge == fleet.MaskedPassword {
			invalid.Append("certificate_authorities.custom_scep_proxy.challenge",
				fmt.Sprintf("Custom SCEP challenge must be set on CA: %s", ca.Name))
			skipVerify = true
		}
		ca.URL = fleet.Preprocess(ca.URL)
		if u, err := url.ParseRequestURI(ca.URL); err != nil {
			invalid.Append("certificate_authorities.custom_scep_proxy.url", err.Error())
			skipVerify = true
		} else if u.Scheme != "https" && u.Scheme != "http" {
			invalid.Append("certificate_authorities.custom_scep_proxy.url", "custom_scep_proxy URL must be https or http")
			skipVerify = true
		}

		if skipVerify {
			continue
		}

		// Note: Unlike DigiCert, we always validate the connection for existing custom SCEP even if the update will be no-op
		if err := svc.scepConfigService.ValidateSCEPURL(ctx, ca.URL); err != nil {
			invalid.Append("certificate_authorities.custom_scep_proxy.url", err.Error())
			continue
		}
	}

	return ops, nil
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
