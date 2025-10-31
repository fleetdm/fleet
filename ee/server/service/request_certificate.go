package service

import (
	"context"
	"crypto/x509"
	"encoding/asn1"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/go-kit/log/level"
)

// This code largely adapted from fleet/website/api/controllers/get-est-device-certificate.js
func (svc *Service) RequestCertificate(ctx context.Context, p fleet.RequestCertificatePayload) (*string, error) {
	if err := svc.authz.Authorize(ctx, &fleet.RequestCertificatePayload{}, fleet.ActionWrite); err != nil {
		return nil, err
	}
	ca, err := svc.ds.GetCertificateAuthorityByID(ctx, p.ID, true)
	if err != nil {
		return nil, err
	}
	if ca.Type != string(fleet.CATypeHydrant) && ca.Type != string(fleet.CATypeCustomESTProxy) {
		return nil, &fleet.BadRequestError{Message: "This API currently only supports Hydrant and EST Certificate Authorities."}
	}
	if ca.Type == string(fleet.CATypeHydrant) && ca.ClientSecret == nil {
		return nil, &fleet.BadRequestError{Message: "Certificate authority does not have a client secret configured."}
	}
	if ca.Type == string(fleet.CATypeCustomESTProxy) && ca.Password == nil {
		return nil, &fleet.BadRequestError{Message: "Certificate authority does not have a password configured."}
	}
	certificateRequest, err := svc.parseCSR(ctx, p.CSR)
	if err != nil {
		level.Error(svc.logger).Log("msg", "Failed to parse CSR during certificate request", "err", err)
		return nil, InvalidCSRError{}
	}

	idpUsername := ""
	if p.IDPClientID != nil || p.IDPToken != nil || p.IDPOauthURL != nil {
		if p.IDPClientID == nil || p.IDPToken == nil || p.IDPOauthURL == nil {
			return nil, &fleet.BadRequestError{Message: "IDP Client ID, Token, and OAuth URL all must be provided, if any are provided when requesting a certificate."}
		}

		csrEmail, csrUsername, err := svc.extractCSRUserInfo(ctx, certificateRequest)
		if err != nil {
			level.Error(svc.logger).Log("msg", "CSR did not have expected format for IDP verification", "err", err)
			return nil, InvalidCSRError{}
		}

		introspectionResponse, err := svc.introspectIDPToken(ctx, *p.IDPClientID, *p.IDPToken, *p.IDPOauthURL)
		if err != nil {
			level.Error(svc.logger).Log("msg", "Failed to introspect IDP token during certificate request", "idp_url", *p.IDPOauthURL, "err", err)
			return nil, InvalidIDPTokenError{}
		}
		if !introspectionResponse.Active {
			level.Error(svc.logger).Log("msg", "Failing Certificate Request due to inactive IDP token", "idp_url", *p.IDPOauthURL)
			return nil, InvalidIDPTokenError{}
		}
		// This field is technically optional in the spec though its omittance may indicate an incompatible IDP or setup
		if introspectionResponse.Username == nil || len(*introspectionResponse.Username) == 0 {
			level.Error(svc.logger).Log("msg", "Failing Certificate Request due to missing username in IDP token introspection response")
			return nil, InvalidIDPTokenError{}
		}

		idpUsername = *introspectionResponse.Username

		// the email should either equal the username or include it as a prefix, i.e.
		// email=username@example.com and username=username
		if !strings.HasPrefix(csrEmail, csrUsername) {
			level.Error(svc.logger).Log("msg", "Failing Certificate Request due to mismatch between CSR email and UPN", "csr_email", csrEmail, "csr_upn", csrUsername)
			return nil, InvalidCSRError{}
		}
		if csrEmail != *introspectionResponse.Username {
			level.Error(svc.logger).Log("msg", "Failing Certificate Request due to mismatch between CSR email and IDP token username", "csr_email", csrEmail, "idp_username", *introspectionResponse.Username)
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

	certificate, err := svc.estService.GetCertificate(ctx, estCA, csrForRequest)
	if err != nil {
		level.Error(svc.logger).Log("msg", "EST certificate request failed", "ca_id", ca.ID, "error", err)
		// Bad request may seem like a strange error here but there are many cases where a malformed
		// CSR can cause this error and Hydrant's API often returns a 5XX error even in these cases
		// so it is not always possible to distinguish between an error caused by a bad request or
		// an internal CA error.
		return nil, &fleet.BadRequestError{Message: fmt.Sprintf("EST certificate request failed: %s", err.Error())}
	}
	level.Info(svc.logger).Log("msg", "Successfully retrieved a certificate from EST", "ca_id", ca.ID, "idp_username", idpUsername)
	// Wrap the certificate in a PEM block for easier consumption by the client
	return ptr.String("-----BEGIN CERTIFICATE-----\n" + string(certificate.Certificate) + "\n-----END CERTIFICATE-----\n"), nil
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
// error otherwise
func (svc *Service) extractCSRUserInfo(ctx context.Context, req *x509.CertificateRequest) (string, string, error) {
	if len(req.EmailAddresses) < 1 {
		return "", "", ctxerr.New(ctx, "CSR does not contain an email address")
	}

	if len(req.EmailAddresses) > 1 {
		return "", "", ctxerr.Errorf(ctx, "CSR contains %d email addresses, only 1 is supported", len(req.EmailAddresses))
	}
	csrEmail := req.EmailAddresses[0]

	upn, err := extractCSRUPN(req)
	if err != nil {
		return "", "", ctxerr.Wrap(ctx, err, "failed to extract UPN from CSR")
	}
	if upn == nil {
		return "", "", ctxerr.New(ctx, "CSR does not contain a UPN")
	}

	return csrEmail, *upn, nil
}

// The go standard library does not provide a way to extract the UPN from a CSR, so we must do it
// manually by first finding the SAN extension then looking in othernames for the UPN and parsing it.
func extractCSRUPN(csr *x509.CertificateRequest) (*string, error) {
	sanOID := asn1.ObjectIdentifier{2, 5, 29, 17}
	upnOID := asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 311, 20, 2, 3}
	for _, ext := range csr.Extensions {
		if ext.Id.Equal(sanOID) {
			nameValues := []asn1.RawValue{}
			if _, err := asn1.Unmarshal(ext.Value, &nameValues); err != nil {
				return nil, fmt.Errorf("failed to unmarshal SAN extension: %w", err)
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
						remainingBytes, err = asn1.Unmarshal(names.Bytes, &oid)
						if err != nil {
							return nil, fmt.Errorf("failed to unmarshal othername OID: %w", err)
						}
						// I am not sure what this would indicate. Perhaps a malformed CSR?
						if len(remainingBytes) == 0 {
							return nil, fmt.Errorf("unexpected end of input bytes after unmarshalling othername OID %s but before unmarshaling value", oid.String())
						}
						remainingBytes, err = asn1.Unmarshal(remainingBytes, &rawValue)
						if err != nil {
							return nil, fmt.Errorf("failed to unmarshal othername value: %w", err)
						}
						if oid.Equal(upnOID) {
							// Unmarshal the raw value into a string
							var upn asn1.RawValue
							if _, err := asn1.Unmarshal(rawValue.Bytes, &upn); err != nil {
								return nil, fmt.Errorf("failed to unmarshal UPN value: %w", err)
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
