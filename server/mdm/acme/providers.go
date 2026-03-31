package acme

import (
	"context"
	"crypto/x509"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

// CSRSigner signs x509 certificate requests. This is the ACME-specific
// interface for certificate signing, decoupled from the SCEP protocol types.
type CSRSigner interface {
	SignCSR(ctx context.Context, csr *x509.CertificateRequest) (*x509.Certificate, error)
}

// CSRSignerFunc is an adapter to allow use of ordinary functions as CSRSigner.
type CSRSignerFunc func(ctx context.Context, csr *x509.CertificateRequest) (*x509.Certificate, error)

func (f CSRSignerFunc) SignCSR(ctx context.Context, csr *x509.CertificateRequest) (*x509.Certificate, error) {
	return f(ctx, csr)
}

// DataProviders combines all external dependency interfaces for the ACME
// bounded context.
type DataProviders interface {
	AppConfig(ctx context.Context) (*fleet.AppConfig, error)
	CSRSigner(ctx context.Context) (CSRSigner, error)
}
