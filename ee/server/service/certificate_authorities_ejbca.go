package service

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/fleetdm/fleet/v4/ee/server/service/ejbca"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

const (
	// ejbcaMaxFieldLength is the size of the varchar columns that hold the EJBCA names and templates.
	ejbcaMaxFieldLength = 255
	// ejbcaMaxPEMLength is the size of a TEXT column, which holds the client certificate chain and the trust CA bundle.
	ejbcaMaxPEMLength = 65535
	// ejbcaMaxP12Size bounds a client PKCS#12, which is a few KB for a certificate, its chain, and a key.
	ejbcaMaxP12Size = 1 << 20
)

// validateEJBCA checks a new EJBCA CA, reads the client certificate and key out of its PKCS#12, and verifies the connection to EJBCA.
func (svc *Service) validateEJBCA(ctx context.Context, ca *fleet.EJBCA, errPrefix string) error {
	if err := validateCAName(ca.Name, errPrefix); err != nil {
		return err
	}
	if err := validateURL(ca.URL, "EJBCA", errPrefix); err != nil {
		return err
	}
	if err := validateEJBCAFields(ca, errPrefix); err != nil {
		return err
	}
	if len(ca.TrustCABundlePEM) > ejbcaMaxPEMLength {
		return fleet.NewInvalidArgumentError("trust_ca_bundle", fmt.Sprintf("%sThe trust CA bundle is too large.", errPrefix))
	}

	creds, err := readEJBCAClientP12(ctx, ca.ClientP12, ca.Password, errPrefix)
	if err != nil {
		return err
	}
	ca.MTLSCertPEM, ca.MTLSKeyPEM = creds.CertPEM, creds.KeyPEM

	return svc.verifyEJBCA(ctx, *ca, errPrefix)
}

// validateEJBCAFields checks the fields that make up an EJBCA CA's enrollment settings, whether it is new or the result of an update.
func validateEJBCAFields(ca *fleet.EJBCA, errPrefix string) error {
	for _, field := range []struct{ name, label, value string }{
		{"certificate_authority_name", "EJBCA CA name", ca.CertificateAuthorityName},
		{"certificate_profile_name", "certificate profile name", ca.CertificateProfileName},
		{"end_entity_profile_name", "end entity profile name", ca.EndEntityProfileName},
		{"certificate_common_name", "common name", ca.CertificateCommonName},
	} {
		if strings.TrimSpace(field.value) == "" {
			return fleet.NewInvalidArgumentError(field.name, fmt.Sprintf("%sThe %s cannot be empty.", errPrefix, field.label))
		}
	}
	if ca.UsernameTemplate != "" && strings.TrimSpace(ca.UsernameTemplate) == "" {
		return fleet.NewInvalidArgumentError("username_template", errPrefix+"The username template cannot be only spaces.")
	}
	for _, field := range []struct{ name, label, value string }{
		{"certificate_authority_name", "EJBCA CA name", ca.CertificateAuthorityName},
		{"certificate_profile_name", "certificate profile name", ca.CertificateProfileName},
		{"end_entity_profile_name", "end entity profile name", ca.EndEntityProfileName},
		{"certificate_common_name", "common name", ca.CertificateCommonName},
		{"username_template", "username template", ca.UsernameTemplate},
	} {
		if utf8.RuneCountInString(field.value) > ejbcaMaxFieldLength {
			return fleet.NewInvalidArgumentError(field.name,
				fmt.Sprintf("%sThe %s cannot be longer than %d characters.", errPrefix, field.label, ejbcaMaxFieldLength))
		}
	}

	if err := validateCAFleetVariables("certificate_common_name", "CA Common Name (CN)", ca.CertificateCommonName,
		fleet.FleetVarsSupportedInEJBCA, errPrefix); err != nil {
		return err
	}
	if err := validateCAFleetVariables("username_template", "EJBCA Username Template", ca.UsernameTemplate, fleet.FleetVarsSupportedInEJBCA,
		errPrefix); err != nil {
		return err
	}
	return validateCAUserPrincipalNames(ca.CertificateUserPrincipalNames, "EJBCA", fleet.FleetVarsSupportedInEJBCA, ejbcaMaxFieldLength, errPrefix)
}

