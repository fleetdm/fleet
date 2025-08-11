package service

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/variables"
)

func (svc *Service) NewCertificateAuthority(ctx context.Context, p fleet.CertificateAuthorityPayload) (*fleet.CertificateAuthority, error) {
	if err := svc.authz.Authorize(ctx, &fleet.CertificateAuthority{}, fleet.ActionWrite); err != nil {
		return nil, err
	}

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
		return nil, &fleet.BadRequestError{Message: "One certificate authority must be specified"}
	}
	if casToCreate > 1 {
		return nil, &fleet.BadRequestError{Message: "Only one certificate authority can be created at a time"}
	}

	if len(svc.config.Server.PrivateKey) == 0 {
		return nil, &fleet.BadRequestError{Message: "Couldn't add certificate authority. Private key must be configured. Learn more: https://fleetdm.com/learn-more-about/fleet-server-private-key"}
	}

	caToCreate := &fleet.CertificateAuthority{}

	var activity fleet.ActivityDetails

	if p.DigiCert != nil {
		p.DigiCert.Name = fleet.Preprocess(p.DigiCert.Name)
		p.DigiCert.URL = fleet.Preprocess(p.DigiCert.URL)
		p.DigiCert.ProfileID = fleet.Preprocess(p.DigiCert.ProfileID)
		if err := svc.validateDigicert(ctx, p.DigiCert); err != nil {
			return nil, err
		}
		caToCreate.Type = string(fleet.CATypeDigiCert)
		caToCreate.Name = p.DigiCert.Name
		caToCreate.URL = p.DigiCert.URL
		caToCreate.APIToken = &p.DigiCert.APIToken
		caToCreate.ProfileID = ptr.String(p.DigiCert.ProfileID)
		caToCreate.CertificateCommonName = &p.DigiCert.CertificateCommonName
		caToCreate.CertificateUserPrincipalNames = p.DigiCert.CertificateUserPrincipalNames
		caToCreate.CertificateSeatID = &p.DigiCert.CertificateSeatID
		activity = fleet.ActivityAddedDigiCert{Name: p.DigiCert.Name}
	}

	if p.Hydrant != nil {
		p.Hydrant.Name = fleet.Preprocess(p.Hydrant.Name)
		p.Hydrant.URL = fleet.Preprocess(p.Hydrant.URL)
		if err := validateHydrant(p.Hydrant); err != nil {
			return nil, err
		}

		caToCreate.Type = string(fleet.CATypeHydrant)
		caToCreate.Name = p.Hydrant.Name
		caToCreate.URL = p.Hydrant.URL
		caToCreate.ClientID = &p.Hydrant.ClientID
		caToCreate.ClientSecret = &p.Hydrant.ClientSecret
		activity = fleet.ActivityAddedHydrant{}
	}

	if p.NDESSCEPProxy != nil {
		p.NDESSCEPProxy.URL = fleet.Preprocess(p.NDESSCEPProxy.URL)
		p.NDESSCEPProxy.AdminURL = fleet.Preprocess(p.NDESSCEPProxy.AdminURL)
		p.NDESSCEPProxy.Username = fleet.Preprocess(p.NDESSCEPProxy.Username)
		if err := svc.validateNDESSCEPProxy(ctx, p.NDESSCEPProxy); err != nil {
			return nil, err
		}

		// TODO check for multiple NDES
		caToCreate.Name = "DEFAULT_NDES_CA"
		caToCreate.Type = string(fleet.CATypeNDESSCEPProxy)
		caToCreate.URL = p.NDESSCEPProxy.URL
		caToCreate.AdminURL = ptr.String(p.NDESSCEPProxy.AdminURL)
		caToCreate.Username = ptr.String(p.NDESSCEPProxy.Username)
		caToCreate.Password = &p.NDESSCEPProxy.Password
		activity = fleet.ActivityAddedNDESSCEPProxy{}
	}

	if p.CustomSCEPProxy != nil {
		p.CustomSCEPProxy.Name = fleet.Preprocess(p.CustomSCEPProxy.Name)
		p.CustomSCEPProxy.URL = fleet.Preprocess(p.CustomSCEPProxy.URL)

		if err := svc.validateCustomSCEPProxy(ctx, p.CustomSCEPProxy); err != nil {
			return nil, err
		}

		caToCreate.Type = string(fleet.CATypeCustomSCEPProxy)
		caToCreate.Name = p.CustomSCEPProxy.Name
		caToCreate.URL = p.CustomSCEPProxy.URL
		caToCreate.Challenge = &p.CustomSCEPProxy.Challenge
		activity = fleet.ActivityAddedCustomSCEPProxy{Name: p.CustomSCEPProxy.Name}
	}

	createdCA, err := svc.ds.NewCertificateAuthority(ctx, caToCreate)
	if err != nil {
		if strings.Contains(err.Error(), "idx_ca_type_name") {
			// TODO Make sure this works
			return nil, &fleet.BadRequestError{Message: fmt.Sprintf("Couldn’t add certificate authority. \"%s\" name is already used by another certificate authority. Please choose a different name and try again.", caToCreate.Name)}
		}
	}

	if err := svc.NewActivity(ctx, authz.UserFromContext(ctx), activity); err != nil {
		return nil, fmt.Errorf("recording activity for new %s certificate authority %s: %w", caToCreate.Type, caToCreate.Name, err)
	}

	return createdCA, nil
}

