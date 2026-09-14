package service

import (
	"context"
	"crypto/x509"
	"encoding/asn1"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/ee/server/service/hostidentity/httpsig"
	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/contexts/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/smallstep/pkcs7"
)

// This code largely adapted from fleet/website/api/controllers/get-est-device-certificate.js
func (svc *Service) RequestCertificate(ctx context.Context, p fleet.RequestCertificatePayload) (*string, error) {
	auth, authOk := authz.FromContext(ctx)
	if !authOk {
		// This shouldn't be possible
		return nil, &fleet.BadRequestError{Message: "Missing authentication authorization context"}
	}

	var hostID *uint

	if auth.AuthnMethod() == authz.AuthnHTTPMessageSignature {
		// Device-auth path
		svc.authz.SkipAuthorization(ctx)

		hostIdentityCert, certOk := httpsig.FromContext(ctx)
		if !certOk {
			return nil, fleet.NewPermissionError("Missing host identity certificate for signed certificate request.")
		}
		if hostIdentityCert.HostID == nil {
			return nil, fleet.NewPermissionError("Host identity certificate is not associated with an enrolled host.")
		}
		hostID = hostIdentityCert.HostID

	} else if err := svc.authz.Authorize(ctx, &fleet.RequestCertificatePayload{}, fleet.ActionWrite); err != nil {
		// User-based auth path
		return nil, err
	}

	ca, err := svc.ds.GetCertificateAuthorityByID(ctx, p.ID, true)
	if err != nil {
		return nil, err
	}
	if ca.Type != string(fleet.CATypeHydrant) && ca.Type != string(fleet.CATypeCustomESTProxy) {
		return nil, &fleet.BadRequestError{Message: "This API currently only supports Hydrant and EST Certificate Authorities."}
	}
	if ca.Type == string(fleet.CATypeHydrant) {
		if ca.ClientID == nil {
			return nil, &fleet.BadRequestError{Message: "Certificate authority does not have a username configured."}
		}
		if ca.ClientSecret == nil {
			return nil, &fleet.BadRequestError{Message: "Certificate authority does not have a client secret configured."}
		}
	}
	if ca.Type == string(fleet.CATypeCustomESTProxy) {
		if ca.Username == nil {
			return nil, &fleet.BadRequestError{Message: "Certificate authority does not have a username configured."}
		}
		if ca.Password == nil {
			return nil, &fleet.BadRequestError{Message: "Certificate authority does not have a password configured."}
		}
	}

	certificateRequest, err := svc.parseCSR(ctx, p.CSR)
	if err != nil {
		svc.logger.ErrorContext(ctx, "Failed to parse CSR during certificate request", "err", err)
		return nil, InvalidCSRError{}
	}

	appConfig, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "loading app config for certificate request")
	}

	idpProvided, err := appConfig.Integrations.CheckCertIdPIntrospection(p.IDPOauthURL, p.IDPToken, p.IDPClientID)
	if err != nil {
		return nil, err
	}

	// Both identity checks bind to the CSR's single email address and to the UPN it must agree
	// with. A CSR missing either cannot be bound, so it is refused rather than partly checked.
	var csrEmail, csrUsername string
	bindHost := appConfig.Integrations.CertificatesRequireHostEndUserBinding.Value && hostID != nil

	if bindHost || idpProvided {
		csrEmail, csrUsername, err = svc.extractCSRUserInfo(ctx, certificateRequest)
		if err != nil {
			svc.logger.ErrorContext(ctx, "CSR did not have expected format for identity verification", "err", err)
			return nil, InvalidCSRError{}
		}
	}

	// Runs before introspection: a pure DB lookup, so a request that cannot pass it never reaches
	// the network.
	if bindHost {
		if err := svc.verifyHostEndUserBinding(ctx, *hostID, csrEmail, csrUsername); err != nil {
			svc.logger.ErrorContext(ctx, "Failing Certificate Request due to host end user binding", "host_id", *hostID, "err", err)
			return nil, err
		}
	}

	idpUsername := ""
	if idpProvided {
		introspectionResponse, err := svc.introspectIDPToken(ctx, *p.IDPClientID, *p.IDPToken, *p.IDPOauthURL)
		if err != nil {
			svc.logger.ErrorContext(ctx, "Failed to introspect IDP token during certificate request", "idp_url", *p.IDPOauthURL, "err", err)
			return nil, InvalidIDPTokenError{}
		}
		if !introspectionResponse.Active {
			svc.logger.ErrorContext(ctx, "Failing Certificate Request due to inactive IDP token", "idp_url", *p.IDPOauthURL)
			return nil, InvalidIDPTokenError{}
		}
		// This field is technically optional in the spec though its omittance may indicate an incompatible IDP or setup
		if introspectionResponse.Username == nil || len(*introspectionResponse.Username) == 0 {
			svc.logger.ErrorContext(ctx, "Failing Certificate Request due to missing username in IDP token introspection response")
			return nil, InvalidIDPTokenError{}
		}

		idpUsername = *introspectionResponse.Username

		// the email should either equal the username or include it as a prefix, i.e.
		// email=username@example.com and username=username
		if !strings.HasPrefix(csrEmail, csrUsername) {
			svc.logger.ErrorContext(ctx, "Failing Certificate Request due to mismatch between CSR email and UPN", "csr_email", csrEmail, "csr_upn", csrUsername)
			return nil, InvalidCSRError{}
		}
		if csrEmail != *introspectionResponse.Username {
			svc.logger.ErrorContext(ctx, "Failing Certificate Request due to mismatch between CSR email and IDP token username", "csr_email", csrEmail, "idp_username", *introspectionResponse.Username)
			// The email in the CSR must match the username from the IDP token introspection
			return nil, InvalidIDPTokenError{}
		}
	}

	csrForRequest := strings.ReplaceAll(p.CSR, "-----BEGIN CERTIFICATE REQUEST-----", "")
	csrForRequest = strings.ReplaceAll(csrForRequest, "-----END CERTIFICATE REQUEST-----", "")
	csrForRequest = strings.ReplaceAll(csrForRequest, "\\n", "")

	var estCA fleet.ESTProxyCA
	if ca.Type == string(fleet.CATypeHydrant) {
		estCA = fleet.ESTProxyCA{
			Name:     *ca.Name,
			URL:      *ca.URL,
			Username: *ca.ClientID,
			Password: *ca.ClientSecret,
		}
	} else {
		estCA = fleet.ESTProxyCA{
			Name:     *ca.Name,
			URL:      *ca.URL,
			Username: *ca.Username,
			Password: *ca.Password,
		}
	}

	certificate, err := svc.estService.GetCertificate(ctx, estCA, csrForRequest) //nolint (staticheck bug)
	if err != nil {
		svc.logger.ErrorContext(ctx, "EST certificate request failed", "ca_id", ca.ID, "err", err)
		// Bad request may seem like a strange error here but there are many cases where a malformed
		// CSR can cause this error and Hydrant's API often returns a 5XX error even in these cases
		// so it is not always possible to distinguish between an error caused by a bad request or
		// an internal CA error.
		return nil, &fleet.BadRequestError{Message: fmt.Sprintf("EST certificate request failed: %s", err.Error())}
	}
	svc.logger.InfoContext(ctx, "Successfully retrieved a certificate from EST", "ca_id", ca.ID, "idp_username", idpUsername)

	if p.ReturnPEMCertificate {
		pemCert, err := pkcs7EnvelopeToPEM(certificate.Certificate)
		if err != nil {
			svc.logger.ErrorContext(ctx, "Failed to convert PKCS7 envelope to PEM certificate", "ca_id", ca.ID, "err", err)
			return nil, ctxerr.Wrap(ctx, err, "converting PKCS7 envelope to PEM certificate")
		}
		return &pemCert, nil
	}

	// Wrap the certificate in a PEM block for easier consumption by the client. TODO: If we ever
	// support CAs other than Hydrant/EST in this API, this may need to be modified to be aware of
	// their formats.
	return new("-----BEGIN PKCS7-----\n" + string(certificate.Certificate) + "\n-----END PKCS7-----\n"), nil
}

