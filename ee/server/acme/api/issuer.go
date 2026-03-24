package api

import (
	"context"
	"crypto/x509"
)

// CertificateIssuer is the interface that CA backends must implement.
//
// The ACME server manages the order/authz/challenge lifecycle and calls
// the issuer at two decision points:
//
//  1. ValidateChallenge — when a device responds to a challenge. The backend
//     decides whether validation passes.
//  2. IssueCertificate — when an order is ready to be finalized. The backend
//     signs the CSR and returns the certificate chain.
//
// Implementations:
//   - RelayBackend: relays to an upstream ACME CA (Hydrant, Sectigo, Smallstep)
//   - LocalCABackend: signs certificates with an embedded CA key
type CertificateIssuer interface {
	// ValidateChallenge is called when a device responds to a challenge.
	// The backend decides whether the challenge response is valid.
	//
	// For relay: Fleet validates device attestation locally, then completes
	//   whatever challenge the upstream CA requires (http-01, dns-01, etc.)
	// For local CA: Fleet validates device attestation directly
	//   (e.g., checks serial number against hosts table).
	ValidateChallenge(ctx context.Context, challenge *Challenge, order *Order) error

	// IssueCertificate is called when an order is ready to be finalized.
	// The backend signs the CSR and returns the certificate chain.
	//
	// For relay: Fleet creates an order on the upstream CA, completes upstream
	//   challenges, forwards the CSR, and returns the upstream-issued certificate.
	// For local CA: Fleet signs the CSR with its embedded CA key.
	IssueCertificate(ctx context.Context, csr *x509.CertificateRequest, order *Order) (*IssuedCertificate, error)

	// RevokeCertificate revokes a previously issued certificate.
	//
	// For relay: Fleet revokes on the upstream CA.
	// For local CA: Fleet marks the certificate as revoked locally.
	RevokeCertificate(ctx context.Context, cert *x509.Certificate, reason int) error
}
