// Package certificate contains functions for handling TLS certificates.
package certificate

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/url"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
)

// LoadPEM loads certificates from a PEM file and returns a cert pool containing
// the certificates.
func LoadPEM(path string) (*x509.CertPool, error) {
	pool := x509.NewCertPool()

	contents, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read certificate file: %w", err)
	}

	if ok := pool.AppendCertsFromPEM(contents); !ok {
		return nil, fmt.Errorf("no valid certificates found in %s", path)
	}

	return pool, nil
}

// ValidateConnection checks that a connection can be successfully established
// to the server URL using the cert pool provided. The validation performed is
// not sufficient to verify authenticity of the server, but it can help to catch
// certificate errors and provide more detailed messages to users.
func ValidateConnection(pool *x509.CertPool, fleetURL string) error {
	return ValidateConnectionContext(context.Background(), pool, fleetURL)
}

// ValidateConnectionContext is like ValidateConnection, but it accepts a
// context that may specify a timeout or deadline for the TLS connection check.
func ValidateConnectionContext(ctx context.Context, pool *x509.CertPool, fleetURL string) error {
	parsed, err := url.Parse(fleetURL)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "parse url")
	}

	dialer := &tls.Dialer{
		Config: &tls.Config{
			RootCAs:            pool,
			InsecureSkipVerify: true,
			VerifyConnection: func(state tls.ConnectionState) error {
				if len(state.PeerCertificates) == 0 {
					return ctxerr.New(ctx, "no peer certificates")
				}

				cert := state.PeerCertificates[0]
				if _, err := cert.Verify(x509.VerifyOptions{
					DNSName: parsed.Hostname(),
					Roots:   pool,
				}); err != nil {
					return ctxerr.Wrap(ctx, err, "verify certificate")
				}

				return nil
			},
		},
	}
	conn, err := dialer.DialContext(ctx, "tcp", parsed.Host)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "dial for validate")
	}
	defer conn.Close()

	return nil
}
