package service

import (
	"context"
	"encoding/base64"
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

				// Request certificate from DigiCert
				certData, err := svc.digiCertService.GetCertificate(ctx, ca)
				if err != nil {
					return "", err
				}

				// Return base64 encoded certificate
				return base64.StdEncoding.EncodeToString(certData.PfxData), nil
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
