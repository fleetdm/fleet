// Package api defines the public API types for the ACME bounded context.
package api

import (
	"crypto/x509"
	"math/big"
	"time"
)

// Identifier represents an ACME identifier (RFC 8555 §9.7.7).
type Identifier struct {
	// Type is the identifier type: "dns", "ip", or "permanent-identifier".
	Type string
	// Value is the identifier value (e.g., "example.com", "192.168.1.1", device serial).
	Value string
}

// Order represents an ACME order managed by the server.
type Order struct {
	// ID is the server-assigned order identifier.
	ID string
	// CAName identifies which CA backend handles this order.
	CAName string
	// Status is the order status: pending, ready, processing, valid, invalid.
	Status string
	// Identifiers are the requested certificate identifiers.
	Identifiers []Identifier
	// Authorizations are the authorization IDs for this order.
	Authorizations []string
	// CertID is set when the order is finalized and a certificate is issued.
	CertID string
	// CSR is the DER-encoded certificate signing request (set at finalize).
	CSR []byte
	// ExpiresAt is when the order expires if not finalized.
	ExpiresAt time.Time
	// CreatedAt is when the order was created.
	CreatedAt time.Time
}

// Authorization represents an ACME authorization (RFC 8555 §7.1.4).
type Authorization struct {
	// ID is the server-assigned authorization identifier.
	ID string
	// OrderID links back to the parent order.
	OrderID string
	// Identifier is what this authorization proves control of.
	Identifier Identifier
	// Status is the authorization status: pending, valid, invalid, deactivated, expired, revoked.
	Status string
	// Challenges are the available challenges for this authorization.
	Challenges []Challenge
	// ExpiresAt is when the authorization expires.
	ExpiresAt time.Time
}

// Challenge represents an ACME challenge (RFC 8555 §7.1.5).
type Challenge struct {
	// ID is the server-assigned challenge identifier.
	ID string
	// AuthzID links back to the parent authorization.
	AuthzID string
	// Type is the challenge type: "http-01", "dns-01", "device-attest-01", etc.
	Type string
	// Token is the challenge token.
	Token string
	// Status is the challenge status: pending, processing, valid, invalid.
	Status string
	// Payload is the device's challenge response (e.g., attestation statement).
	// Set when the device responds to the challenge.
	Payload []byte
}

// IssuedCertificate contains the result of certificate issuance.
type IssuedCertificate struct {
	// DERChain is the certificate chain in DER format (leaf first).
	DERChain [][]byte
	// Leaf is the parsed leaf certificate (for metadata extraction).
	Leaf *x509.Certificate
	// SerialNumber is the certificate serial number.
	SerialNumber *big.Int
	// NotBefore is the certificate validity start.
	NotBefore time.Time
	// NotAfter is the certificate validity end.
	NotAfter time.Time
}

// CAConfig holds the configuration for a CA backend.
type CAConfig struct {
	// Name is the identifier used in the URL path segment (e.g., "smallstep").
	Name string
	// Type is the backend type: "relay" or "local".
	Type string
	// DirectoryURL is the upstream CA's ACME directory URL (relay only).
	DirectoryURL string
	// EABKeyID is the External Account Binding key ID (relay only).
	// Required when the upstream CA has externalAccountRequired=true.
	EABKeyID string
	// EABHMACKey is the EAB HMAC key, base64url-encoded (relay only).
	// Used to sign the account binding JWS during registration (RFC 8555 §7.3.4).
	EABHMACKey string
	// CACert is the PEM-encoded CA certificate for TLS verification (optional, relay only).
	CACert []byte
	// ClientCert is the PEM-encoded mTLS client certificate (optional, relay only).
	ClientCert []byte
	// ClientKey is the PEM-encoded mTLS client private key (optional, relay only).
	ClientKey []byte
}
