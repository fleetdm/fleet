// Package certificate contains functions for handling TLS certificates.
package certificate

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"net/url"

	"github.com/pkg/errors"
)

// LoadPEM loads certificates from a PEM file and returns a cert pool containing
// the certificates.
func LoadPEM(path string) (*x509.CertPool, error) {
	pool := x509.NewCertPool()

	contents, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, errors.Wrap(err, "read certificate file")
	}

	if ok := pool.AppendCertsFromPEM(contents); !ok {
		return nil, errors.Errorf("no valid ceritificates found in %s", path)
	}

	return pool, nil
}

// ValidateConnection checks that a connection can be successfully established
// to the server URL using the cert pool provided. The validation performed is
// not sufficient to verify authenticity of the server, but it can help to catch
// certificate errors and provide more detailed messages to users.
func ValidateConnection(pool *x509.CertPool, fleetURL string) error {
	parsed, err := url.Parse(fleetURL)
	if err != nil {
		return errors.Wrap(err, "parse url")
	}
	conn, err := tls.Dial("tcp", parsed.Host, &tls.Config{
		ClientCAs:          pool,
		InsecureSkipVerify: true,
		VerifyConnection: func(state tls.ConnectionState) error {
			if len(state.PeerCertificates) == 0 {
				return errors.New("no peer certificates")
			}

			cert := state.PeerCertificates[0]
			if _, err := cert.Verify(x509.VerifyOptions{
				DNSName: parsed.Hostname(),
				Roots:   pool,
			}); err != nil {
				return errors.Wrap(err, "verify certificate")
			}

			return nil
		},
	})
	if err != nil {
		return errors.Wrap(err, "dial for validate")
	}
	defer conn.Close()

	return nil
}