// readEJBCAClientP12 reads the client certificate chain and key out of a base64 PKCS#12 and rejects a certificate that isn't valid now.
func readEJBCAClientP12(ctx context.Context, p12Base64, password, errPrefix string) (*ejbca.ClientCredentials, error) {
	if strings.TrimSpace(p12Base64) == "" {
		return nil, fleet.NewInvalidArgumentError("client_p12", fmt.Sprintf("%sThe client certificate PKCS#12 cannot be empty.", errPrefix))
	}
	if password == "" {
		return nil, fleet.NewInvalidArgumentError("password", fmt.Sprintf("%sThe PKCS#12 password cannot be empty.", errPrefix))
	}
	p12, err := base64.StdEncoding.DecodeString(strings.TrimSpace(p12Base64))
	if err != nil {
		return nil, fleet.NewInvalidArgumentError("client_p12", fmt.Sprintf("%sThe client certificate PKCS#12 must be base64 encoded.", errPrefix))
	}
	if len(p12) > ejbcaMaxP12Size {
		return nil, fleet.NewInvalidArgumentError("client_p12", fmt.Sprintf("%sThe client certificate PKCS#12 is too large.", errPrefix))
	}

	creds, err := ejbca.ParseClientP12(ctx, p12, password)
	if err != nil {
		field := "client_p12"
		if errors.Is(err, ejbca.ErrP12Password) {
			field = "password"
		}
		return nil, fleet.NewInvalidArgumentError(field, errPrefix+err.Error())
	}
	if len(creds.CertPEM) > ejbcaMaxPEMLength {
		return nil, fleet.NewInvalidArgumentError("client_p12", fmt.Sprintf("%sThe client certificate chain is too large.", errPrefix))
	}
	now := time.Now()
	if now.After(creds.Leaf.NotAfter) {
		return nil, fleet.NewInvalidArgumentError("client_p12", fmt.Sprintf(
			"%sThe client certificate expired on %s. Upload a client certificate that hasn't expired.",
			errPrefix, creds.Leaf.NotAfter.UTC().Format(time.DateOnly)))
	}
	if now.Before(creds.Leaf.NotBefore) {
		return nil, fleet.NewInvalidArgumentError("client_p12", fmt.Sprintf(
			"%sThe client certificate isn't valid until %s. Upload a client certificate that is valid now.",
			errPrefix, creds.Leaf.NotBefore.UTC().Format(time.DateOnly)))
	}
	return creds, nil
}

// verifyEJBCA checks that EJBCA accepts the client certificate and knows the CA and certificate profile.
func (svc *Service) verifyEJBCA(ctx context.Context, ca fleet.EJBCA, errPrefix string) error {
	if err := svc.ejbcaService.VerifyConnection(ctx, ca); err != nil {
		svc.logger.InfoContext(ctx, "Failed to verify the EJBCA connection", "err", err)
		return &fleet.BadRequestError{Message: errPrefix + err.Error()}
	}
	return nil
}