// verifyHostEndUserBinding requires the CSR to name the identity the calling host is recorded as
// belonging to. It fails closed: a host with no recorded IdP username is rejected, which is what
// stops a host enrolled with a leaked enroll secret even when it holds a valid stolen token.
func (svc *Service) verifyHostEndUserBinding(ctx context.Context, hostID uint, csrEmail, csrUsername string) error {
	mismatch := fleet.NewPermissionError("Certificate subject does not match the end user identity recorded for this host.")

	// The UPN is a SAN entry independent of the email, and it is the field 802.1X and AD-backed
	// mTLS authenticate on, so binding the email alone would still let a caller name a victim
	// there. Require the relationship the IdP path documents: the UPN is the email or a shorthand
	// prefix of it, which binds the UPN transitively once the email is bound below.
	if !strings.HasPrefix(strings.ToLower(csrEmail), strings.ToLower(csrUsername)) {
		return mismatch
	}

	endUsers, err := fleet.GetEndUsers(ctx, svc.ds, hostID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "getting end users for host end user binding")
	}
	// GetEndUsers returns at most one record, the SCIM one where it exists and otherwise one built
	// from device mapping, so index zero is the host's identity. Same shape as the Fleet variable
	// expansion in certificate_templates.go and profile_variables.go.
	if len(endUsers) == 0 || endUsers[0].IdpUserName == "" {
		return mismatch
	}
	// Case-insensitive, as IdPs treat usernames and as SetHostDeviceMapping compares this field.
	if !strings.EqualFold(endUsers[0].IdpUserName, csrEmail) {
		return mismatch
	}
	return nil
}