func (svc *Service) validateDigicert(ctx context.Context, digicertCA *fleet.DigiCertIntegration) error {
	if err := validateURL(digicertCA.URL); err != nil {
		return err
	}
	if digicertCA.APIToken == "" {
		return fleet.NewInvalidArgumentError("api_token", "Invalid API token. Please correct and try again.")
	}
	if digicertCA.ProfileID == "" {
		return fleet.NewInvalidArgumentError("profile_id", "Invalid profile GUID. Please correct and try again.")
	}
	if err := validateCAName(digicertCA.Name); err != nil {
		return err
	}
	if err := validateDigicertCACN(digicertCA.CertificateCommonName); err != nil {
		return err
	}
	if err := validateDigicertUserPrincipalNames(digicertCA.CertificateUserPrincipalNames); err != nil {
		return err
	}
	if err := validateDigicertSeatID(digicertCA.CertificateSeatID); err != nil {
		return err
	}

	if err := svc.digiCertService.VerifyProfileID(ctx, *digicertCA); err != nil {
		return err
	}
	return nil
}

func validateCAName(name string) error {
	if name == "NDES" {
		return fleet.NewInvalidArgumentError("name", "CA name cannot be NDES")
	}
	if len(name) == 0 {
		return fleet.NewInvalidArgumentError("name", "CA name cannot be empty")
	}
	if len(name) > 255 {
		return fleet.NewInvalidArgumentError("name", "CA name cannot be longer than 255 characters")
	}
	if !isAlphanumeric(name) {
		return fleet.NewInvalidArgumentError("name", "Invalid characters in the \"name\" field. Only letters, "+
			"numbers and underscores allowed.")
	}
	return nil
}

func validateDigicertCACN(cn string) error {
	if len(strings.TrimSpace(cn)) == 0 {
		return fleet.NewInvalidArgumentError("certificate_common_name", "CA Common Name (CN) cannot be empty")
	}
	fleetVars := variables.Find(cn)
	for fleetVar := range fleetVars {
		switch fleetVar {
		case string(fleet.FleetVarHostEndUserEmailIDP), string(fleet.FleetVarHostHardwareSerial):
			// ok
		default:
			return fleet.NewInvalidArgumentError("certificate_common_name", "FLEET_VAR_"+fleetVar+" is not allowed in CA Common Name (CN)")
		}
	}
	return nil
}

var alphanumeric = regexp.MustCompile(`^\w+$`)

func isAlphanumeric(s string) bool {
	return alphanumeric.MatchString(s)
}

