package service

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/assets"
	"github.com/smallstep/scep"
)

////////////////////////////////////////////////////////////////////////////////
// POST /api/v1/fleet/certificate
////////////////////////////////////////////////////////////////////////////////

type certificateRequest struct {
	CSR          string `json:"csr"`
	SessionToken string `json:"session_token"`
}

type certificateResponse struct {
	Certificate string `json:"certificate"`
	Err         error  `json:"error,omitempty"`
}

func (r certificateResponse) Error() error { return r.Err }

func certificateEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*certificateRequest)
	s := svc.(*Service)
	cert, err := s.generateCertificate(ctx, req.CSR, req.SessionToken)
	if err != nil {
		return certificateResponse{Err: err}, nil
	}
	return certificateResponse{Certificate: cert}, nil
}

// generateCertificate generates a new PKCS #7 certificate from a CSR
func (svc *Service) generateCertificate(ctx context.Context, csrStr string, sessionToken string) (string, error) {
	// Verify the user is authenticated
	_, ok := viewer.FromContext(ctx)
	if !ok {
		return "", errors.New("could not fetch user from context")
	}

	// Verify the session token is valid
	if err := svc.ValidateSessionToken(ctx, sessionToken); err != nil {
		return "", ctxerr.Wrap(ctx, err, "validating session token")
	}

	// Decode the CSR from base64
	csrBytes, err := base64.StdEncoding.DecodeString(csrStr)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "decoding CSR")
	}

	// Parse the CSR
	block, _ := pem.Decode(csrBytes)
	if block == nil || block.Type != "CERTIFICATE REQUEST" {
		return "", ctxerr.Wrap(ctx, errors.New("invalid CSR format"), "parsing CSR")
	}

	csr, err := x509.ParseCertificateRequest(block.Bytes)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "parsing CSR")
	}

	// Verify the CSR signature
	if err := csr.CheckSignature(); err != nil {
		return "", ctxerr.Wrap(ctx, err, "verifying CSR signature")
	}

	// Get the CA certificate and key
	tlsCert, err := assets.CAKeyPair(ctx, svc.ds)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "getting CA certificate")
	}

	// Create a CSR request message
	csrReqMsg := &scep.CSRReqMessage{
		CSR: csr,
		// Add any additional fields needed for the CSR request
	}

	// Sign the CSR
	signedCert, err := svc.signCSR(ctx, csrReqMsg, tlsCert)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "signing CSR")
	}

	// Create a PKCS7 certificate
	p7SignedData, err := scep.DegenerateCertificates([]*x509.Certificate{signedCert})
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "creating PKCS7 certificate")
	}

	// Encode the certificate as base64
	certBase64 := base64.StdEncoding.EncodeToString(p7SignedData)

	return certBase64, nil
}

// ValidateSessionToken validates a session token
func (svc *Service) ValidateSessionToken(ctx context.Context, token string) error {
	// Implement session token validation logic
	// This could involve checking the token against a database of valid tokens
	// For now, we'll just check if the token is not empty
	if token == "" {
		return errors.New("empty session token")
	}

	// In a real implementation, you would verify the token against your session store
	// For example:
	// session, err := svc.ds.SessionByToken(ctx, token)
	// if err != nil {
	//     return err
	// }
	// if session.Expired() {
	//     return errors.New("session expired")
	// }

	return nil
}

// signCSR signs a certificate signing request
func (svc *Service) signCSR(ctx context.Context, csrReqMsg *scep.CSRReqMessage, cert *tls.Certificate) (*x509.Certificate, error) {
	// Generate a random serial number
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serial, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, fmt.Errorf("generating serial number: %w", err)
	}

	// Create a certificate template
	template := &x509.Certificate{
		SerialNumber:       serial,
		Subject:            csrReqMsg.CSR.Subject,
		PublicKey:          csrReqMsg.CSR.PublicKey,
		PublicKeyAlgorithm: csrReqMsg.CSR.PublicKeyAlgorithm,
		SignatureAlgorithm: x509.SHA256WithRSA,
		NotBefore:          cert.Leaf.NotBefore,
		NotAfter:           cert.Leaf.NotAfter,
		KeyUsage:           x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:        []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}

	// Create the certificate
	certBytes, err := x509.CreateCertificate(
		rand.Reader,
		template,
		cert.Leaf,
		csrReqMsg.CSR.PublicKey,
		cert.PrivateKey,
	)
	if err != nil {
		return nil, fmt.Errorf("creating certificate: %w", err)
	}

	// Parse the certificate
	signedCert, err := x509.ParseCertificate(certBytes)
	if err != nil {
		return nil, fmt.Errorf("parsing certificate: %w", err)
	}

	return signedCert, nil
}