// pkcs7EnvelopeToPEM converts a base64-encoded PKCS7 envelope (as returned by an EST
// /simpleenroll response, per RFC 7030) into a single PEM-encoded CERTIFICATE block.
// It returns an error unless the envelope contains exactly one certificate.
func pkcs7EnvelopeToPEM(envelope []byte) (string, error) {
	// EST returns base64-encoded PKCS7 with potential whitespace; strip it before decoding.
	stripped := strings.Map(func(r rune) rune {
		if r == '\n' || r == '\r' || r == ' ' || r == '\t' {
			return -1
		}
		return r
	}, string(envelope))

	derBytes, err := base64.StdEncoding.DecodeString(stripped)
	if err != nil {
		return "", fmt.Errorf("decoding base64 PKCS7 envelope: %w", err)
	}

	p7, err := pkcs7.Parse(derBytes)
	if err != nil {
		return "", fmt.Errorf("parsing PKCS7 envelope: %w", err)
	}
	// Per RFC 7030 §4.2.3, the EST /simpleenroll SimplePKIResponse carries the single
	// issued certificate. Reject anything else so callers don't have to guess which cert
	// is the leaf.
	if len(p7.Certificates) != 1 {
		return "", fmt.Errorf("expected exactly 1 certificate in EST PKCS7 envelope, got %d", len(p7.Certificates))
	}

	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: p7.Certificates[0].Raw})
	if pemBytes == nil {
		return "", errors.New("encoding certificate to PEM")
	}
	return string(pemBytes), nil
}

func (svc *Service) introspectIDPToken(ctx context.Context, idpClientID, idpToken, idpOauthURL string) (*oauthIntrospectionResponse, error) {
	httpClient := fleethttp.NewClient(fleethttp.WithTimeout(20 * time.Second))
	introspectionRequest := url.Values{
		"client_id": []string{idpClientID},
		"token":     []string{idpToken},
	}
	introspectionBody := introspectionRequest.Encode()
	req, err := http.NewRequestWithContext(ctx, "POST", idpOauthURL, strings.NewReader(introspectionBody))
	if err != nil {
		return nil, ctxerr.Wrapf(ctx, err, "Failed to create introspection request")
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, ctxerr.Wrapf(ctx, err, "Failed to introspect IDP token")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, ctxerr.Errorf(ctx, "IDP token introspection failed with status code %d", resp.StatusCode)
	}

	oauthIntrospectionResponse := &oauthIntrospectionResponse{}
	if err := json.NewDecoder(resp.Body).Decode(oauthIntrospectionResponse); err != nil {
		return nil, ctxerr.Wrapf(ctx, err, "Failed to decode IDP token introspection response")
	}

	return oauthIntrospectionResponse, nil
}

