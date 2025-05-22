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
	"io"
	"net/http"
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

////////////////////////////////////////////////////////////////////////////////
// POST /api/v1/fleet/certificates
////////////////////////////////////////////////////////////////////////////////

type requestCertificateRequest struct {
	CertificateAuthorityName  string `json:"certificate_authority_name"`
	CSR                       string `json:"csr"`
	CommonName                string `json:"common_name,omitempty"`
	UserPrincipalName         string `json:"user_principal_name,omitempty"`
	SubjectAlternativeName    string `json:"subject_alternative_name,omitempty"`
	UniformResourceIdentifier string `json:"uniform_resource_identifier,omitempty"`
	IDPToken                  string `json:"idp_token,omitempty"`
}

type requestCertificateResponse struct {
	Certificate string `json:"certificate"`
	Err         error  `json:"error,omitempty"`
}

func (r requestCertificateResponse) Error() error { return r.Err }

func requestCertificateEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*requestCertificateRequest)

	// Verify user is authenticated
	_, ok := viewer.FromContext(ctx)
	if !ok {
		return requestCertificateResponse{Err: fleet.ErrNoContext}, nil
	}

	// Request the certificate
	cert, err := svc.RequestCertificate(ctx, req.CertificateAuthorityName, req.CSR, req.CommonName,
		req.UserPrincipalName, req.SubjectAlternativeName, req.UniformResourceIdentifier, req.IDPToken)
	if err != nil {
		return requestCertificateResponse{Err: err}, nil
	}

	// Return the certificate
	return requestCertificateResponse{Certificate: cert}, nil
}

// RequestCertificate requests a certificate from the specified certificate authority
func (svc *Service) RequestCertificate(ctx context.Context, caName, csr, commonName,
	userPrincipalName, subjectAlternativeName, uniformResourceIdentifier, idpToken string,
) (string, error) {
	// Validate the certificate authority name
	if caName == "" {
		return "", fleet.NewInvalidArgumentError("certificate_authority_name", "Certificate authority name is required")
	}

	// Validate the CSR
	if csr == "" {
		return "", fleet.NewInvalidArgumentError("csr", "Certificate signing request (CSR) is required")
	}

	// Get the app config to check if the certificate authority exists
	appConfig, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return "", err
	}

	// Check if the certificate authority exists
	var caFound bool

	// Check DigiCert CAs
	if appConfig.Integrations.DigiCert.Valid {
		for _, ca := range appConfig.Integrations.DigiCert.Value {
			if ca.Name == caName {
				caFound = true

				// Get the API token
				asset, err := svc.ds.GetCAConfigAsset(ctx, ca.Name, fleet.CAConfigDigiCert)
				if err != nil {
					return "", err
				}

				ca.APIToken = string(asset.Value)

				// Request certificate from DigiCert using provided CSR
				certData, err := GetDigicertCertificate(ctx, caName, csr, commonName, userPrincipalName, subjectAlternativeName, uniformResourceIdentifier, idpToken, ca.APIToken)
				if err != nil {
					return "", err
				}
				// svc.digiCertService.GetCertificate

				// Return base64 encoded certificate
				return base64.StdEncoding.EncodeToString([]byte(certData)), nil
			}
		}
	}

	// Check Custom SCEP CAs
	if appConfig.Integrations.CustomSCEPProxy.Valid {
		for _, ca := range appConfig.Integrations.CustomSCEPProxy.Value {
			if ca.Name == caName {
				caFound = true

				// Get the challenge
				asset, err := svc.ds.GetCAConfigAsset(ctx, ca.Name, fleet.CAConfigCustomSCEPProxy)
				if err != nil {
					return "", err
				}

				ca.Challenge = string(asset.Value)

				// TODO: Implement SCEP certificate request
				return "", fmt.Errorf("requesting certificates from SCEP CAs is not yet implemented")
			}
		}
	}

	// Check NDES SCEP CA
	if appConfig.Integrations.NDESSCEPProxy.Valid && caName == "NDES" {
		caFound = true

		// TODO: Implement NDES SCEP certificate request
		return "", fmt.Errorf("requesting certificates from NDES SCEP CA is not yet implemented")
	}

	if !caFound {
		return "", fleet.NewInvalidArgumentError("certificate_authority_name",
			fmt.Sprintf("Certificate authority '%s' not found", caName))
	}

	return "", fmt.Errorf("unexpected error: certificate authority found but no implementation available")
}