// updateEJBCA validates an update against the stored CA and returns the changes to store. Fields the update doesn't set keep their stored
// values, and the result is validated as a whole. A new client certificate only replaces the stored one after EJBCA accepts it.
func (svc *Service) updateEJBCA(
	ctx context.Context, u *fleet.EJBCAUpdatePayload, oldCA *fleet.CertificateAuthority, errPrefix string,
) (*fleet.CertificateAuthority, error) {
	u.Preprocess()
	// A masked value is what a read returns for a secret, so it means "keep the stored one".
	if u.ClientP12 != nil && *u.ClientP12 == fleet.MaskedPassword {
		u.ClientP12 = nil
	}
	if u.Password != nil && *u.Password == fleet.MaskedPassword {
		u.Password = nil
	}
	if u.IsEmpty() {
		return nil, &fleet.BadRequestError{Message: fmt.Sprintf("%sEJBCA CA update payload is empty", errPrefix)}
	}
	if u.ClientP12 != nil && u.Password == nil {
		return nil, fleet.NewInvalidArgumentError("password", fmt.Sprintf("%sThe PKCS#12 password must be set with client_p12.", errPrefix))
	}
	if u.ClientP12 == nil && u.Password != nil {
		return nil, fleet.NewInvalidArgumentError("client_p12", fmt.Sprintf("%sThe password can only be set with client_p12.", errPrefix))
	}

	merged := oldCA.EJBCA()
	overrides := []struct {
		field  *string
		update *string
	}{
		{&merged.Name, u.Name},
		{&merged.URL, u.URL},
		{&merged.TrustCABundlePEM, u.TrustCABundle},
		{&merged.CertificateAuthorityName, u.CertificateAuthorityName},
		{&merged.CertificateProfileName, u.CertificateProfileName},
		{&merged.EndEntityProfileName, u.EndEntityProfileName},
		{&merged.CertificateCommonName, u.CertificateCommonName},
		{&merged.UsernameTemplate, u.UsernameTemplate},
	}
	for _, o := range overrides {
		if o.update != nil {
			*o.field = *o.update
		}
	}
	if u.CertificateUserPrincipalNames != nil {
		merged.CertificateUserPrincipalNames = *u.CertificateUserPrincipalNames
	}

	if u.Name != nil {
		if err := validateCAName(merged.Name, errPrefix); err != nil {
			return nil, err
		}
	}
	if u.URL != nil {
		if err := validateURL(merged.URL, "EJBCA", errPrefix); err != nil {
			return nil, err
		}
	}
	if err := validateEJBCAFields(&merged, errPrefix); err != nil {
		return nil, err
	}
	if len(merged.TrustCABundlePEM) > ejbcaMaxPEMLength {
		return nil, fleet.NewInvalidArgumentError("trust_ca_bundle", fmt.Sprintf("%sThe trust CA bundle is too large.", errPrefix))
	}

	caToUpdate := &fleet.CertificateAuthority{
		Type:                          string(fleet.CATypeEJBCA),
		Name:                          u.Name,
		URL:                           u.URL,
		TrustCABundlePEM:              u.TrustCABundle,
		EJBCACAName:                   u.CertificateAuthorityName,
		EJBCACertificateProfileName:   u.CertificateProfileName,
		EJBCAEndEntityProfileName:     u.EndEntityProfileName,
		CertificateCommonName:         u.CertificateCommonName,
		EJBCAUsernameTemplate:         u.UsernameTemplate,
		CertificateUserPrincipalNames: u.CertificateUserPrincipalNames,
	}
	if u.ClientP12 != nil && u.Password != nil { // the checks above leave both set or neither
		creds, err := readEJBCAClientP12(ctx, *u.ClientP12, *u.Password, errPrefix)
		if err != nil {
			return nil, err
		}
		merged.MTLSCertPEM, merged.MTLSKeyPEM = creds.CertPEM, creds.KeyPEM
		caToUpdate.MTLSCertPEM, caToUpdate.MTLSKeyPEM = &creds.CertPEM, &creds.KeyPEM
	}

	// The end entity profile, templates, and name can't be checked against EJBCA, so changing only them doesn't need the connection.
	if u.URL != nil || u.ClientP12 != nil || u.TrustCABundle != nil || u.CertificateAuthorityName != nil || u.CertificateProfileName != nil {
		if err := svc.verifyEJBCA(ctx, merged, errPrefix); err != nil {
			return nil, err
		}
	}
	return caToUpdate, nil
}
