package service

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/variables"
	"github.com/go-kit/log/level"
)

func (svc *Service) GetCertificateAuthority(ctx context.Context, id uint) (*fleet.CertificateAuthority, error) {
	if err := svc.authz.Authorize(ctx, &fleet.CertificateAuthority{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	ca, err := svc.ds.GetCertificateAuthorityByID(ctx, id, false)
	if err != nil {
		return nil, err
	}

	return ca, nil
}

func (svc *Service) ListCertificateAuthorities(ctx context.Context) ([]*fleet.CertificateAuthoritySummary, error) {
	if err := svc.authz.Authorize(ctx, &fleet.CertificateAuthority{}, fleet.ActionList); err != nil {
		return nil, err
	}
	cas, err := svc.ds.ListCertificateAuthorities(ctx)
	if err != nil {
		return nil, err
	}

	return cas, nil
}

func (svc *Service) NewCertificateAuthority(ctx context.Context, p fleet.CertificateAuthorityPayload) (*fleet.CertificateAuthority, error) {
	if err := svc.authz.Authorize(ctx, &fleet.CertificateAuthority{}, fleet.ActionWrite); err != nil {
		return nil, err
	}

	errPrefix := "Couldn't add certificate authority. "

	if err := svc.validatePayload(&p, errPrefix); err != nil {
		return nil, err
	}

	caToCreate := &fleet.CertificateAuthority{}

	var activity fleet.ActivityDetails

	caDisplayType := "Unknown"

	if p.DigiCert != nil {
		p.DigiCert.Name = fleet.Preprocess(p.DigiCert.Name)
		p.DigiCert.URL = fleet.Preprocess(p.DigiCert.URL)
		p.DigiCert.ProfileID = fleet.Preprocess(p.DigiCert.ProfileID)
		if err := svc.validateDigicert(ctx, p.DigiCert, errPrefix); err != nil {
			return nil, err
		}
		caToCreate.Type = string(fleet.CATypeDigiCert)
		caToCreate.Name = &p.DigiCert.Name
		caToCreate.URL = &p.DigiCert.URL
		caToCreate.APIToken = &p.DigiCert.APIToken
		caToCreate.ProfileID = ptr.String(p.DigiCert.ProfileID)
		caToCreate.CertificateCommonName = &p.DigiCert.CertificateCommonName
		caToCreate.CertificateUserPrincipalNames = &p.DigiCert.CertificateUserPrincipalNames
		caToCreate.CertificateSeatID = &p.DigiCert.CertificateSeatID
		caDisplayType = "DigiCert"
		activity = fleet.ActivityAddedDigiCert{Name: p.DigiCert.Name}
	}

	if p.Hydrant != nil {
		p.Hydrant.Name = fleet.Preprocess(p.Hydrant.Name)
		p.Hydrant.URL = fleet.Preprocess(p.Hydrant.URL)
		if err := svc.validateHydrant(ctx, p.Hydrant, errPrefix); err != nil {
			return nil, err
		}

		caToCreate.Type = string(fleet.CATypeHydrant)
		caToCreate.Name = &p.Hydrant.Name
		caToCreate.URL = &p.Hydrant.URL
		caToCreate.ClientID = &p.Hydrant.ClientID
		caToCreate.ClientSecret = &p.Hydrant.ClientSecret
		caDisplayType = "Hydrant"
		activity = fleet.ActivityAddedHydrant{}
	}

	if p.NDESSCEPProxy != nil {
		p.NDESSCEPProxy.URL = fleet.Preprocess(p.NDESSCEPProxy.URL)
		p.NDESSCEPProxy.AdminURL = fleet.Preprocess(p.NDESSCEPProxy.AdminURL)
		p.NDESSCEPProxy.Username = fleet.Preprocess(p.NDESSCEPProxy.Username)
		if err := svc.validateNDESSCEPProxy(ctx, p.NDESSCEPProxy, errPrefix); err != nil {
			return nil, err
		}

		caToCreate.Name = ptr.String("NDES")
		caToCreate.Type = string(fleet.CATypeNDESSCEPProxy)
		caToCreate.URL = &p.NDESSCEPProxy.URL
		caToCreate.AdminURL = ptr.String(p.NDESSCEPProxy.AdminURL)
		caToCreate.Username = ptr.String(p.NDESSCEPProxy.Username)
		caToCreate.Password = &p.NDESSCEPProxy.Password
		caDisplayType = "NDES SCEP"
		activity = fleet.ActivityAddedNDESSCEPProxy{}
	}

	if p.CustomSCEPProxy != nil {
		p.CustomSCEPProxy.Name = fleet.Preprocess(p.CustomSCEPProxy.Name)
		p.CustomSCEPProxy.URL = fleet.Preprocess(p.CustomSCEPProxy.URL)

		if err := svc.validateCustomSCEPProxy(ctx, p.CustomSCEPProxy, errPrefix); err != nil {
			return nil, err
		}

		caToCreate.Type = string(fleet.CATypeCustomSCEPProxy)
		caToCreate.Name = &p.CustomSCEPProxy.Name
		caToCreate.URL = &p.CustomSCEPProxy.URL
		caToCreate.Challenge = &p.CustomSCEPProxy.Challenge
		caDisplayType = "custom SCEP"
		activity = fleet.ActivityAddedCustomSCEPProxy{Name: p.CustomSCEPProxy.Name}
	}

	createdCA, err := svc.ds.NewCertificateAuthority(ctx, caToCreate)
	if err != nil {
		if errors.As(err, &fleet.ConflictError{}) {
			if caToCreate.Type == string(fleet.CATypeNDESSCEPProxy) {
				return nil, &fleet.BadRequestError{Message: fmt.Sprintf("%s. Only a single NDES CA can be added.", errPrefix)}
			}
			return nil, &fleet.BadRequestError{Message: fmt.Sprintf("%s\"%s\" name is already used by another %s certificate authority. Please choose a different name and try again.", errPrefix, *caToCreate.Name, caDisplayType)}
		}
		return nil, err
	}

	if err := svc.NewActivity(ctx, authz.UserFromContext(ctx), activity); err != nil {
		return nil, fmt.Errorf("recording activity for new %s certificate authority %s: %w", caToCreate.Type, *caToCreate.Name, err)
	}

	return createdCA, nil
}

func (svc *Service) validatePayload(p *fleet.CertificateAuthorityPayload, errPrefix string) error {
	casToCreate := 0
	if p.DigiCert != nil {
		casToCreate++
	}
	if p.Hydrant != nil {
		casToCreate++
	}
	if p.NDESSCEPProxy != nil {
		casToCreate++
	}
	if p.CustomSCEPProxy != nil {
		casToCreate++
	}
	if casToCreate == 0 {
		return &fleet.BadRequestError{Message: fmt.Sprintf("%sA certificate authority must be specified", errPrefix)}
	}
	if casToCreate > 1 {
		return &fleet.BadRequestError{Message: fmt.Sprintf("%sOnly one certificate authority can be created at a time", errPrefix)}
	}

	if len(svc.config.Server.PrivateKey) == 0 {
		return &fleet.BadRequestError{Message: fmt.Sprintf("%sPrivate key must be configured. Learn more: https://fleetdm.com/learn-more-about/fleet-server-private-key", errPrefix)}
	}
	return nil
}

func (svc *Service) validateDigicert(ctx context.Context, digicertCA *fleet.DigiCertCA, errPrefix string) error {
	if err := validateURL(digicertCA.URL, "DigiCert", errPrefix); err != nil {
		return err
	}
	if digicertCA.APIToken == "" {
		return fleet.NewInvalidArgumentError("api_token", fmt.Sprintf("%sInvalid API token. Please correct and try again.", errPrefix))
	}
	if digicertCA.ProfileID == "" {
		return fleet.NewInvalidArgumentError("profile_id", fmt.Sprintf("%sInvalid profile GUID. Please correct and try again.", errPrefix))
	}
	if err := validateCAName(digicertCA.Name, errPrefix); err != nil {
		return err
	}
	if err := validateDigicertCACN(digicertCA.CertificateCommonName, errPrefix); err != nil {
		return err
	}
	if err := validateDigicertUserPrincipalNames(digicertCA.CertificateUserPrincipalNames, errPrefix); err != nil {
		return err
	}
	if err := validateDigicertSeatID(digicertCA.CertificateSeatID, errPrefix); err != nil {
		return err
	}

	if err := svc.digiCertService.VerifyProfileID(ctx, *digicertCA); err != nil {
		level.Error(svc.logger).Log("msg", "Failed to validate DigiCert profile GUID", "err", err)
		return &fleet.BadRequestError{Message: fmt.Sprintf("%sCould not verify DigiCert profile ID: %s. Please correct and try again.", errPrefix, err.Error())}
	}
	return nil
}

func validateCAName(name string, errPrefix string) error {
	// This is used by NDES itself which doesn't have a name the user can set so we must reserve it
	if name == "NDES" {
		return fleet.NewInvalidArgumentError("name", fmt.Sprintf("%sCA name cannot be NDES", errPrefix))
	}
	if len(name) == 0 {
		return fleet.NewInvalidArgumentError("name", fmt.Sprintf("%sCA name cannot be empty", errPrefix))
	}
	if len(name) > 255 {
		return fleet.NewInvalidArgumentError("name", fmt.Sprintf("%sCA name cannot be longer than 255 characters", errPrefix))
	}
	if !isAlphanumeric(name) {
		return fleet.NewInvalidArgumentError("name", fmt.Sprintf("%sInvalid characters in the \"name\" field. Only letters, numbers and underscores allowed.", errPrefix))
	}
	return nil
}

func validateDigicertCACN(cn string, errPrefix string) error {
	if len(strings.TrimSpace(cn)) == 0 {
		return fleet.NewInvalidArgumentError("certificate_common_name", fmt.Sprintf("%sCA Common Name (CN) cannot be empty", errPrefix))
	}
	fleetVars := variables.Find(cn)
	for fleetVar := range fleetVars {
		switch fleetVar {
		case string(fleet.FleetVarHostEndUserEmailIDP), string(fleet.FleetVarHostHardwareSerial):
			// ok
		default:
			return fleet.NewInvalidArgumentError("certificate_common_name", fmt.Sprintf("%sFLEET_VAR_%s is not allowed in CA Common Name (CN)", errPrefix, fleetVar))
		}
	}
	return nil
}

var alphanumeric = regexp.MustCompile(`^\w+$`)

func isAlphanumeric(s string) bool {
	return alphanumeric.MatchString(s)
}

func validateDigicertSeatID(seatID string, errPrefix string) error {
	if len(strings.TrimSpace(seatID)) == 0 {
		return fleet.NewInvalidArgumentError("certificate_seat_id", fmt.Sprintf("%sCA Seat ID cannot be empty", errPrefix))
	}
	fleetVars := variables.Find(seatID)
	for fleetVar := range fleetVars {
		switch fleetVar {
		case string(fleet.FleetVarHostEndUserEmailIDP), string(fleet.FleetVarHostHardwareSerial):
			// ok
		default:
			return fleet.NewInvalidArgumentError("certificate_seat_id", fmt.Sprintf("%sFLEET_VAR_%s is not allowed in DigiCert Seat ID", errPrefix, fleetVar))
		}
	}
	return nil
}

func validateDigicertUserPrincipalNames(userPrincipalNames []string, errPrefix string) error {
	if len(userPrincipalNames) == 0 {
		return nil
	}
	if len(userPrincipalNames) > 1 {
		return fleet.NewInvalidArgumentError("certificate_user_principal_names",
			fmt.Sprintf("%sCurrently, only one item can be added to certificate_user_principal_names.", errPrefix))
	}
	if len(strings.TrimSpace(userPrincipalNames[0])) == 0 {
		return fleet.NewInvalidArgumentError("certificate_user_principal_names",
			fmt.Sprintf("%sDigiCert certificate_user_principal_name cannot be empty if specified", errPrefix))
	}
	fleetVars := variables.Find(userPrincipalNames[0])
	for fleetVar := range fleetVars {
		switch fleetVar {
		case string(fleet.FleetVarHostEndUserEmailIDP), string(fleet.FleetVarHostHardwareSerial):
			// ok
		default:
			return fleet.NewInvalidArgumentError("certificate_user_principal_names",
				fmt.Sprintf("%sFLEET_VAR_%s is not allowed in CA User Principal Name", errPrefix, fleetVar))
		}
	}
	return nil
}

func (svc *Service) validateHydrant(ctx context.Context, hydrantCA *fleet.HydrantCA, errPrefix string) error {
	if err := validateCAName(hydrantCA.Name, errPrefix); err != nil {
		return err
	}
	if err := validateURL(hydrantCA.URL, "Hydrant", errPrefix); err != nil {
		return err
	}
	if hydrantCA.ClientID == "" {
		return fleet.NewInvalidArgumentError("client_id", fmt.Sprintf("%sInvalid Hydrant Client ID. Please correct and try again.", errPrefix))
	}
	if hydrantCA.ClientSecret == "" {
		return fleet.NewInvalidArgumentError("client_secret", fmt.Sprintf("%sInvalid Hydrant Client Secret. Please correct and try again.", errPrefix))
	}
	if err := svc.hydrantService.ValidateHydrantURL(ctx, *hydrantCA); err != nil {
		return fleet.NewInvalidArgumentError("url", fmt.Sprintf("%sInvalid Hydrant URL. Please correct and try again.", errPrefix))
	}
	return nil
}

func validateURL(caURL, displayType, errPrefix string) error {
	if u, err := url.ParseRequestURI(caURL); err != nil {
		return fleet.NewInvalidArgumentError("url", fmt.Sprintf("%sInvalid %s URL. Please correct and try again.", errPrefix, displayType))
	} else if u.Scheme != "https" && u.Scheme != "http" {
		return fleet.NewInvalidArgumentError("url", fmt.Sprintf("%s%s URL scheme must be https or http", errPrefix, displayType))
	}
	return nil
}

func (svc *Service) validateNDESSCEPProxy(ctx context.Context, ndesSCEP *fleet.NDESSCEPProxyCA, errPrefix string) error {
	if err := validateURL(ndesSCEP.URL, "NDES SCEP", errPrefix); err != nil {
		return err
	}
	if err := svc.scepConfigService.ValidateSCEPURL(ctx, ndesSCEP.URL); err != nil {
		level.Error(svc.logger).Log("msg", "Failed to validate NDES SCEP URL", "err", err)
		return &fleet.BadRequestError{Message: fmt.Sprintf("%sInvalid SCEP URL. Please correct and try again.", errPrefix)}
	}
	if err := svc.scepConfigService.ValidateNDESSCEPAdminURL(ctx, *ndesSCEP); err != nil {
		level.Error(svc.logger).Log("msg", "Failed to validate NDES SCEP admin URL", "err", err)
		switch {
		case errors.As(err, &NDESPasswordCacheFullError{}):
			return &fleet.BadRequestError{Message: fmt.Sprintf("%sThe NDES password cache is full. Please increase the number of cached passwords in NDES and try again.", errPrefix)}
		case errors.As(err, &NDESInsufficientPermissionsError{}):
			return &fleet.BadRequestError{Message: fmt.Sprintf("%sInsufficient permissions for NDES SCEP admin URL. Please correct and try again.", errPrefix)}
		default:
			return &fleet.BadRequestError{Message: fmt.Sprintf("%sInvalid NDES SCEP admin URL or credentials. Please correct and try again.", errPrefix)}
		}
	}
	return nil
}

func (svc *Service) validateCustomSCEPProxy(ctx context.Context, customSCEP *fleet.CustomSCEPProxyCA, errPrefix string) error {
	if err := validateCAName(customSCEP.Name, errPrefix); err != nil {
		return err
	}
	if err := validateURL(customSCEP.URL, "SCEP", errPrefix); err != nil {
		return err
	}
	if customSCEP.Challenge == "" {
		return fleet.NewInvalidArgumentError("challenge", fmt.Sprintf("%sCustom SCEP Proxy challenge cannot be empty", errPrefix))
	}
	if err := svc.scepConfigService.ValidateSCEPURL(ctx, customSCEP.URL); err != nil {
		level.Error(svc.logger).Log("msg", "Failed to validate custom SCEP URL", "err", err)
		return &fleet.BadRequestError{Message: fmt.Sprintf("%sInvalid SCEP URL. Please correct and try again.", errPrefix)}
	}
	return nil
}

type oauthIntrospectionResponse struct {
	Username *string `json:"username"`
	// Only active is required in the body by the spec
	Active bool `json:"active"`
}

func (svc *Service) DeleteCertificateAuthority(ctx context.Context, certificateAuthorityID uint) error {
	if err := svc.authz.Authorize(ctx, &fleet.CertificateAuthority{}, fleet.ActionWrite); err != nil {
		return err
	}

	ca, err := svc.ds.DeleteCertificateAuthority(ctx, certificateAuthorityID)
	if err != nil {
		return err
	}

	var activity fleet.ActivityDetails
	switch ca.Type {
	case string(fleet.CATypeCustomSCEPProxy):
		activity = fleet.ActivityDeletedCustomSCEPProxy{
			Name: ca.Name,
		}
	case string(fleet.CATypeDigiCert):
		activity = fleet.ActivityDeletedDigiCert{
			Name: ca.Name,
		}
	case string(fleet.CATypeNDESSCEPProxy):
		activity = fleet.ActivityDeletedNDESSCEPProxy{}
	case string(fleet.CATypeHydrant):
		activity = fleet.ActivityDeletedHydrant{
			Name: ca.Name,
		}
	}

	if err := svc.NewActivity(ctx, authz.UserFromContext(ctx), activity); err != nil {
		return err
	}

	return nil
}

func (svc *Service) ApplyCertificateAuthoritiesSpec(ctx context.Context, incoming fleet.CertificateAuthoritiesSpec, dryRun bool, viaGitOps bool) error {
	if err := svc.authz.Authorize(ctx, &fleet.CertificateAuthority{}, fleet.ActionWrite); err != nil {
		return err
	}

	// TODO: Implement the logic to apply the certificate authorities spec
	return nil
}

func (svc *Service) UpdateCertificateAuthority(ctx context.Context, id uint, p fleet.CertificateAuthorityUpdatePayload) error {
	if err := svc.authz.Authorize(ctx, &fleet.CertificateAuthority{}, fleet.ActionWrite); err != nil {
		return err
	}

	errPrefix := "Couldn't edit certificate authority. "

	if err := p.ValidatePayload(svc.config.Server.PrivateKey, errPrefix); err != nil {
		return err
	}

	oldCA, err := svc.ds.GetCertificateAuthorityByID(ctx, id, true)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return &fleet.BadRequestError{Message: fmt.Sprintf("%sCertificate authority with ID %d does not exist.", errPrefix, id)}
		}
		return err
	}

	caToUpdate := fleet.CertificateAuthority{}
	var activity fleet.ActivityDetails
	var caActivityName string

	if p.DigiCertCAUpdatePayload != nil {
		if err := p.DigiCertCAUpdatePayload.ValidateRelatedFields(errPrefix, *oldCA.Name); err != nil {
			return err
		}
		p.DigiCertCAUpdatePayload.Preprocess()
		if err := svc.validateDigicertUpdate(ctx, p.DigiCertCAUpdatePayload, oldCA, errPrefix); err != nil {
			return err
		}
		caToUpdate.Type = string(fleet.CATypeDigiCert)
		caToUpdate.Name = p.DigiCertCAUpdatePayload.Name
		caToUpdate.URL = p.DigiCertCAUpdatePayload.URL
		caToUpdate.APIToken = p.DigiCertCAUpdatePayload.APIToken
		caToUpdate.ProfileID = p.DigiCertCAUpdatePayload.ProfileID
		caToUpdate.CertificateCommonName = p.DigiCertCAUpdatePayload.CertificateCommonName
		caToUpdate.CertificateUserPrincipalNames = p.DigiCertCAUpdatePayload.CertificateUserPrincipalNames
		caToUpdate.CertificateSeatID = p.DigiCertCAUpdatePayload.CertificateSeatID

		if caToUpdate.Name != nil {
			caActivityName = *caToUpdate.Name
		} else {
			caActivityName = *oldCA.Name
		}
		activity = fleet.ActivityEditedDigiCert{Name: caActivityName}
	}
	if p.HydrantCAUpdatePayload != nil {
		if err := p.HydrantCAUpdatePayload.ValidateRelatedFields(errPrefix, *oldCA.Name); err != nil {
			return err
		}
		p.HydrantCAUpdatePayload.Preprocess()
		if err := svc.validateHydrantUpdate(p.HydrantCAUpdatePayload, errPrefix); err != nil {
			return err
		}
		caToUpdate.Type = string(fleet.CATypeHydrant)
		caToUpdate.Name = p.HydrantCAUpdatePayload.Name
		caToUpdate.URL = p.HydrantCAUpdatePayload.URL
		caToUpdate.ClientID = p.HydrantCAUpdatePayload.ClientID
		caToUpdate.ClientSecret = p.HydrantCAUpdatePayload.ClientSecret
		if caToUpdate.Name != nil {
			caActivityName = *caToUpdate.Name
		} else {
			caActivityName = *oldCA.Name
		}
		activity = fleet.ActivityEditedHydrant{Name: caActivityName}
	}
	if p.NDESSCEPProxyCAUpdatePayload != nil {
		if err := p.NDESSCEPProxyCAUpdatePayload.ValidateRelatedFields(errPrefix, *oldCA.Name); err != nil {
			return err
		}
		p.NDESSCEPProxyCAUpdatePayload.Preprocess()
		if err := svc.validateNDESSCEPProxyUpdate(ctx, p.NDESSCEPProxyCAUpdatePayload, oldCA, errPrefix); err != nil {
			return err
		}
		caToUpdate.Type = string(fleet.CATypeNDESSCEPProxy)
		caToUpdate.URL = p.NDESSCEPProxyCAUpdatePayload.URL
		caToUpdate.AdminURL = p.NDESSCEPProxyCAUpdatePayload.AdminURL
		caToUpdate.Username = p.NDESSCEPProxyCAUpdatePayload.Username
		caToUpdate.Password = p.NDESSCEPProxyCAUpdatePayload.Password
		if caToUpdate.Name != nil {
			caActivityName = *caToUpdate.Name
		} else {
			caActivityName = *oldCA.Name
		}
		activity = fleet.ActivityEditedNDESSCEPProxy{}
	}
	if p.CustomSCEPProxyCAUpdatePayload != nil {
		if err := p.CustomSCEPProxyCAUpdatePayload.ValidateRelatedFields(errPrefix, *oldCA.Name); err != nil {
			return err
		}
		p.CustomSCEPProxyCAUpdatePayload.Preprocess()
		if err := svc.validateCustomSCEPProxyUpdate(ctx, p.CustomSCEPProxyCAUpdatePayload, errPrefix); err != nil {
			return err
		}
		caToUpdate.Type = string(fleet.CATypeCustomSCEPProxy)
		caToUpdate.Name = p.CustomSCEPProxyCAUpdatePayload.Name
		caToUpdate.URL = p.CustomSCEPProxyCAUpdatePayload.URL
		caToUpdate.Challenge = p.CustomSCEPProxyCAUpdatePayload.Challenge
		if caToUpdate.Name != nil {
			caActivityName = *caToUpdate.Name
		} else {
			caActivityName = *oldCA.Name
		}
		activity = fleet.ActivityEditedCustomSCEPProxy{Name: caActivityName}

	}

	if oldCA.Type != caToUpdate.Type {
		return &fleet.BadRequestError{Message: fmt.Sprintf("%sThe certificate authority types must be the same.", errPrefix)}
	}

	if err := svc.ds.UpdateCertificateAuthorityByID(ctx, id, &caToUpdate); err != nil {
		return err
	}

	if err := svc.NewActivity(ctx, authz.UserFromContext(ctx), activity); err != nil {
		return fmt.Errorf("recording activity for edited %s certificate authority %s: %w", caToUpdate.Type, caActivityName, err)
	}

	return nil
}