func (svc *Service) parseCSR(ctx context.Context, csr string) (*x509.CertificateRequest, error) {
	// unescape newlines
	block, _ := pem.Decode([]byte(strings.ReplaceAll(csr, "\\n", "\n")))
	if block == nil {
		return nil, ctxerr.New(ctx, "invalid CSR format")
	}

	req, err := x509.ParseCertificateRequest(block.Bytes)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "failed to parse CSR")
	}

	return req, nil
}

// Extract email and UPN fields from the provided CSR. Assumes there is exactly 1 email and that there is a UPN SAN extension, will
// error otherwise. More than one email is rejected rather than picked between: every identity
// check binds to this address, so an ambiguous CSR has no answer.
func (svc *Service) extractCSRUserInfo(ctx context.Context, req *x509.CertificateRequest) (string, string, error) {
	if len(req.EmailAddresses) < 1 {
		return "", "", ctxerr.New(ctx, "CSR does not contain an email address")
	}
	if len(req.EmailAddresses) > 1 {
		return "", "", ctxerr.Errorf(ctx, "CSR contains %d email addresses, only 1 is supported", len(req.EmailAddresses))
	}

	upn, err := extractCSRUPN(ctx, req)
	if err != nil {
		return "", "", ctxerr.Wrap(ctx, err, "failed to extract UPN from CSR")
	}

	return req.EmailAddresses[0], upn, nil
}

// The go standard library does not provide a way to extract the UPN from a CSR, so we must do it
// manually by first finding the SAN extension then looking in othernames for the UPN and parsing it.
func extractCSRUPN(ctx context.Context, csr *x509.CertificateRequest) (string, error) {
	sanOID := asn1.ObjectIdentifier{2, 5, 29, 17}
	upnOID := asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 311, 20, 2, 3}
	for _, ext := range csr.Extensions {
		if ext.Id.Equal(sanOID) {
			nameValues := []asn1.RawValue{}
			if _, err := asn1.Unmarshal(ext.Value, &nameValues); err != nil {
				return "", fmt.Errorf("failed to unmarshal SAN extension: %w", err)
			}
			for _, names := range nameValues {
				// We are looking for the othernames(tag 0) in the SAN extension
				if names.Tag == 0 {
					var oid asn1.ObjectIdentifier
					var rawValue asn1.RawValue
					var err error
					remainingBytes := names.Bytes
					// This will be a sequence of OID-value pairs that we must parse
					for len(remainingBytes) > 0 {
						remainingBytes, err = asn1.Unmarshal(remainingBytes, &oid)
						if err != nil {
							return "", fmt.Errorf("failed to unmarshal othername OID: %w", err)
						}
						// I am not sure what this would indicate. Perhaps a malformed CSR?
						if len(remainingBytes) == 0 {
							return "", fmt.Errorf("unexpected end of input bytes after unmarshalling othername OID %s but before unmarshaling value", oid.String())
						}
						remainingBytes, err = asn1.Unmarshal(remainingBytes, &rawValue)
						if err != nil {
							return "", fmt.Errorf("failed to unmarshal othername value: %w", err)
						}
						if oid.Equal(upnOID) {
							// Unmarshal the raw value into a string
							var upn asn1.RawValue
							if _, err := asn1.Unmarshal(rawValue.Bytes, &upn); err != nil {
								return "", fmt.Errorf("failed to unmarshal UPN value: %w", err)
							}
							return string(upn.Bytes), nil
						}
					}
				}
			}
		}
	}
	return "", ctxerr.New(ctx, "CSR does not contain a UPN")
}
