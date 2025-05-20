package service

import (
	"context"
	"fmt"

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

	// Only support Custom SCEP CAs for now
	if !appConfig.Integrations.CustomSCEPProxy.Valid {
		return "", fleet.NewInvalidArgumentError("certificate_authority_name",
			"No custom SCEP certificate authorities configured")
	}

	var caFound bool

	// Check Custom SCEP CAs
	for _, ca := range appConfig.Integrations.CustomSCEPProxy.Value {
		if ca.Name == caName {
			caFound = true

			// Get the challenge
			asset, err := svc.ds.GetCAConfigAsset(ctx, ca.Name, fleet.CAConfigCustomSCEPProxy)
			if err != nil {
				return "", err
			}

			ca.Challenge = string(asset.Value)

			// TODO: Implement actual SCEP certificate request
			// For now, return a placeholder certificate for testing
			// In a real implementation, we would:
			// 1. Parse the CSR
			// 2. Connect to the SCEP server using the URL and challenge
			// 3. Request a certificate
			// 4. Return the base64 encoded certificate

			// Placeholder implementation - returns a dummy certificate
			dummyCert := "-----BEGIN CERTIFICATE-----\nMIICfDCCAeWgAwIBAgIBADANBgkqhkiG9w0BAQsFADCBgDELMAkGA1UEBhMCVVMx\nEzARBgNVBAgMCldhc2hpbmd0b24xEDAOBgNVBAcMB1NlYXR0bGUxFTATBgNVBAoM\nDEZsZWV0IERldmljZTEUMBIGA1UECwwLRW5naW5lZXJpbmcxHTAbBgNVBAMMFEZs\nZWV0IFNDRVAgQ2VydGlmaWNhdGUwHhcNMjUwNTIwMTcwMDAwWhcNMjYwNTIwMTcw\nMDAwWjCBgDELMAkGA1UEBhMCVVMxEzARBgNVBAgMCldhc2hpbmd0b24xEDAOBgNV\nBAcMB1NlYXR0bGUxFTATBgNVBAoMDEZsZWV0IERldmljZTEUMBIGA1UECwwLRW5n\naW5lZXJpbmcxHTAbBgNVBAMMFEZsZWV0IFNDRVAgQ2VydGlmaWNhdGUwgZ8wDQYJ\nKoZIhvcNAQEBBQADgY0AMIGJAoGBALJZtbxathh+RfK+Z613ar4EYSIem8yAvv2J\nZJtopjD3noy1yF+nGRyF/ocm+FhYvjR5u7teJXlcv24tAAHuWL4UuPIql0Slakjd\nsfl098salkj324lkjmtElWDi6XRjUIXEj1zyCnZTCxGmyHcYB/+f3fyv/gZ8SkPq\nocNOCpX6cSW8hxOlaF9aZUC+xMHRdjQgxQ79hleb5K/n2gCJjiW1sV0EsRg+MX0c\nbPCpahpzlvIAkzA7TTUTOd7ZN+V0GW0fH86uMstrqeW2QUuZmSDC9fNyjQhk6n5i\nURaHXdFjSmyrhW5AVvw1nIblHodhUtD6J+g9kjhBg1frss3ndQtnNrnMAgMBAAEw\nDQYJKoZIhvcNAQELBQADgYEAH2U6Or14b4O22YjM22kXI9QDC5P+sDczcLjivv4M\nYXQL1ks8R6B1nXCrOmiLPPLaZ09f+UkeMnyuGAxW8Ce6LTKquwvlifZ+5TjyANz0\nI/d9ETLQF2MTphEZd4ySNLtq2RwYyDOBKaxMdW0sUsd6M3WyAuTBVgBkTVIqbMJB\nzFsgXSrr2a0LJEHszOO2BN3yT5muDQsKPJ1uXL7tNUv16pGaYpQZR8yGAmWyISHh\nAyLaJ1N1R8L77SLxdd/Sj7RunNNxqFqaEgIJMgsyu08GharLkQcIoW7qPHZuaLa5\n4xMF/s/vfKH6rgGbbCAgw9kw8Klt+6H3OH1FSMeRfZ/DWs=\n-----END CERTIFICATE-----"

			return dummyCert, nil
		}
	}

	if !caFound {
		return "", fleet.NewInvalidArgumentError("certificate_authority_name",
			fmt.Sprintf("Certificate authority '%s' not found", caName))
	}

	return "", fmt.Errorf("unexpected error: certificate authority found but no implementation available")
}