func (svc *Service) validateDigicertUpdate(ctx context.Context, digicert *fleet.DigiCertCAUpdatePayload, oldCA *fleet.CertificateAuthority, errPrefix string) error {
	if digicert.Name != nil {
		if err := validateCAName(*digicert.Name, errPrefix); err != nil {
			return err
		}
	}
	if digicert.URL != nil {
		if err := validateURL(*digicert.URL, "DigiCert", errPrefix); err != nil {
			return err
		}
	}
	if digicert.APIToken != nil && *digicert.APIToken == "" {
		return &fleet.BadRequestError{
			Message: fmt.Sprintf("%sInvalid DigiCert API token. Please correct and try again.", errPrefix),
		}
	}
	if digicert.ProfileID != nil {
		if *digicert.ProfileID == "" {
			return &fleet.BadRequestError{
				Message: fmt.Sprintf("%sInvalid profile GUID. Please correct and try again.", errPrefix),
			}
		}

		// We want to generate a DigiCertCA struct with all required fields to verify the profile ID.
		// If URL or APIToken are not being updated we use the existing values from oldCA
		digicertCA := fleet.DigiCertCA{
			ProfileID: *digicert.ProfileID,
		}
		if digicert.URL != nil {
			digicertCA.URL = *digicert.URL
		} else {
			digicertCA.URL = *oldCA.URL
		}
		if digicert.APIToken != nil {
			digicertCA.APIToken = *digicert.APIToken
		} else {
			digicertCA.APIToken = *oldCA.APIToken
		}
		if err := svc.digiCertService.VerifyProfileID(ctx, digicertCA); err != nil {
			level.Error(svc.logger).Log("msg", "Failed to validate DigiCert profile GUID", "err", err)
			return &fleet.BadRequestError{Message: fmt.Sprintf("%sCould not verify DigiCert profile ID: %s. Please correct and try again.", errPrefix, err.Error())}
		}
	}
	if digicert.CertificateCommonName != nil {
		if err := validateDigicertCACN(*digicert.CertificateCommonName, errPrefix); err != nil {
			return err
		}
	}
	if digicert.CertificateUserPrincipalNames != nil {
		if err := validateDigicertUserPrincipalNames(*digicert.CertificateUserPrincipalNames, errPrefix); err != nil {
			return err
		}
	}
	if digicert.CertificateSeatID != nil {
		if err := validateDigicertSeatID(*digicert.CertificateSeatID, errPrefix); err != nil {
			return err
		}
	}
	return nil
}

