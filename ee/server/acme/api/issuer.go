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
//  1. ValidateChallenge — when a device responds to a device-attest-01
//     challenge. Fleet validates the attestation (genuine Apple hardware,
//     correct serial number, enrolled in Fleet). This validation is the
//     same regardless of backend — Fleet is the trust boundary for device
//     identity.
//
//  2. IssueCertificate — when an order is ready to be finalized. The
//     backend obtains a signed certificate for the CSR.
//
// # Trust Model
//
// The device attestation (device-attest-01) is cryptographically bound to
// Fleet's ACME session — it includes Fleet's challenge token and the
// device's account key thumbprint. This attestation cannot be forwarded to
// an upstream CA because the upstream has a different session (different
// tokens, different account keys). Only the device can create attestation
// statements (Secure Enclave), and it only participates in one session.
//
// Therefore:
//   - Fleet verifies device identity (attestation)
//   - The upstream CA trusts Fleet (via EAB credentials, RA mode, etc.)
//   - The upstream CA never sees the device attestation
//
// This is the same trust model as Jamf's SCEP Proxy and ESnet's acme-proxy.
//
// # Implementations
//
//   - RelayBackend: authenticates to an upstream ACME CA (via EAB or other
//     mechanism), creates an order, and forwards the CSR. The upstream CA
//     trusts Fleet and issues the certificate without its own device
//     validation.
//   - LocalCABackend: signs the CSR directly with an embedded CA key.
type CertificateIssuer interface {
	// ValidateChallenge is called when a device responds to a challenge.
	//
	// The challenge payload contains the device's attestation statement.
	// Fleet validates that the device is genuine hardware with the expected
	// identity (serial number, enrollment status). This validation is
	// performed by Fleet regardless of backend type.
	//
	// For relay: validation is identical to local CA — Fleet checks the
	//   attestation locally. The upstream CA is not involved in device
	//   validation.
	// For local CA: same — Fleet checks the attestation locally.
	ValidateChallenge(ctx context.Context, challenge *Challenge, order *Order) error

	// IssueCertificate is called when an order is ready to be finalized.
	// The backend obtains a signed certificate for the CSR.
	//
	// For relay: Fleet authenticates to the upstream CA (e.g., via EAB),
	//   creates an order, and submits the CSR. The upstream CA trusts
	//   Fleet and issues the certificate — the upstream does not perform
	//   its own device validation. If the upstream CA pre-authorizes
	//   EAB-authenticated clients, no upstream challenges are needed.
	// For local CA: Fleet signs the CSR with its embedded CA key.
	IssueCertificate(ctx context.Context, csr *x509.CertificateRequest, order *Order) (*IssuedCertificate, error)
}