// RequestCertificate requests a certificate from the specified certificate authority
func GetDigicertCertificate(ctx context.Context, caName, csr, commonName,
	userPrincipalName, subjectAlternativeName, uniformResourceIdentifier, idpToken, apiToken string,
) (string, error) {
	return "", nil
}

type IntrospectResponse struct {
	Active   bool   `json:"active"`
	Username string `json:"username"`
}

func GetDeviceCertificate(
	csrData, authToken, introspectEndpoint, idpClientId,
	estEndpoint, estClientId, estClientKey string,
) (string, error) {
	// Step 1: Token introspection
	form := fmt.Sprintf("client_id=%s&token=%s", idpClientId, authToken)
	resp, err := http.Post(introspectEndpoint, "application/x-www-form-urlencoded", strings.NewReader(form))
	if err != nil {
		return "", fmt.Errorf("introspection request failed: %w", err)
	}
	defer resp.Body.Close()

	var introspect IntrospectResponse
	if err := json.NewDecoder(resp.Body).Decode(&introspect); err != nil || !introspect.Active {
		return "", errors.New("invalidToken")
	}

	// Step 2: Decode CSR
	block, _ := pemDecode([]byte(csrData))
	if block == nil || block.Type != "CERTIFICATE REQUEST" {
		return "", errors.New("invalidCsr")
	}
	csr, err := x509.ParseCertificateRequest(block.Bytes)
	if err != nil {
		return "", errors.New("invalidCsr")
	}

	var csrEmail, csrUsername string
	for _, ext := range csr.Extensions {
		if ext.Id.Equal([]int{2, 5, 29, 17}) { // subjectAltName
			var rawValues asn1.RawValue
			if _, err := asn1.Unmarshal(ext.Value, &rawValues); err == nil {
				rest := rawValues.Bytes
				for len(rest) > 0 {
					var entry asn1.RawValue
					rest, _ = asn1.Unmarshal(rest, &entry)
					switch entry.Tag {
					case 1: // rfc822Name
						csrEmail = string(entry.Bytes)
					case 0: // otherName
						if strings.Contains(string(entry.FullBytes), "1.3.6.1.4.1.311.20.2.3") {
							csrUsername = extractUTF8FromASN1(entry.Bytes)
						}
					}
				}
			}
		}
	}

	if csrEmail == "" || !strings.HasPrefix(csrEmail, csrUsername) {
		return "", errors.New("invalidCsr")
	}
	if csrEmail != introspect.Username {
		return "", errors.New("invalidToken")
	}

	// Step 3: Send to EST endpoint
	csrClean := strings.ReplaceAll(csrData, "-----BEGIN CERTIFICATE REQUEST-----", "")
	csrClean = strings.ReplaceAll(csrClean, "-----END CERTIFICATE REQUEST-----", "")
	csrClean = strings.ReplaceAll(csrClean, "\n", "")

	req, err := http.NewRequest("POST", estEndpoint, strings.NewReader(csrClean))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/pkcs10")
	auth := base64.StdEncoding.EncodeToString([]byte(estClientId + ":" + estClientKey))
	req.Header.Set("Authorization", "Basic "+auth)

	client := &http.Client{}
	resp, err = client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	certBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	cert := "-----BEGIN CERTIFICATE-----\n" + string(certBody) + "\n-----END CERTIFICATE-----"
	return cert, nil
}

// Helpers
func pemDecode(data []byte) (*pem.Block, []byte) {
	return pem.Decode(data)
}

func extractUTF8FromASN1(data []byte) string {
	var v struct {
		OID   asn1.ObjectIdentifier
		Value asn1.RawValue
	}
	if _, err := asn1.Unmarshal(data, &v); err == nil && v.Value.Tag == 12 { // UTF8String
		return string(v.Value.Bytes)
	}
	return ""
}