func (svc *Service) validateHydrantUpdate(hydrant *fleet.HydrantCAUpdatePayload, errPrefix string) error {
	if hydrant.Name != nil {
		if err := validateCAName(*hydrant.Name, errPrefix); err != nil {
			return err
		}
	}
	if hydrant.URL != nil {
		if err := validateURL(*hydrant.URL, "Hydrant", errPrefix); err != nil {
			return err
		}
	}
	if hydrant.ClientID != nil && *hydrant.ClientID == "" {
		return &fleet.BadRequestError{
			Message: fmt.Sprintf("%sInvalid Hydrant Client ID. Please correct and try again.", errPrefix),
		}
	}
	if hydrant.ClientSecret != nil && *hydrant.ClientSecret == "" {
		return &fleet.BadRequestError{
			Message: fmt.Sprintf("%sInvalid Hydrant Client Secret. Please correct and try again.", errPrefix),
		}
	}
	return nil
}

func (svc *Service) validateNDESSCEPProxyUpdate(ctx context.Context, ndesSCEP *fleet.NDESSCEPProxyCAUpdatePayload, oldCA *fleet.CertificateAuthority, errPrefix string) error {
	// some methods in this fuction require the NDESSCEPProxyCA type so we convert the ndes update payload here

	if ndesSCEP.URL != nil {
		if err := validateURL(*ndesSCEP.URL, "NDES SCEP", errPrefix); err != nil {
			return err
		}
		if err := svc.scepConfigService.ValidateSCEPURL(ctx, *ndesSCEP.URL); err != nil {
			level.Error(svc.logger).Log("msg", "Failed to validate NDES SCEP URL", "err", err)
			return &fleet.BadRequestError{Message: fmt.Sprintf("%sInvalid SCEP URL. Please correct and try again.", errPrefix)}
		}
	}
	if ndesSCEP.AdminURL != nil {
		if *ndesSCEP.AdminURL == "" {
			return &fleet.BadRequestError{
				Message: fmt.Sprintf("%sInvalid NDES SCEP admin URL. Please correct and try again.", errPrefix),
			}
		}

		// We want to generate a NDESSCEPProxyCA struct with all required fields to verify the admin URL.
		// If URL, Username or Password are not being updated we use the existing values from oldCA
		NDESProxy := fleet.NDESSCEPProxyCA{
			AdminURL: *ndesSCEP.AdminURL,
		}
		if ndesSCEP.URL != nil {
			NDESProxy.URL = *ndesSCEP.URL
		} else {
			NDESProxy.URL = *oldCA.URL
		}
		if ndesSCEP.Username != nil {
			NDESProxy.Username = *ndesSCEP.Username
		} else {
			NDESProxy.Username = *oldCA.Username
		}
		if ndesSCEP.Password != nil {
			NDESProxy.Password = *ndesSCEP.Password
		} else {
			NDESProxy.Password = *oldCA.Password
		}

		if err := svc.scepConfigService.ValidateNDESSCEPAdminURL(ctx, NDESProxy); err != nil {
			level.Error(svc.logger).Log("msg", "Failed to validate NDES SCEP admin URL", "err", err)
			switch {
			case errors.As(err, &NDESPasswordCacheFullError{}):
				return &fleet.BadRequestError{Message: fmt.Sprintf("%sThe NDES password cache is full. Please increase the number of cached passwords in NDES and try again.", errPrefix)}
			case errors.As(err, &NDESInsufficientPermissionsError{}):
				return &fleet.BadRequestError{Message: fmt.Sprintf("%sInsufficient permissions for NDES SCEP admin URL. Please correct and try again.", errPrefix)}
			default:
				return &fleet.BadRequestError{Message: fmt.Sprintf("%sInvalid NDES SCEP admin URL or credentials. Please correct and try again.", errPrefix)}
			}
		}
	}
	return nil
}

func (svc *Service) validateCustomSCEPProxyUpdate(ctx context.Context, customSCEP *fleet.CustomSCEPProxyCAUpdatePayload, errPrefix string) error {
	if customSCEP.Name != nil {
		if err := validateCAName(*customSCEP.Name, errPrefix); err != nil {
			return err
		}
	}
	if customSCEP.URL != nil {
		if err := validateURL(*customSCEP.URL, "SCEP", errPrefix); err != nil {
			return err
		}
		if err := svc.scepConfigService.ValidateSCEPURL(ctx, *customSCEP.URL); err != nil {
			level.Error(svc.logger).Log("msg", "Failed to validate custom SCEP URL", "err", err)
			return &fleet.BadRequestError{Message: fmt.Sprintf("%sInvalid SCEP URL. Please correct and try again.", errPrefix)}
		}
	}
	if customSCEP.Challenge != nil && *customSCEP.Challenge == "" {
		return &fleet.BadRequestError{
			Message: fmt.Sprintf("%sCustom SCEP Proxy challenge cannot be empty", errPrefix),
		}
	}

	return nil
}
