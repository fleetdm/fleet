// Package certificate contains functions for handling TLS certificates.
package certificate

import (
	"bytes"
	"crypto/tls"
	"encoding/pem"

	"github.com/pkg/errors"
)

// FetchPEM retrieves the certificate chain presented by the server listening at
// hostname in PEM format.
//
// Adapted from https://stackoverflow.com/a/46735876/491710
func FetchPEM(hostname string) ([]byte, error) {
	conn, err := tls.Dial("tcp", hostname, &tls.Config{
		InsecureSkipVerify: true,
	})
	if err != nil {
		return nil, errors.Wrap(err, "dial server to fetch PEM")
	}
	defer conn.Close()

	var b bytes.Buffer
	for _, cert := range conn.ConnectionState().PeerCertificates {
		err := pem.Encode(&b, &pem.Block{
			Type:  "CERTIFICATE",
			Bytes: cert.Raw,
		})
		if err != nil {
			return nil, errors.Wrap(err, "encode PEM")
		}
	}
	return b.Bytes(), nil
}
