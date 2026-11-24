package acme

import (
	"context"
	"crypto/x509"
	"crypto/x509/pkix"

	redigo "github.com/gomodule/redigo/redis"
)

// RedisPool is the minimal Redis pool interface needed by the ACME bounded context.
// fleet.RedisPool satisfies this implicitly via Go's structural typing.
type RedisPool interface {
	Get() redigo.Conn
}

// CSRSigner signs x509 certificate requests. This is the ACME-specific
// interface for certificate signing, decoupled from the SCEP protocol types.
type CSRSigner interface {
	SignX509CSRWithCallback(csr *x509.CertificateRequest, subject pkix.Name, callback func(tmpl *x509.Certificate)) (*x509.Certificate, error)
}

// CSRSignerFunc is an adapter to allow use of ordinary functions as CSRSigner.
type CSRSignerFunc func(csr *x509.CertificateRequest, subject pkix.Name, callback func(tmpl *x509.Certificate)) (*x509.Certificate, error)

func (f CSRSignerFunc) SignX509CSRWithCallback(csr *x509.CertificateRequest, subject pkix.Name, callback func(tmpl *x509.Certificate)) (*x509.Certificate, error) {
	return f(csr, subject, callback)
}

// DataProviders combines all external dependency interfaces for the ACME
// bounded context. Methods are narrowed to exactly what ACME needs,
// keeping the bounded context decoupled from Fleet-internal types.
type DataProviders interface {
	// ServerURL returns the base URL used to construct ACME endpoint URLs.
	ServerURL(ctx context.Context) (string, error)

	// GetCACertificatePEM returns the PEM-encoded root CA certificate
	// used to build the certificate chain in download-certificate responses.
	GetCACertificatePEM(ctx context.Context) ([]byte, error)

	// CSRSigner returns the signer used to sign certificate requests
	// during order finalization.
	CSRSigner(ctx context.Context) (CSRSigner, error)

	// IsDEPEnrolled reports whether the given serial number has an active
	// DEP assignment, used during device attestation challenge validation.
	IsDEPEnrolled(ctx context.Context, serial string) (bool, error)
}
