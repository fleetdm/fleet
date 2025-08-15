package service

import (
	"context"
	"crypto/x509"
	"encoding/asn1"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
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
	errPrefix := "Couldn't add certificate authority. "
	if casToCreate == 0 {
		return nil, &fleet.BadRequestError{Message: errPrefix + "A certificate authority must be specified"}
	}
	if casToCreate > 1 {
		return nil, &fleet.BadRequestError{Message: errPrefix + "Only one certificate authority can be created at a time"}
	}

	if len(svc.config.Server.PrivateKey) == 0 {
		return nil, &fleet.BadRequestError{Message: errPrefix + "Private key must be configured. Learn more: https://fleetdm.com/learn-more-about/fleet-server-private-key"}
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
		caToCreate.Name = p.DigiCert.Name
		caToCreate.URL = p.DigiCert.URL
		caToCreate.APIToken = &p.DigiCert.APIToken
		caToCreate.ProfileID = ptr.String(p.DigiCert.ProfileID)
		caToCreate.CertificateCommonName = &p.DigiCert.CertificateCommonName
		caToCreate.CertificateUserPrincipalNames = p.DigiCert.CertificateUserPrincipalNames
		caToCreate.CertificateSeatID = &p.DigiCert.CertificateSeatID
		caDisplayType = "DigiCert"
		activity = fleet.ActivityAddedDigiCert{Name: p.DigiCert.Name}
	}

	if p.Hydrant != nil {
		p.Hydrant.Name = fleet.Preprocess(p.Hydrant.Name)
		p.Hydrant.URL = fleet.Preprocess(p.Hydrant.URL)
		if err := validateHydrant(p.Hydrant, errPrefix); err != nil {
			return nil, err
		}

		caToCreate.Type = string(fleet.CATypeHydrant)
		caToCreate.Name = p.Hydrant.Name
		caToCreate.URL = p.Hydrant.URL
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

		caToCreate.Name = "NDES"
		caToCreate.Type = string(fleet.CATypeNDESSCEPProxy)
		caToCreate.URL = p.NDESSCEPProxy.URL
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
		caToCreate.Name = p.CustomSCEPProxy.Name
		caToCreate.URL = p.CustomSCEPProxy.URL
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
			return nil, &fleet.BadRequestError{Message: fmt.Sprintf("%s\"%s\" name is already used by another %s certificate authority. Please choose a different name and try again.", errPrefix, caToCreate.Name, caDisplayType)}
		}
		return nil, err
	}

	if err := svc.NewActivity(ctx, authz.UserFromContext(ctx), activity); err != nil {
		return nil, fmt.Errorf("recording activity for new %s certificate authority %s: %w", caToCreate.Type, caToCreate.Name, err)
	}

	return createdCA, nil
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