func validateDigicertSeatID(seatID string) error {
	if len(strings.TrimSpace(seatID)) == 0 {
		return fleet.NewInvalidArgumentError("certificate_seat_id", "CA Seat ID cannot be empty")
	}
	fleetVars := variables.Find(seatID)
	for fleetVar := range fleetVars {
		switch fleetVar {
		case string(fleet.FleetVarHostEndUserEmailIDP), string(fleet.FleetVarHostHardwareSerial):
			// ok
		default:
			return fleet.NewInvalidArgumentError("certificate_seat_id", "FLEET_VAR_"+fleetVar+" is not allowed in DigiCert Seat ID")
		}
	}
	return nil
}

func validateDigicertUserPrincipalNames(userPrincipalNames []string) error {
	if len(userPrincipalNames) == 0 {
		return nil
	}
	if len(userPrincipalNames) > 1 {
		return fleet.NewInvalidArgumentError("certificate_user_principal_names",
			"Currently, only one item can be added to certificate_user_principal_names.")
	}
	if len(strings.TrimSpace(userPrincipalNames[0])) == 0 {
		return fleet.NewInvalidArgumentError("certificate_user_principal_names",
			"DigiCert certificate_user_principal_name cannot be empty if specified")
	}
	fleetVars := variables.Find(userPrincipalNames[0])
	for fleetVar := range fleetVars {
		switch fleetVar {
		case string(fleet.FleetVarHostEndUserEmailIDP), string(fleet.FleetVarHostHardwareSerial):
			// ok
		default:
			return fleet.NewInvalidArgumentError("certificate_user_principal_names",
				"FLEET_VAR_"+fleetVar+" is not allowed in CA User Principal Name")
		}
	}
	return nil
}

func validateHydrant(hydrantCA *fleet.HydrantCA) error {
	validateCAName(hydrantCA.Name)
	if err := validateURL(hydrantCA.URL); err != nil {
		return err
	}
	// TODO HCA Validate Hydrant Parameters
	return nil
}

func validateURL(caURL string) error {
	if u, err := url.ParseRequestURI(caURL); err != nil {
		return fleet.NewInvalidArgumentError("url",
			err.Error())
	} else if u.Scheme != "https" && u.Scheme != "http" {
		return fleet.NewInvalidArgumentError("url", "URL scheme must be https or http")
	}
	return nil
}

func (svc *Service) validateNDESSCEPProxy(ctx context.Context, ndesSCEP *fleet.NDESSCEPProxyIntegration) error {
	if err := validateURL(ndesSCEP.URL); err != nil {
		return err
	}
	if err := svc.scepConfigService.ValidateSCEPURL(ctx, ndesSCEP.URL); err != nil {
		return fleet.NewInvalidArgumentError("url", err.Error())
	}
	if err := svc.scepConfigService.ValidateNDESSCEPAdminURL(ctx, *ndesSCEP); err != nil {
		return fleet.NewInvalidArgumentError("admin_url", err.Error())
	}

	if len(strings.TrimSpace(ndesSCEP.AdminURL)) == 0 {
		return fleet.NewInvalidArgumentError("admin_url", "NDES SCEP Proxy Admin URL cannot be empty")
	}
	if len(strings.TrimSpace(ndesSCEP.Username)) == 0 {
		return fleet.NewInvalidArgumentError("username", "NDES SCEP Proxy username cannot be empty")
	}
	if ndesSCEP.Password == "" {
		return fleet.NewInvalidArgumentError("password", "NDES SCEP Proxy password cannot be empty")
	}
	return nil
}

func (svc *Service) validateCustomSCEPProxy(ctx context.Context, customSCEP *fleet.CustomSCEPProxyIntegration) error {
	if err := validateURL(customSCEP.URL); err != nil {
		return err
	}
	if customSCEP.Challenge == "" {
		return fleet.NewInvalidArgumentError("challenge", "Custom SCEP Proxy challenge cannot be empty")
	}
	if err := svc.scepConfigService.ValidateSCEPURL(ctx, customSCEP.URL); err != nil {
		return fleet.NewInvalidArgumentError("url", err.Error())
	}
	return nil
}
