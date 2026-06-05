package service

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/fleetdm/fleet/v4/ee/server/service/scep"
	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/variables"
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
		p.DigiCert.Preprocess()
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
		p.Hydrant.Preprocess()

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

	if p.CustomESTProxy != nil {
		p.CustomESTProxy.Preprocess()

		if err := svc.validateEST(ctx, p.CustomESTProxy, errPrefix); err != nil {
			return nil, err
		}

		caToCreate.Type = string(fleet.CATypeCustomESTProxy)
		caToCreate.Name = &p.CustomESTProxy.Name
		caToCreate.URL = &p.CustomESTProxy.URL
		caToCreate.Username = &p.CustomESTProxy.Username
		caToCreate.Password = &p.CustomESTProxy.Password
		caDisplayType = "custom EST"
		activity = fleet.ActivityAddedCustomESTProxy{Name: p.CustomESTProxy.Name}
	}

	if p.NDESSCEPProxy != nil {
		p.NDESSCEPProxy.Preprocess()

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
		p.CustomSCEPProxy.Preprocess()

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

	if p.Smallstep != nil {
		p.Smallstep.Preprocess()

		if err := svc.validateSmallstepSCEPProxy(ctx, p.Smallstep, errPrefix); err != nil {
			return nil, err
		}

		caToCreate.Type = string(fleet.CATypeSmallstep)
		caToCreate.Name = &p.Smallstep.Name
		caToCreate.URL = &p.Smallstep.URL
		caToCreate.ChallengeURL = &p.Smallstep.ChallengeURL
		caToCreate.Username = &p.Smallstep.Username
		caToCreate.Password = &p.Smallstep.Password
		caDisplayType = "Smallstep"
		activity = fleet.ActivityAddedSmallstep{Name: p.Smallstep.Name}
	}

	if p.EJBCA != nil {
		p.EJBCA.Preprocess()

		// validateEJBCA decodes the uploaded P12 and populates
		// p.EJBCA.ClientCertPEM / ClientKeyPEM, then probes the EJBCA REST API
		// to confirm reachability and mTLS auth.
		if err := svc.validateEJBCA(ctx, p.EJBCA, errPrefix); err != nil {
			return nil, err
		}

		caToCreate.Type = string(fleet.CATypeEJBCA)
		caToCreate.Name = &p.EJBCA.Name
		caToCreate.URL = &p.EJBCA.URL
		caToCreate.ClientCertPEM = &p.EJBCA.ClientCertPEM
		caToCreate.ClientKeyPEM = &p.EJBCA.ClientKeyPEM
		if p.EJBCA.TrustCABundlePEM != "" {
			caToCreate.TrustCABundlePEM = &p.EJBCA.TrustCABundlePEM
		}
		caToCreate.EJBCACAName = &p.EJBCA.CertificateAuthorityNameEJBCA
		caToCreate.EJBCACertificateProfileName = &p.EJBCA.CertificateProfileName
		caToCreate.EJBCAEndEntityProfileName = &p.EJBCA.EndEntityProfileName
		caToCreate.EJBCAUsernameTemplate = &p.EJBCA.UsernameTemplate
		if len(p.EJBCA.CertificateUserPrincipalNames) > 0 {
			caToCreate.CertificateUserPrincipalNames = &p.EJBCA.CertificateUserPrincipalNames
		}
		caDisplayType = "EJBCA"
		activity = fleet.ActivityAddedEJBCA{Name: p.EJBCA.Name}
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
	if p.Smallstep != nil {
		casToCreate++
	}
	if p.CustomESTProxy != nil {
		casToCreate++
	}
	if p.EJBCA != nil {
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
	if digicertCA.APIToken == "" || digicertCA.APIToken == fleet.MaskedPassword {
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
		svc.logger.ErrorContext(ctx, "Failed to validate DigiCert profile GUID", "err", err)
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
	for _, fleetVar := range fleetVars {
		switch fleetVar {
		case string(fleet.FleetVarHostEndUserEmailIDP), string(fleet.FleetVarHostHardwareSerial), string(fleet.FleetVarHostPlatform):
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
	for _, fleetVar := range fleetVars {
		switch fleetVar {
		case string(fleet.FleetVarHostEndUserEmailIDP), string(fleet.FleetVarHostHardwareSerial), string(fleet.FleetVarHostPlatform):
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
	for _, fleetVar := range fleetVars {
		switch fleetVar {
		case string(fleet.FleetVarHostEndUserEmailIDP), string(fleet.FleetVarHostHardwareSerial), string(fleet.FleetVarHostPlatform):
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
	if hydrantCA.ClientSecret == "" || hydrantCA.ClientSecret == fleet.MaskedPassword {
		return fleet.NewInvalidArgumentError("client_secret", fmt.Sprintf("%sInvalid Hydrant Client Secret. Please correct and try again.", errPrefix))
	}
	if err := svc.estService.ValidateESTURL(ctx, fleet.ESTProxyCA{
		ID:       hydrantCA.ID,
		Name:     hydrantCA.Name,
		URL:      hydrantCA.URL,
		Username: hydrantCA.ClientID,
		Password: hydrantCA.ClientSecret,
	}); err != nil {
		return fleet.NewInvalidArgumentError("url", fmt.Sprintf("%sInvalid Hydrant URL. Please correct and try again.", errPrefix))
	}
	return nil
}

func (svc *Service) validateEST(ctx context.Context, estProxyCA *fleet.ESTProxyCA, errPrefix string) error {
	if err := validateCAName(estProxyCA.Name, errPrefix); err != nil {
		return err
	}
	if err := validateURL(estProxyCA.URL, "EST", errPrefix); err != nil {
		return err
	}
	if estProxyCA.Username == "" {
		return fleet.NewInvalidArgumentError("username", fmt.Sprintf("%sInvalid EST Username. Please correct and try again.", errPrefix))
	}
	if estProxyCA.Password == "" || estProxyCA.Password == fleet.MaskedPassword {
		return fleet.NewInvalidArgumentError("password", fmt.Sprintf("%sInvalid EST Password. Please correct and try again.", errPrefix))
	}
	if err := svc.estService.ValidateESTURL(ctx, *estProxyCA); err != nil {
		return fleet.NewInvalidArgumentError("url", fmt.Sprintf("%sInvalid EST URL. Please correct and try again.", errPrefix))
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
		svc.logger.ErrorContext(ctx, "Failed to validate NDES SCEP URL", "err", err)
		return &fleet.BadRequestError{Message: fmt.Sprintf("%sInvalid SCEP URL. Please correct and try again.", errPrefix)}
	}
	if err := svc.scepConfigService.ValidateNDESSCEPAdminURL(ctx, *ndesSCEP); err != nil {
		svc.logger.ErrorContext(ctx, "Failed to validate NDES SCEP admin URL", "err", err)
		switch {
		case errors.As(err, &scep.NDESPasswordCacheFullError{}):
			return &fleet.BadRequestError{Message: fmt.Sprintf("%sThe NDES password cache is full. Please increase the number of cached passwords in NDES and try again.", errPrefix)}
		case errors.As(err, &scep.NDESInsufficientPermissionsError{}):
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
	if customSCEP.Challenge == "" || customSCEP.Challenge == fleet.MaskedPassword {
		return fleet.NewInvalidArgumentError("challenge", fmt.Sprintf("%sCustom SCEP Proxy challenge cannot be empty", errPrefix))
	}
	if err := svc.scepConfigService.ValidateSCEPURL(ctx, customSCEP.URL); err != nil {
		svc.logger.ErrorContext(ctx, "Failed to validate custom SCEP URL", "err", err)
		return &fleet.BadRequestError{Message: fmt.Sprintf("%sInvalid SCEP URL. Please correct and try again.", errPrefix)}
	}
	return nil
}

func (svc *Service) validateSmallstepSCEPProxy(ctx context.Context, smallstepSCEP *fleet.SmallstepSCEPProxyCA, errPrefix string) error {
	if err := validateCAName(smallstepSCEP.Name, errPrefix); err != nil {
		return err
	}
	if err := validateURL(smallstepSCEP.URL, "Smallstep SCEP", errPrefix); err != nil {
		return err
	}
	if smallstepSCEP.Username == "" {
		return fleet.NewInvalidArgumentError("username", fmt.Sprintf("%sSmallstep username cannot be empty", errPrefix))
	}
	if smallstepSCEP.Password == "" || smallstepSCEP.Password == fleet.MaskedPassword {
		return fleet.NewInvalidArgumentError("password", fmt.Sprintf("%sSmallstep password cannot be empty", errPrefix))
	}
	if err := svc.scepConfigService.ValidateSCEPURL(ctx, smallstepSCEP.URL); err != nil {
		svc.logger.ErrorContext(ctx, "Failed to validate Smallstep SCEP URL", "err", err)
		return &fleet.BadRequestError{Message: fmt.Sprintf("%sInvalid SCEP URL. Please correct and try again.", errPrefix)}
	}
	if err := svc.scepConfigService.ValidateSmallstepChallengeURL(ctx, *smallstepSCEP); err != nil {
		svc.logger.ErrorContext(ctx, "Failed to validate Smallstep SCEP admin URL", "err", err)
		return &fleet.BadRequestError{Message: fmt.Sprintf("%sInvalid challenge URL or credentials. Please correct and try again.", errPrefix)}
	}
	return nil
}

// validateEJBCA verifies the EJBCA CA payload and populates ClientCertPEM /
// ClientKeyPEM by decoding the supplied PKCS#12 once. The P12 bytes and
// password are not stored — only the extracted PEM cert and (encrypted) PEM
// key are persisted. Finally, probes the EJBCA REST API to confirm
// connectivity and mTLS authentication.
func (svc *Service) validateEJBCA(ctx context.Context, ejbcaCA *fleet.EJBCACA, errPrefix string) error {
	if err := validateCAName(ejbcaCA.Name, errPrefix); err != nil {
		return err
	}
	if err := validateURL(ejbcaCA.URL, "EJBCA", errPrefix); err != nil {
		return err
	}
	if len(ejbcaCA.ClientP12) == 0 {
		return fleet.NewInvalidArgumentError("client_p12", fmt.Sprintf("%sA PKCS#12 client certificate is required for EJBCA.", errPrefix))
	}
	if ejbcaCA.ClientP12Password == "" || ejbcaCA.ClientP12Password == fleet.MaskedPassword {
		return fleet.NewInvalidArgumentError("client_p12_password", fmt.Sprintf("%sA password for the PKCS#12 client certificate is required.", errPrefix))
	}

	certPEM, keyPEM, err := decodeEJBCAClientP12(ejbcaCA.ClientP12, ejbcaCA.ClientP12Password)
	if err != nil {
		return fleet.NewInvalidArgumentError("client_p12", fmt.Sprintf("%sCould not decode PKCS#12 client certificate: %s", errPrefix, err.Error()))
	}
	ejbcaCA.ClientCertPEM = certPEM
	ejbcaCA.ClientKeyPEM = keyPEM
	// Discard the upload-only material so it doesn't accidentally get
	// persisted by a downstream caller that copies the whole struct.
	ejbcaCA.ClientP12 = nil
	ejbcaCA.ClientP12Password = ""

	if ejbcaCA.TrustCABundlePEM != "" {
		if !x509.NewCertPool().AppendCertsFromPEM([]byte(ejbcaCA.TrustCABundlePEM)) {
			return fleet.NewInvalidArgumentError("trust_ca_bundle", fmt.Sprintf("%strust_ca_bundle did not contain any usable certificates.", errPrefix))
		}
	}

	if strings.TrimSpace(ejbcaCA.CertificateAuthorityNameEJBCA) == "" {
		return fleet.NewInvalidArgumentError("certificate_authority_name_ejbca", fmt.Sprintf("%sThe EJBCA Certificate Authority name is required.", errPrefix))
	}
	if strings.TrimSpace(ejbcaCA.CertificateProfileName) == "" {
		return fleet.NewInvalidArgumentError("certificate_profile_name", fmt.Sprintf("%sThe EJBCA Certificate Profile name is required.", errPrefix))
	}
	if strings.TrimSpace(ejbcaCA.EndEntityProfileName) == "" {
		return fleet.NewInvalidArgumentError("end_entity_profile_name", fmt.Sprintf("%sThe EJBCA End Entity Profile name is required.", errPrefix))
	}
	if strings.TrimSpace(ejbcaCA.UsernameTemplate) == "" {
		return fleet.NewInvalidArgumentError("username_template", fmt.Sprintf("%sA username template is required for EJBCA.", errPrefix))
	}
	if err := validateEJBCAFleetVarUsage(ejbcaCA.UsernameTemplate, "username_template", errPrefix); err != nil {
		return err
	}
	for _, upn := range ejbcaCA.CertificateUserPrincipalNames {
		if strings.TrimSpace(upn) == "" {
			return fleet.NewInvalidArgumentError("certificate_user_principal_names", fmt.Sprintf("%sUPN entries cannot be empty.", errPrefix))
		}
		if err := validateEJBCAFleetVarUsage(upn, "certificate_user_principal_names", errPrefix); err != nil {
			return err
		}
	}

	if err := svc.ejbcaService.VerifyConnection(ctx, *ejbcaCA); err != nil {
		svc.logger.ErrorContext(ctx, "Failed to verify EJBCA connection", "err", err)
		return &fleet.BadRequestError{Message: fmt.Sprintf("%sCould not verify EJBCA connection: %s. Please correct and try again.", errPrefix, err.Error())}
	}
	return nil
}

// decodeEJBCAClientP12 decodes the uploaded PKCS#12 bundle once with the
// supplied password and returns the PEM-encoded certificate (with any
// intermediate chain) and private key. The password is used once here and
// never persisted.
//
// POC NOTE — DO NOT SHIP THIS SUBPROCESS APPROACH TO PRODUCTION.
// Both Go PKCS#12 libraries (software.sslmate.com/src/go-pkcs12 and
// golang.org/x/crypto/pkcs12) require strict DER and reject BER-encoded
// input. EJBCA's RA Web (Java/BouncyCastle) emits BER P12s with
// indefinite-length sequences. To unblock the POC we shell out to the
// `openssl` binary, which handles BER. This adds a runtime binary
// dependency the Fleet server doesn't otherwise have and broadens the
// trust surface (subprocess hardening, PATH lookup, openssl version
// drift).
//
// Production (fleet#30986) should replace this with an in-process
// BER→DER normalizer in pure Go before parsing with one of the existing
// Go PKCS#12 libraries. See openspec/changes/add-ejbca-rest-ca-poc/
// research.md "Open follow-ups" for the design notes.
//
// The password is written into the child process over an extra file
// descriptor (consumed via openssl's `-passin fd:3`) rather than the
// command line so it doesn't appear in `ps` listings on the Fleet host.
func decodeEJBCAClientP12(p12Data []byte, password string) (certPEM, keyPEM string, err error) {
	passR, passW, err := os.Pipe()
	if err != nil {
		return "", "", fmt.Errorf("creating password pipe: %v", err)
	}
	defer passR.Close()

	// Write the password to the pipe in a goroutine, then close so openssl
	// sees EOF after the single read.
	go func() {
		defer passW.Close()
		_, _ = passW.Write([]byte(password))
	}()

	cmd := exec.Command("openssl", "pkcs12", "-passin", "fd:3", "-nodes")
	cmd.ExtraFiles = []*os.File{passR} // child sees this as fd 3
	cmd.Stdin = bytes.NewReader(p12Data)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if runErr := cmd.Run(); runErr != nil {
		// openssl's stderr is user-actionable for common failures
		// (wrong password, corrupt P12).
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = runErr.Error()
		}
		return "", "", fmt.Errorf("openssl failed to decode PKCS#12: %s", msg)
	}

	var certBuf, keyBuf strings.Builder
	rest := stdout.Bytes()
	for {
		var block *pem.Block
		block, rest = pem.Decode(rest)
		if block == nil {
			break
		}
		// Drop openssl's "Bag Attributes" comment headers so the resulting
		// PEM is directly consumable by tls.X509KeyPair.
		block.Headers = nil
		switch block.Type {
		case "CERTIFICATE":
			if err := pem.Encode(&certBuf, block); err != nil {
				return "", "", fmt.Errorf("encoding certificate PEM block: %v", err)
			}
		case "PRIVATE KEY", "RSA PRIVATE KEY", "EC PRIVATE KEY":
			if err := pem.Encode(&keyBuf, block); err != nil {
				return "", "", fmt.Errorf("encoding private key PEM block: %v", err)
			}
		}
	}

	if certBuf.Len() == 0 {
		return "", "", errors.New("PKCS#12 bundle did not contain a certificate")
	}
	if keyBuf.Len() == 0 {
		return "", "", errors.New("PKCS#12 bundle did not contain a private key")
	}

	return certBuf.String(), keyBuf.String(), nil
}

// validateEJBCAFleetVarUsage enforces the same Fleet-variable allow-list used
// by DigiCert for templated fields.
func validateEJBCAFleetVarUsage(value, fieldName, errPrefix string) error {
	for _, v := range variables.Find(value) {
		switch v {
		case string(fleet.FleetVarHostEndUserEmailIDP),
			string(fleet.FleetVarHostHardwareSerial),
			string(fleet.FleetVarHostPlatform):
			// ok
		default:
			return fleet.NewInvalidArgumentError(fieldName, fmt.Sprintf("%sFLEET_VAR_%s is not allowed in %s for EJBCA.", errPrefix, v, fieldName))
		}
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
	case string(fleet.CATypeSmallstep):
		activity = fleet.ActivityDeletedSmallstep{
			Name: ca.Name,
		}
	case string(fleet.CATypeCustomESTProxy):
		activity = fleet.ActivityDeletedCustomESTProxy{
			Name: ca.Name,
		}
	case string(fleet.CATypeEJBCA):
		activity = fleet.ActivityDeletedEJBCA{
			Name: ca.Name,
		}
	}

	if err := svc.NewActivity(ctx, authz.UserFromContext(ctx), activity); err != nil {
		return err
	}

	return nil
}

func (svc *Service) BatchApplyCertificateAuthorities(ctx context.Context, incoming fleet.GroupedCertificateAuthorities, opts fleet.BatchApplyCertificateAuthoritiesOpts) error {
	if err := svc.authz.Authorize(ctx, &fleet.CertificateAuthority{}, fleet.ActionWrite); err != nil {
		return err
	}

	if !opts.ViaGitOps {
		// Note: This check is here primarily for future reference to help make the usage intent
		// clear and to differentiate behavior from dual-use endpoints that support patch semantics (e.g., app config)
		return fleet.NewInvalidArgumentError("gitops", "certificate_authorities: batch apply is intended only for use with gitops")
	}

	ops, err := svc.getCertificateAuthoritiesBatchOperations(ctx, incoming)
	if err != nil {
		return err
	}

	if ops == nil {
		svc.logger.DebugContext(ctx, "batch apply certificate authorities: no certificate authority changes to apply")
		return nil
	}

	if opts.SkipDeletes {
		ops.Delete = nil
	}

	if opts.DryRun {
		svc.logger.DebugContext(ctx, "batch apply certificate authorities: no certificate authority changes to apply")
		return nil
	}

	if err := svc.ds.BatchApplyCertificateAuthorities(ctx, *ops); err != nil {
		return err
	}

	if err := svc.recordActivitiesBatchApplyCAs(ctx, ops); err != nil {
		return err
	}

	return nil
}

func (svc *Service) getCertificateAuthoritiesBatchOperations(ctx context.Context, incoming fleet.GroupedCertificateAuthorities) (*fleet.CertificateAuthoritiesBatchOperations, error) {
	batchOps := &fleet.CertificateAuthoritiesBatchOperations{
		Add:    make([]*fleet.CertificateAuthority, 0),
		Delete: make([]*fleet.CertificateAuthority, 0),
		Update: make([]*fleet.CertificateAuthority, 0),
	}

	existing, err := svc.ds.GetGroupedCertificateAuthorities(ctx, true)
	if err != nil {
		return nil, err
	}

	// track processed CA names for duplicate name checking
	allNames := make(map[string][]string)
	checkAllNames := func(name, caType, displayCAType string) error {
		for i := 0; i < len(allNames[name]); i++ {
			if allNames[name][i] == caType {
				return fmtDuplicateCANameError(name, caType, displayCAType)
			}
		}
		allNames[name] = append(allNames[name], caType)
		return nil
	}

	// preprocess digicert
	for _, ca := range incoming.DigiCert {
		if ca.Name == "" {
			return nil, fleet.NewInvalidArgumentError("name", "certificate_authorities.digicert: CA name cannot be empty.")
		}
		ca.Preprocess()

		if err := checkAllNames(ca.Name, "digicert", "DigiCert"); err != nil {
			return nil, err
		}
	}
	// preprocess custom scep proxy
	for _, ca := range incoming.CustomScepProxy {
		if ca.Name == "" {
			return nil, fleet.NewInvalidArgumentError("name", "certificate_authorities.custom_scep_proxy: CA name cannot be empty.")
		}
		ca.Preprocess()

		if err := checkAllNames(ca.Name, "custom_scep_proxy", "Custom SCEP Proxy"); err != nil {
			return nil, err
		}
	}
	// preprocess hydrant
	for _, ca := range incoming.Hydrant {
		if ca.Name == "" {
			return nil, fleet.NewInvalidArgumentError("name", "certificate_authorities.hydrant: CA name cannot be empty.")
		}
		ca.Preprocess()

		if err := checkAllNames(ca.Name, "hydrant", "Hydrant"); err != nil {
			return nil, err
		}
	}
	for _, ca := range incoming.EST {
		if strings.TrimSpace(ca.Name) == "" {
			return nil, fleet.NewInvalidArgumentError("name", "certificate_authorities.custom_est_proxy: CA name cannot be empty.")
		}
		ca.Preprocess()

		if err := checkAllNames(ca.Name, "custom_est_proxy", "Custom EST Proxy"); err != nil {
			return nil, err
		}
	}
	// preprocess smallstep
	for _, ca := range incoming.Smallstep {
		if ca.Name == "" {
			return nil, fleet.NewInvalidArgumentError("name", "certificate_authorities.smallstep: CA name cannot be empty.")
		}
		ca.Preprocess()

		if err := checkAllNames(ca.Name, "smallstep", "Smallstep"); err != nil {
			return nil, err
		}
	}
	// preprocess ndes
	if incoming.NDESSCEP != nil {
		incoming.NDESSCEP.Preprocess()
	}

	if err := svc.processNDESSCEP(ctx, batchOps, incoming.NDESSCEP, existing.NDESSCEP); err != nil {
		return nil, err
	}
	if err := svc.processDigiCertCAs(ctx, batchOps, incoming.DigiCert, existing.DigiCert); err != nil {
		return nil, err
	}
	if err := svc.processCustomSCEPProxyCAs(ctx, batchOps, incoming.CustomScepProxy, existing.CustomScepProxy); err != nil {
		return nil, err
	}
	if err := svc.processHydrantCAs(ctx, batchOps, incoming.Hydrant, existing.Hydrant); err != nil {
		return nil, err
	}
	if err := svc.processESTCAs(ctx, batchOps, incoming.EST, existing.EST); err != nil {
		return nil, err
	}
	if err := svc.processSmallstepCAs(ctx, batchOps, incoming.Smallstep, existing.Smallstep); err != nil {
		return nil, err
	}

	return batchOps, nil
}

func (svc *Service) processNDESSCEP(ctx context.Context, batchOps *fleet.CertificateAuthoritiesBatchOperations, incoming *fleet.NDESSCEPProxyCA, existing *fleet.NDESSCEPProxyCA) error {
	ndesName := "NDES"

	if existing == nil && incoming == nil {
		// do nothing
		svc.logger.DebugContext(ctx, "no existing or incoming NDES SCEP CA, skipping")
		return nil
	}

	if existing != nil && incoming != nil && incoming.URL == existing.URL && incoming.AdminURL == existing.AdminURL && incoming.Username == existing.Username && incoming.Password == existing.Password {
		// all fields are identical so we can skip further validation and processing
		svc.logger.DebugContext(ctx, "existing and incoming NDES SCEP CA are identical, skipping")
		return nil
	}

	if existing != nil && (incoming == nil || (incoming.URL == "" && incoming.AdminURL == "" && incoming.Username == "" && incoming.Password == "")) {
		// delete current
		svc.logger.DebugContext(ctx, "deleting existing NDES SCEP CA as incoming is empty")
		batchOps.Delete = append(batchOps.Delete, &fleet.CertificateAuthority{
			Type:     string(fleet.CATypeNDESSCEPProxy),
			Name:     &ndesName,
			AdminURL: &existing.AdminURL,
			Username: &existing.Username,
			Password: &existing.Password,
		})
		return nil
	}

	if incoming.Password == "" || incoming.Password == fleet.MaskedPassword {
		return fleet.NewInvalidArgumentError("password", "certificate_authorities.ndes_scep_proxy.password: NDES SCEP password cannot be empty.")
	}

	if err := svc.validateNDESSCEPProxy(ctx, incoming, "certificate_authorities.ndes_scep_proxy: "); err != nil {
		return err
	}

	// add if there is no existing
	if existing == nil || (existing.URL == "" && existing.AdminURL == "" && existing.Username == "" && existing.Password == "") {
		svc.logger.DebugContext(ctx, "adding new NDES SCEP CA as none exists")
		batchOps.Add = append(batchOps.Add, &fleet.CertificateAuthority{
			Type:     string(fleet.CATypeNDESSCEPProxy),
			Name:     &ndesName,
			URL:      &incoming.URL,
			AdminURL: &incoming.AdminURL,
			Username: &incoming.Username,
			Password: &incoming.Password,
		})
		return nil
	}

	// otherwise update with existing id
	svc.logger.DebugContext(ctx, "updating existing NDES SCEP CA")
	incoming.ID = existing.ID
	batchOps.Update = append(batchOps.Update, &fleet.CertificateAuthority{
		Type:     string(fleet.CATypeNDESSCEPProxy),
		Name:     &ndesName,
		URL:      &incoming.URL,
		AdminURL: &incoming.AdminURL,
		Username: &incoming.Username,
		Password: &incoming.Password,
	})

	return nil
}

func (svc *Service) processDigiCertCAs(ctx context.Context, batchOps *fleet.CertificateAuthoritiesBatchOperations, incomingCAs []fleet.DigiCertCA, existingCAs []fleet.DigiCertCA) error {
	incomingByName := make(map[string]*fleet.DigiCertCA)
	for _, incoming := range incomingCAs {
		if incoming.Name == "" {
			return fleet.NewInvalidArgumentError("name", "certificate_authorities.digicert.name: Name cannot be empty.")
		}
		// Note: caller is responsible for ensuring incoming list has no duplicates
		incomingByName[incoming.Name] = &incoming
	}

	existingByName := make(map[string]*fleet.DigiCertCA)
	for _, existing := range existingCAs {
		if _, ok := incomingByName[existing.Name]; !ok {
			// if current CA isn't in the incoming list, we should delete it
			batchOps.Delete = append(batchOps.Delete, &fleet.CertificateAuthority{
				Type:                          string(fleet.CATypeDigiCert),
				Name:                          &existing.Name,
				URL:                           &existing.URL,
				APIToken:                      &existing.APIToken,
				ProfileID:                     &existing.ProfileID,
				CertificateCommonName:         &existing.CertificateCommonName,
				CertificateUserPrincipalNames: &existing.CertificateUserPrincipalNames,
				CertificateSeatID:             &existing.CertificateSeatID,
			})
		}
		// Note: datastore is responsible for ensuring no existing list has no duplicates
		existingByName[existing.Name] = &existing
	}

	for name, incoming := range incomingByName {
		// check if incoming name matches existing
		existing, ok := existingByName[name]
		switch {
		case ok && incoming.Equals(existing):
			// found and identical so do nothing
			continue
		case ok:
			// found but not identical so update
			batchOps.Update = append(batchOps.Update, &fleet.CertificateAuthority{
				Type:                          string(fleet.CATypeDigiCert),
				Name:                          &incoming.Name,
				URL:                           &incoming.URL,
				APIToken:                      &incoming.APIToken,
				ProfileID:                     &incoming.ProfileID,
				CertificateCommonName:         &incoming.CertificateCommonName,
				CertificateUserPrincipalNames: &incoming.CertificateUserPrincipalNames,
				CertificateSeatID:             &incoming.CertificateSeatID,
			})
		default:
			// not found so add
			batchOps.Add = append(batchOps.Add, &fleet.CertificateAuthority{
				Type:                          string(fleet.CATypeDigiCert),
				Name:                          &incoming.Name,
				URL:                           &incoming.URL,
				APIToken:                      &incoming.APIToken,
				ProfileID:                     &incoming.ProfileID,
				CertificateCommonName:         &incoming.CertificateCommonName,
				CertificateUserPrincipalNames: &incoming.CertificateUserPrincipalNames,
				CertificateSeatID:             &incoming.CertificateSeatID,
			})
		}

		if err := svc.validateDigicert(ctx, incoming, "certificate_authorities.digicert: "); err != nil {
			return err
		}
	}

	return nil
}

func (svc *Service) processCustomSCEPProxyCAs(ctx context.Context, batchOps *fleet.CertificateAuthoritiesBatchOperations, incomingCAs []fleet.CustomSCEPProxyCA, existingCAs []fleet.CustomSCEPProxyCA) error {
	incomingByName := make(map[string]*fleet.CustomSCEPProxyCA)
	for _, incoming := range incomingCAs {
		// Note: caller is responsible for ensuring incoming list has no duplicates
		incomingByName[incoming.Name] = &incoming
	}

	existingByName := make(map[string]*fleet.CustomSCEPProxyCA)
	for _, existing := range existingCAs {
		// if existing CA isn't in the incoming list, we should delete it
		if _, ok := incomingByName[existing.Name]; !ok {
			batchOps.Delete = append(batchOps.Delete, &fleet.CertificateAuthority{
				Type:      string(fleet.CATypeCustomSCEPProxy),
				Name:      &existing.Name,
				URL:       &existing.URL,
				Challenge: &existing.Challenge,
			})
		}
		// Note: datastore is responsible for ensuring no existing list has no duplicates
		existingByName[existing.Name] = &existing
	}

	for name, incoming := range incomingByName {
		if err := svc.validateCustomSCEPProxy(ctx, incoming, "certificate_authorities.custom_scep_proxy: "); err != nil {
			return err
		}
		// create the payload to be added or updated
		if _, ok := existingByName[name]; ok {
			// update existing
			batchOps.Update = append(batchOps.Update, &fleet.CertificateAuthority{
				Type:      string(fleet.CATypeCustomSCEPProxy),
				Name:      &incoming.Name,
				URL:       &incoming.URL,
				Challenge: &incoming.Challenge,
			})
		} else {
			// add new
			batchOps.Add = append(batchOps.Add, &fleet.CertificateAuthority{
				Type:      string(fleet.CATypeCustomSCEPProxy),
				Name:      &incoming.Name,
				URL:       &incoming.URL,
				Challenge: &incoming.Challenge,
			})
		}
	}

	return nil
}

func (svc *Service) processHydrantCAs(ctx context.Context, batchOps *fleet.CertificateAuthoritiesBatchOperations, incomingCAs []fleet.HydrantCA, existingCAs []fleet.HydrantCA) error {
	incomingByName := make(map[string]*fleet.HydrantCA)
	for _, incoming := range incomingCAs {
		// Note: caller is responsible for ensuring incoming list has no duplicates
		incomingByName[incoming.Name] = &incoming
	}

	existingByName := make(map[string]*fleet.HydrantCA)
	for _, existing := range existingCAs {
		// if existing CA isn't in the incoming list, we should delete it
		if _, ok := incomingByName[existing.Name]; !ok {
			batchOps.Delete = append(batchOps.Delete, &fleet.CertificateAuthority{
				Type:         string(fleet.CATypeHydrant),
				Name:         &existing.Name,
				URL:          &existing.URL,
				ClientID:     &existing.ClientID,
				ClientSecret: &existing.ClientSecret,
			})
		}
		// Note: datastore is responsible for ensuring no existing list has no duplicates
		existingByName[existing.Name] = &existing
	}

	for name, incoming := range incomingByName {
		if err := svc.validateHydrant(ctx, incoming, "certificate_authorities.hydrant: "); err != nil {
			return err
		}

		// create the payload to be added or updated
		if _, ok := existingByName[name]; ok {
			// update existing
			batchOps.Update = append(batchOps.Update, &fleet.CertificateAuthority{
				Type:         string(fleet.CATypeHydrant),
				Name:         &incoming.Name,
				URL:          &incoming.URL,
				ClientID:     &incoming.ClientID,
				ClientSecret: &incoming.ClientSecret,
			})
		} else {
			// add new
			batchOps.Add = append(batchOps.Add, &fleet.CertificateAuthority{
				Type:         string(fleet.CATypeHydrant),
				Name:         &incoming.Name,
				URL:          &incoming.URL,
				ClientID:     &incoming.ClientID,
				ClientSecret: &incoming.ClientSecret,
			})
		}
	}

	return nil
}

func (svc *Service) processESTCAs(ctx context.Context, batchOps *fleet.CertificateAuthoritiesBatchOperations, incomingCAs []fleet.ESTProxyCA, existingCAs []fleet.ESTProxyCA) error {
	incomingByName := make(map[string]*fleet.ESTProxyCA)
	for _, incoming := range incomingCAs {
		// Note: caller is responsible for ensuring incoming list has no duplicates
		incomingByName[incoming.Name] = &incoming
	}

	existingByName := make(map[string]*fleet.ESTProxyCA)
	for _, existing := range existingCAs {
		// if existing CA isn't in the incoming list, we should delete it
		if _, ok := incomingByName[existing.Name]; !ok {
			batchOps.Delete = append(batchOps.Delete, &fleet.CertificateAuthority{
				Type:     string(fleet.CATypeCustomESTProxy),
				Name:     &existing.Name,
				URL:      &existing.URL,
				Username: &existing.Username,
				Password: &existing.Password,
			})
		}
		// Note: datastore is responsible for ensuring no existing list has no duplicates
		existingByName[existing.Name] = &existing
	}

	for name, incoming := range incomingByName {
		if err := svc.validateEST(ctx, incoming, "certificate_authorities.custom_est_proxy: "); err != nil {
			return err
		}

		// create the payload to be added or updated
		if _, ok := existingByName[name]; ok {
			// update existing
			batchOps.Update = append(batchOps.Update, &fleet.CertificateAuthority{
				Type:     string(fleet.CATypeCustomESTProxy),
				Name:     &incoming.Name,
				URL:      &incoming.URL,
				Username: &incoming.Username,
				Password: &incoming.Password,
			})
		} else {
			// add new
			batchOps.Add = append(batchOps.Add, &fleet.CertificateAuthority{
				Type:     string(fleet.CATypeCustomESTProxy),
				Name:     &incoming.Name,
				URL:      &incoming.URL,
				Username: &incoming.Username,
				Password: &incoming.Password,
			})
		}
	}

	return nil
}

func (svc *Service) processSmallstepCAs(ctx context.Context, batchOps *fleet.CertificateAuthoritiesBatchOperations, incomingCAs []fleet.SmallstepSCEPProxyCA, existingCAs []fleet.SmallstepSCEPProxyCA) error {
	incomingByName := make(map[string]*fleet.SmallstepSCEPProxyCA)
	for _, incoming := range incomingCAs {
		// Note: caller is responsible for ensuring incoming list has no duplicates
		incomingByName[incoming.Name] = &incoming
	}

	existingByName := make(map[string]*fleet.SmallstepSCEPProxyCA)
	for _, existing := range existingCAs {
		// if existing CA isn't in the incoming list, we should delete it
		if _, ok := incomingByName[existing.Name]; !ok {
			batchOps.Delete = append(batchOps.Delete, &fleet.CertificateAuthority{
				Type:         string(fleet.CATypeSmallstep),
				Name:         &existing.Name,
				URL:          &existing.URL,
				ChallengeURL: &existing.ChallengeURL,
				Username:     &existing.Username,
				Password:     &existing.Password,
			})
		}
		// Note: datastore is responsible for ensuring no existing list has no duplicates
		existingByName[existing.Name] = &existing
	}

	for name, incoming := range incomingByName {
		if err := svc.validateSmallstepSCEPProxy(ctx, incoming, "certificate_authorities.smallstep: "); err != nil {
			return err
		}

		// create the payload to be added or updated
		if _, ok := existingByName[name]; ok {
			// update existing
			batchOps.Update = append(batchOps.Update, &fleet.CertificateAuthority{
				Type:         string(fleet.CATypeSmallstep),
				Name:         &incoming.Name,
				URL:          &incoming.URL,
				ChallengeURL: &incoming.ChallengeURL,
				Username:     &incoming.Username,
				Password:     &incoming.Password,
			})
		} else {
			// add new
			batchOps.Add = append(batchOps.Add, &fleet.CertificateAuthority{
				Type:         string(fleet.CATypeSmallstep),
				Name:         &incoming.Name,
				URL:          &incoming.URL,
				ChallengeURL: &incoming.ChallengeURL,
				Username:     &incoming.Username,
				Password:     &incoming.Password,
			})
		}
	}

	return nil
}

// recordActivitiesBatchApplyCAs records activities for batch operations on certificate authorities
// (i.e. added, edited, deleted).
func (svc *Service) recordActivitiesBatchApplyCAs(ctx context.Context, ops *fleet.CertificateAuthoritiesBatchOperations) error {
	if ops == nil {
		return nil
	}

	for _, ca := range ops.Add {
		switch ca.Type {
		case string(fleet.CATypeNDESSCEPProxy):
			if err := svc.NewActivity(ctx, authz.UserFromContext(ctx), fleet.ActivityAddedNDESSCEPProxy{}); err != nil {
				return ctxerr.Wrap(ctx, err, "create activity for added NDES SCEP proxy")
			}
		case string(fleet.CATypeDigiCert):
			if err := svc.NewActivity(ctx, authz.UserFromContext(ctx), fleet.ActivityAddedDigiCert{Name: *ca.Name}); err != nil {
				return ctxerr.Wrap(ctx, err, "create activity for added DigiCert")
			}
		case string(fleet.CATypeCustomSCEPProxy):
			if err := svc.NewActivity(ctx, authz.UserFromContext(ctx), fleet.ActivityAddedCustomSCEPProxy{Name: *ca.Name}); err != nil {
				return ctxerr.Wrap(ctx, err, "create activity for added custom SCEP proxy")
			}
		case string(fleet.CATypeHydrant):
			if err := svc.NewActivity(ctx, authz.UserFromContext(ctx), fleet.ActivityAddedHydrant{Name: *ca.Name}); err != nil {
				return ctxerr.Wrap(ctx, err, "create activity for added hydrant")
			}
		case string(fleet.CATypeCustomESTProxy):
			if err := svc.NewActivity(ctx, authz.UserFromContext(ctx), fleet.ActivityAddedCustomESTProxy{Name: *ca.Name}); err != nil {
				return ctxerr.Wrap(ctx, err, "create activity for added custom est proxy")
			}
		case string(fleet.CATypeSmallstep):
			if err := svc.NewActivity(ctx, authz.UserFromContext(ctx), fleet.ActivityAddedSmallstep{Name: *ca.Name}); err != nil {
				return ctxerr.Wrap(ctx, err, "create activity for added smallstep SCEP proxy")
			}
		}
	}
	for _, ca := range ops.Update {
		switch ca.Type {
		case string(fleet.CATypeNDESSCEPProxy):
			if err := svc.NewActivity(ctx, authz.UserFromContext(ctx), fleet.ActivityEditedNDESSCEPProxy{}); err != nil {
				return ctxerr.Wrap(ctx, err, "create activity for edited NDES SCEP proxy")
			}
		case string(fleet.CATypeDigiCert):
			if err := svc.NewActivity(ctx, authz.UserFromContext(ctx), fleet.ActivityEditedDigiCert{Name: *ca.Name}); err != nil {
				return ctxerr.Wrap(ctx, err, "create activity for edited DigiCert")
			}
		case string(fleet.CATypeCustomSCEPProxy):
			if err := svc.NewActivity(ctx, authz.UserFromContext(ctx), fleet.ActivityEditedCustomSCEPProxy{Name: *ca.Name}); err != nil {
				return ctxerr.Wrap(ctx, err, "create activity for edited custom SCEP proxy")
			}
		case string(fleet.CATypeHydrant):
			if err := svc.NewActivity(ctx, authz.UserFromContext(ctx), fleet.ActivityEditedHydrant{Name: *ca.Name}); err != nil {
				return ctxerr.Wrap(ctx, err, "create activity for edited hydrant")
			}
		case string(fleet.CATypeCustomESTProxy):
			if err := svc.NewActivity(ctx, authz.UserFromContext(ctx), fleet.ActivityEditedCustomESTProxy{Name: *ca.Name}); err != nil {
				return ctxerr.Wrap(ctx, err, "create activityu for edited custom EST proxy")
			}
		case string(fleet.CATypeSmallstep):
			if err := svc.NewActivity(ctx, authz.UserFromContext(ctx), fleet.ActivityEditedSmallstep{Name: *ca.Name}); err != nil {
				return ctxerr.Wrap(ctx, err, "create activity for edited smallstep SCEP proxy")
			}
		}
	}
	for _, ca := range ops.Delete {
		switch ca.Type {
		case string(fleet.CATypeNDESSCEPProxy):
			if err := svc.NewActivity(ctx, authz.UserFromContext(ctx), fleet.ActivityDeletedNDESSCEPProxy{}); err != nil {
				return ctxerr.Wrap(ctx, err, "create activity for deleted NDES SCEP proxy")
			}
		case string(fleet.CATypeDigiCert):
			if err := svc.NewActivity(ctx, authz.UserFromContext(ctx), fleet.ActivityDeletedDigiCert{Name: *ca.Name}); err != nil {
				return ctxerr.Wrap(ctx, err, "create activity for deleted DigiCert")
			}
		case string(fleet.CATypeCustomSCEPProxy):
			if err := svc.NewActivity(ctx, authz.UserFromContext(ctx), fleet.ActivityDeletedCustomSCEPProxy{Name: *ca.Name}); err != nil {
				return ctxerr.Wrap(ctx, err, "create activity for deleted custom SCEP proxy")
			}
		case string(fleet.CATypeHydrant):
			if err := svc.NewActivity(ctx, authz.UserFromContext(ctx), fleet.ActivityDeletedHydrant{Name: *ca.Name}); err != nil {
				return ctxerr.Wrap(ctx, err, "create activity for deleted hydrant")
			}
		case string(fleet.CATypeCustomESTProxy):
			if err := svc.NewActivity(ctx, authz.UserFromContext(ctx), fleet.ActivityDeletedCustomESTProxy{Name: *ca.Name}); err != nil {
				return ctxerr.Wrap(ctx, err, "create activity for deleted custom EST proxy")
			}
		case string(fleet.CATypeSmallstep):
			if err := svc.NewActivity(ctx, authz.UserFromContext(ctx), fleet.ActivityDeletedSmallstep{Name: *ca.Name}); err != nil {
				return ctxerr.Wrap(ctx, err, "create activity for deleted smallstep SCEP proxy")
			}
		}
	}

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

	switch {
	case p.DigiCertCAUpdatePayload != nil:
		if p.DigiCertCAUpdatePayload.IsEmpty() {
			return &fleet.BadRequestError{Message: fmt.Sprintf("%sDigiCert CA update payload is empty", errPrefix)}
		}

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
	case p.HydrantCAUpdatePayload != nil:
		if p.HydrantCAUpdatePayload.IsEmpty() {
			return &fleet.BadRequestError{Message: fmt.Sprintf("%sHydrant CA update payload is empty", errPrefix)}
		}

		if err := p.HydrantCAUpdatePayload.ValidateRelatedFields(errPrefix, *oldCA.Name); err != nil {
			return err
		}
		p.HydrantCAUpdatePayload.Preprocess()
		if err := svc.validateHydrantUpdate(ctx, p.HydrantCAUpdatePayload, oldCA, errPrefix); err != nil {
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
	case p.CustomESTCAUpdatePayload != nil:
		if p.CustomESTCAUpdatePayload.IsEmpty() {
			return &fleet.BadRequestError{Message: fmt.Sprintf("%sCustom EST CA update payload is empty", errPrefix)}
		}

		if err := p.CustomESTCAUpdatePayload.ValidateRelatedFields(errPrefix, *oldCA.Name); err != nil {
			return err
		}
		p.CustomESTCAUpdatePayload.Preprocess()
		if err := svc.validateCustomESTUpdate(ctx, p.CustomESTCAUpdatePayload, oldCA, errPrefix); err != nil {
			return err
		}
		caToUpdate.Type = string(fleet.CATypeCustomESTProxy)
		caToUpdate.Name = p.CustomESTCAUpdatePayload.Name
		caToUpdate.URL = p.CustomESTCAUpdatePayload.URL
		caToUpdate.Username = p.CustomESTCAUpdatePayload.Username
		caToUpdate.Password = p.CustomESTCAUpdatePayload.Password
		if caToUpdate.Name != nil {
			caActivityName = *caToUpdate.Name
		} else {
			caActivityName = *oldCA.Name
		}
		activity = fleet.ActivityEditedCustomESTProxy{Name: caActivityName}
	case p.NDESSCEPProxyCAUpdatePayload != nil:
		if p.NDESSCEPProxyCAUpdatePayload.IsEmpty() {
			return &fleet.BadRequestError{Message: fmt.Sprintf("%sNDES SCEP Proxy CA update payload is empty", errPrefix)}
		}

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
	case p.CustomSCEPProxyCAUpdatePayload != nil:
		if p.CustomSCEPProxyCAUpdatePayload.IsEmpty() {
			return &fleet.BadRequestError{Message: fmt.Sprintf("%sCustom SCEP Proxy CA update payload is empty", errPrefix)}
		}

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

	case p.SmallstepSCEPProxyCAUpdatePayload != nil:
		if p.SmallstepSCEPProxyCAUpdatePayload.IsEmpty() {
			return &fleet.BadRequestError{Message: fmt.Sprintf("%sSmallstep SCEP Proxy CA update payload is empty", errPrefix)}
		}

		if err := p.SmallstepSCEPProxyCAUpdatePayload.ValidateRelatedFields(errPrefix, *oldCA.Name); err != nil {
			return err
		}
		p.SmallstepSCEPProxyCAUpdatePayload.Preprocess()
		if err := svc.validateSmallstepSCEPProxyUpdate(ctx, p.SmallstepSCEPProxyCAUpdatePayload, oldCA, errPrefix); err != nil {
			return err
		}

		caToUpdate.Type = string(fleet.CATypeSmallstep)
		caToUpdate.Name = p.SmallstepSCEPProxyCAUpdatePayload.Name
		caToUpdate.URL = p.SmallstepSCEPProxyCAUpdatePayload.URL
		caToUpdate.ChallengeURL = p.SmallstepSCEPProxyCAUpdatePayload.ChallengeURL
		caToUpdate.Username = p.SmallstepSCEPProxyCAUpdatePayload.Username
		caToUpdate.Password = p.SmallstepSCEPProxyCAUpdatePayload.Password
		if caToUpdate.Name != nil {
			caActivityName = *caToUpdate.Name
		} else {
			caActivityName = *oldCA.Name
		}
		activity = fleet.ActivityEditedSmallstep{Name: caActivityName}
	case p.EJBCACAUpdatePayload != nil:
		if p.EJBCACAUpdatePayload.IsEmpty() {
			return &fleet.BadRequestError{Message: fmt.Sprintf("%sEJBCA CA update payload is empty", errPrefix)}
		}
		if err := p.EJBCACAUpdatePayload.ValidateRelatedFields(errPrefix, *oldCA.Name); err != nil {
			return err
		}
		p.EJBCACAUpdatePayload.Preprocess()
		certPEM, keyPEM, err := svc.validateEJBCAUpdate(ctx, p.EJBCACAUpdatePayload, oldCA, errPrefix)
		if err != nil {
			return err
		}

		caToUpdate.Type = string(fleet.CATypeEJBCA)
		caToUpdate.Name = p.EJBCACAUpdatePayload.Name
		caToUpdate.URL = p.EJBCACAUpdatePayload.URL
		// When a P12 was supplied, certPEM/keyPEM hold the decoded material;
		// otherwise the stored values are retained (pointers stay nil so the
		// datastore's generateUpdateQueryWithArgs skips those columns).
		if certPEM != "" {
			caToUpdate.ClientCertPEM = &certPEM
		}
		if keyPEM != "" {
			caToUpdate.ClientKeyPEM = &keyPEM
		}
		caToUpdate.TrustCABundlePEM = p.EJBCACAUpdatePayload.TrustCABundlePEM
		caToUpdate.EJBCACAName = p.EJBCACAUpdatePayload.CertificateAuthorityNameEJBCA
		caToUpdate.EJBCACertificateProfileName = p.EJBCACAUpdatePayload.CertificateProfileName
		caToUpdate.EJBCAEndEntityProfileName = p.EJBCACAUpdatePayload.EndEntityProfileName
		caToUpdate.EJBCAUsernameTemplate = p.EJBCACAUpdatePayload.UsernameTemplate
		caToUpdate.CertificateUserPrincipalNames = p.EJBCACAUpdatePayload.CertificateUserPrincipalNames
		if caToUpdate.Name != nil {
			caActivityName = *caToUpdate.Name
		} else {
			caActivityName = *oldCA.Name
		}
		activity = fleet.ActivityEditedEJBCA{Name: caActivityName}
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

		// We want to generate a DigiCertCA struct with all required fields to verify the new URL.
		// If URL or APIToken are not being updated we use the existing values from oldCA
		digicertCA := fleet.DigiCertCA{
			URL: *digicert.URL,
		}
		if digicert.ProfileID != nil {
			digicertCA.ProfileID = *digicert.ProfileID
		} else {
			digicertCA.ProfileID = *oldCA.ProfileID
		}
		if digicert.APIToken != nil {
			digicertCA.APIToken = *digicert.APIToken
		} else {
			digicertCA.APIToken = *oldCA.APIToken
		}
		if err := svc.digiCertService.VerifyProfileID(ctx, digicertCA); err != nil {
			svc.logger.ErrorContext(ctx, "Failed to validate DigiCert URL", "err", err)
			return &fleet.BadRequestError{Message: fmt.Sprintf("%sCould not verify DigiCert URL: %s. Please correct and try again.", errPrefix, err.Error())}
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
			svc.logger.ErrorContext(ctx, "Failed to validate DigiCert profile GUID", "err", err)
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

func (svc *Service) validateHydrantUpdate(ctx context.Context, hydrant *fleet.HydrantCAUpdatePayload, oldCA *fleet.CertificateAuthority, errPrefix string) error {
	if hydrant.Name != nil {
		if err := validateCAName(*hydrant.Name, errPrefix); err != nil {
			return err
		}
	}
	if hydrant.URL != nil {
		if err := validateURL(*hydrant.URL, "Hydrant", errPrefix); err != nil {
			return err
		}

		hydrantCAToVerify := fleet.ESTProxyCA{ // The hydrant service for verification only requires the URL.
			URL: *hydrant.URL,
		}
		if err := svc.estService.ValidateESTURL(ctx, hydrantCAToVerify); err != nil {
			return &fleet.BadRequestError{Message: fmt.Sprintf("%sInvalid Hydrant URL. Please correct and try again.", errPrefix)}
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

func (svc *Service) validateCustomESTUpdate(ctx context.Context, estUpdate *fleet.CustomESTCAUpdatePayload, oldCA *fleet.CertificateAuthority, errPrefix string) error {
	if estUpdate.Name != nil {
		if err := validateCAName(*estUpdate.Name, errPrefix); err != nil {
			return err
		}
	}
	if estUpdate.URL != nil {
		if err := validateURL(*estUpdate.URL, "EST", errPrefix); err != nil {
			return err
		}

		hydrantCAToVerify := fleet.ESTProxyCA{ // The EST service for verification only requires the URL.
			URL: *estUpdate.URL,
		}
		if err := svc.estService.ValidateESTURL(ctx, hydrantCAToVerify); err != nil {
			return &fleet.BadRequestError{Message: fmt.Sprintf("%sInvalid EST URL. Please correct and try again.", errPrefix)}
		}
	}
	if estUpdate.Username != nil && *estUpdate.Username == "" {
		return &fleet.BadRequestError{
			Message: fmt.Sprintf("%sInvalid EST Username. Please correct and try again.", errPrefix),
		}
	}
	if estUpdate.Password != nil && *estUpdate.Password == "" {
		return &fleet.BadRequestError{
			Message: fmt.Sprintf("%sInvalid EST Password. Please correct and try again.", errPrefix),
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
			svc.logger.ErrorContext(ctx, "Failed to validate NDES SCEP URL", "err", err)
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
			svc.logger.ErrorContext(ctx, "Failed to validate NDES SCEP admin URL", "err", err)
			switch {
			case errors.As(err, &scep.NDESPasswordCacheFullError{}):
				return &fleet.BadRequestError{Message: fmt.Sprintf("%sThe NDES password cache is full. Please increase the number of cached passwords in NDES and try again.", errPrefix)}
			case errors.As(err, &scep.NDESInsufficientPermissionsError{}):
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
			svc.logger.ErrorContext(ctx, "Failed to validate custom SCEP URL", "err", err)
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

func (svc *Service) validateSmallstepSCEPProxyUpdate(ctx context.Context, smallstep *fleet.SmallstepSCEPProxyCAUpdatePayload, oldCa *fleet.CertificateAuthority, errPrefix string) error {
	if smallstep.Name != nil {
		if err := validateCAName(*smallstep.Name, errPrefix); err != nil {
			return err
		}
	}
	if smallstep.URL != nil {
		if err := validateURL(*smallstep.URL, "SCEP", errPrefix); err != nil {
			return err
		}
		if err := svc.scepConfigService.ValidateSCEPURL(ctx, *smallstep.URL); err != nil {
			svc.logger.ErrorContext(ctx, "Failed to validate Smallstep SCEP URL", "err", err)
			return &fleet.BadRequestError{Message: fmt.Sprintf("%sInvalid SCEP URL. Please correct and try again.", errPrefix)}
		}
	}
	// Call challenge URL to validate all fields are valid
	if smallstep.ChallengeURL != nil || smallstep.Username != nil || smallstep.Password != nil {

		smallstepSCEPProxy := fleet.SmallstepSCEPProxyCA{
			ChallengeURL: *oldCa.ChallengeURL,
			Username:     *oldCa.Username,
			Password:     *oldCa.Password,
		} // The object we are building to validate fields are valid.

		if smallstep.URL != nil {
			smallstepSCEPProxy.URL = *smallstep.URL
		} else {
			smallstepSCEPProxy.URL = *oldCa.URL
		}

		// Additional validation if url was updated
		if smallstep.ChallengeURL != nil {
			if err := validateURL(*smallstep.ChallengeURL, "Challenge", errPrefix); err != nil {
				return err
			}
			smallstepSCEPProxy.ChallengeURL = *smallstep.ChallengeURL
		}

		if smallstep.Username != nil {
			if *smallstep.Username == "" {
				return &fleet.BadRequestError{
					Message: fmt.Sprintf("%sSmallstep SCEP Proxy username cannot be empty", errPrefix),
				}
			}
			smallstepSCEPProxy.Username = *smallstep.Username
		}

		if smallstep.Password != nil {
			if *smallstep.Password == "" {
				return &fleet.BadRequestError{
					Message: fmt.Sprintf("%sSmallstep SCEP Proxy password cannot be empty", errPrefix),
				}
			}
			smallstepSCEPProxy.Password = *smallstep.Password
		}

		if err := svc.scepConfigService.ValidateSmallstepChallengeURL(ctx, smallstepSCEPProxy); err != nil {
			svc.logger.ErrorContext(ctx, "Failed to validate Smallstep challenge URL", "err", err)
			return &fleet.BadRequestError{Message: fmt.Sprintf("%sInvalid challenge URL or credentials. Please correct and try again.", errPrefix)}
		}
	}

	return nil
}

// validateEJBCAUpdate validates an EJBCA update payload and, when a new P12
// is supplied, decodes it into PEM-encoded cert + key strings returned to the
// caller. Returns empty strings when no rotation is in progress.
//
// After validating, probes the EJBCA REST API end-to-end with the merged
// (new + existing) configuration to confirm the change is functional before
// persisting.
func (svc *Service) validateEJBCAUpdate(ctx context.Context, ejbcaUpdate *fleet.EJBCACAUpdatePayload, oldCA *fleet.CertificateAuthority, errPrefix string) (certPEM, keyPEM string, err error) {
	if ejbcaUpdate.Name != nil {
		if err := validateCAName(*ejbcaUpdate.Name, errPrefix); err != nil {
			return "", "", err
		}
	}
	if ejbcaUpdate.URL != nil {
		if err := validateURL(*ejbcaUpdate.URL, "EJBCA", errPrefix); err != nil {
			return "", "", err
		}
	}

	if ejbcaUpdate.ClientP12 != nil {
		certPEM, keyPEM, err = decodeEJBCAClientP12(*ejbcaUpdate.ClientP12, *ejbcaUpdate.ClientP12Password)
		if err != nil {
			return "", "", fleet.NewInvalidArgumentError("client_p12", fmt.Sprintf("%sCould not decode PKCS#12 client certificate: %s", errPrefix, err.Error()))
		}
		// Discard the upload-only material so downstream callers don't
		// accidentally persist it.
		ejbcaUpdate.ClientP12 = nil
		ejbcaUpdate.ClientP12Password = nil
	}

	if ejbcaUpdate.TrustCABundlePEM != nil && *ejbcaUpdate.TrustCABundlePEM != "" {
		if !x509.NewCertPool().AppendCertsFromPEM([]byte(*ejbcaUpdate.TrustCABundlePEM)) {
			return "", "", fleet.NewInvalidArgumentError("trust_ca_bundle", fmt.Sprintf("%strust_ca_bundle did not contain any usable certificates.", errPrefix))
		}
	}

	if ejbcaUpdate.UsernameTemplate != nil {
		if strings.TrimSpace(*ejbcaUpdate.UsernameTemplate) == "" {
			return "", "", fleet.NewInvalidArgumentError("username_template", fmt.Sprintf("%susername_template cannot be empty.", errPrefix))
		}
		if err := validateEJBCAFleetVarUsage(*ejbcaUpdate.UsernameTemplate, "username_template", errPrefix); err != nil {
			return "", "", err
		}
	}
	if ejbcaUpdate.CertificateUserPrincipalNames != nil {
		for _, upn := range *ejbcaUpdate.CertificateUserPrincipalNames {
			if strings.TrimSpace(upn) == "" {
				return "", "", fleet.NewInvalidArgumentError("certificate_user_principal_names", fmt.Sprintf("%sUPN entries cannot be empty.", errPrefix))
			}
			if err := validateEJBCAFleetVarUsage(upn, "certificate_user_principal_names", errPrefix); err != nil {
				return "", "", err
			}
		}
	}

	// Build the merged config (new field where supplied, old field otherwise)
	// and probe the EJBCA REST API to confirm reachability + mTLS auth before
	// we persist.
	merged := fleet.EJBCACA{
		Name:                          stringOrDeref(ejbcaUpdate.Name, oldCA.Name),
		URL:                           stringOrDeref(ejbcaUpdate.URL, oldCA.URL),
		ClientCertPEM:                 stringOrDeref(strPtrIf(certPEM), oldCA.ClientCertPEM),
		ClientKeyPEM:                  stringOrDeref(strPtrIf(keyPEM), oldCA.ClientKeyPEM),
		TrustCABundlePEM:              stringOrDeref(ejbcaUpdate.TrustCABundlePEM, oldCA.TrustCABundlePEM),
		CertificateAuthorityNameEJBCA: stringOrDeref(ejbcaUpdate.CertificateAuthorityNameEJBCA, oldCA.EJBCACAName),
		CertificateProfileName:        stringOrDeref(ejbcaUpdate.CertificateProfileName, oldCA.EJBCACertificateProfileName),
		EndEntityProfileName:          stringOrDeref(ejbcaUpdate.EndEntityProfileName, oldCA.EJBCAEndEntityProfileName),
		UsernameTemplate:              stringOrDeref(ejbcaUpdate.UsernameTemplate, oldCA.EJBCAUsernameTemplate),
	}
	if err := svc.ejbcaService.VerifyConnection(ctx, merged); err != nil {
		svc.logger.ErrorContext(ctx, "Failed to verify EJBCA connection on update", "err", err)
		return "", "", &fleet.BadRequestError{Message: fmt.Sprintf("%sCould not verify EJBCA connection: %s. Please correct and try again.", errPrefix, err.Error())}
	}

	return certPEM, keyPEM, nil
}

// stringOrDeref returns *newPtr if non-nil, else *oldPtr, else "".
func stringOrDeref(newPtr *string, oldPtr *string) string {
	if newPtr != nil {
		return *newPtr
	}
	if oldPtr != nil {
		return *oldPtr
	}
	return ""
}

// strPtrIf returns &s if s != "", else nil. Helper for routing newly-decoded
// PEM through stringOrDeref alongside the existing pointer-based plumbing.
func strPtrIf(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func fmtDuplicateCANameError(name, caType, displayCAType string) error {
	return fleet.NewInvalidArgumentError("name", fmt.Sprintf("certificate_authorities.%s.name: Couldn’t edit certificate authority. "+
		"\"%s\" name is already used by another %s certificate authority. Please choose a different name and try again.", caType, name, displayCAType))
}