func validateHydrant(hydrantCA *fleet.HydrantCA, errPrefix string) error {
	if err := validateCAName(hydrantCA.Name, errPrefix); err != nil {
		return err
	}
	if err := validateURL(hydrantCA.URL, "Hydrant", errPrefix); err != nil {
		return err
	}
	// TODO HCA Validate Hydrant Parameters by actually connecting to Hydrant(ideally)
	if hydrantCA.ClientID == "" {
		return fleet.NewInvalidArgumentError("client_id", fmt.Sprintf("%sInvalid Hydrant Client ID. Please correct and try again.", errPrefix))
	}
	if hydrantCA.ClientSecret == "" {
		return fleet.NewInvalidArgumentError("client_secret", fmt.Sprintf("%sInvalid Hydrant Client Secret. Please correct and try again.", errPrefix))
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

func (svc *Service) RequestCertificate(ctx context.Context, p fleet.RequestCertificatePayload) (string, error) {
	if err := svc.authz.Authorize(ctx, &fleet.RequestCertificatePayload{}, fleet.ActionWrite); err != nil {
		return "", err
	}
	ca, err := svc.ds.GetCertificateAuthorityByID(ctx, p.ID, true)
	if err != nil {
		return "", err
	}
	if ca.Type != string(fleet.CATypeHydrant) {
		return "", &fleet.BadRequestError{Message: "Only Hydrant certificate authorities support requesting certificates via Fleet."}
	}

	if p.IDPClientID == nil || p.IDPToken == nil || p.IDPOauthURL == nil {
		return "", &fleet.BadRequestError{Message: "IDP Client ID, Token, and OAuth URL must be provided for requesting a certificate."}
	}
	httpClient := fleethttp.NewClient(fleethttp.WithTimeout(20 * time.Second))
	introspectionRequest := url.Values{
		"client_id": []string{*p.IDPClientID},
		"token":     []string{*p.IDPToken},
	}
	introspectionBody := introspectionRequest.Encode()
	req, err := http.NewRequestWithContext(ctx, "POST", *p.IDPOauthURL, strings.NewReader(introspectionBody))
	if err != nil {
		return "", &fleet.BadRequestError{Message: fmt.Sprintf("Failed to create introspection request: %s", err.Error())}
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", &fleet.BadRequestError{Message: fmt.Sprintf("Failed to introspect IDP token: %s", err.Error())}
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", &fleet.BadRequestError{Message: fmt.Sprintf("IDP token introspection failed with status code %d", resp.StatusCode)}
	}

	csr := strings.ReplaceAll(p.CSR, "-----BEGIN CERTIFICATE REQUEST-----", "")
	csr = strings.ReplaceAll(csr, "-----END CERTIFICATE REQUEST-----", "")
	csr = strings.ReplaceAll(csr, "\\n", "")

	certificate, err := svc.hydrantService.GetCertificate(ctx, fleet.HydrantCA{
		Name:         ca.Name,
		URL:          ca.URL,
		ClientID:     *p.IDPClientID,
		ClientSecret: *p.IDPToken,
	}, csr)
	if err != nil {
		level.Error(svc.logger).Log("msg", "Failed to get Hydrant certificate", "error", err)
		return "", &fleet.BadRequestError{Message: fmt.Sprintf("Failed to get certificate from Hydrant: %s", err.Error())}
	}
	// TODO Do we need to convert this?
	return "-----BEGIN CERTIFICATE-----\n" + string(certificate.Certificate) + "\n-----END CERTIFICATE-----", nil
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

func extractCSRUPN(csr *x509.CertificateRequest) (*string, error) {
	sanOID := asn1.ObjectIdentifier{2, 5, 29, 17}
	upnOID := asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 311, 20, 2, 3}
	for _, ext := range csr.Extensions {
		if ext.Id.Equal(sanOID) {
			nameValues := []asn1.RawValue{}
			if _, err := asn1.Unmarshal(ext.Value, &nameValues); err != nil {
				// TODO Wrap err logger.Log("msg", "Failed to unmarshal SAN extension", "error", err)
				return nil, err
			}
			for _, names := range nameValues {
				// We are looking for the othernames in the SAN extension(tag 0)
				if names.Tag == 0 {
					var oid asn1.ObjectIdentifier
					var rawValue asn1.RawValue
					var err error
					remainingBytes := names.Bytes
					for len(remainingBytes) > 0 {
						// Parse the sequence of OIDs and Values looking for the UPN OID
						remainingBytes, err = asn1.Unmarshal(names.Bytes, &oid)
						if err != nil {
							// TODO Wrap err logger.Log("msg", "Failed to unmarshal othername OID", "error", err)
							return nil, err
						}
						remainingBytes, err = asn1.Unmarshal(remainingBytes, &rawValue)
						if err != nil {
							// TODO Wrap err logger.Log("msg", "Failed to unmarshal othername value", "error", err)
							return nil, err
						}
						if oid.Equal(upnOID) {
							// UPN found, now to unmarshal its value
							var upn asn1.RawValue

							if _, err := asn1.Unmarshal(rawValue.Bytes, &upn); err != nil {
								// TODO Wrap err logger.Log("msg", "Failed to unmarshal UPN value", "error", err)
								return nil, err
							}
							upnString := string(upn.Bytes)
							return &upnString, nil
						}
					}
				}
			}
		}
	}
	return nil, nil // No UPN found
}
