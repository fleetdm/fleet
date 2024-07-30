// Package certificate contains functions for handling TLS certificates.
package certificate

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
)

// LoadPEM loads certificates from a PEM file and returns a cert pool containing
// the certificates.
func LoadPEM(path string) (*x509.CertPool, error) {
	pool := x509.NewCertPool()

	contents, err := os.ReadFile(path)
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
func ValidateConnectionContext(ctx context.Context, pool *x509.CertPool, targetURL string) error {
	parsed, err := url.Parse(targetURL)
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
				intermediates := x509.NewCertPool()
				for _, intermediate := range state.PeerCertificates[1:] {
					intermediates.AddCert(intermediate)
				}

				if _, err := cert.Verify(x509.VerifyOptions{
					DNSName:       parsed.Hostname(),
					Roots:         pool,
					Intermediates: intermediates,
				}); err != nil {
					return ctxerr.Wrap(ctx, err, "verify certificate")
				}

				return nil
			},
		},
	}
	conn, err := dialer.DialContext(ctx, "tcp", getHostPort(parsed))
	if err != nil {
		return ctxerr.Wrap(ctx, err, "dial for validate")
	}
	conn.Close()

	return nil
}

// ValidateClientAuthTLSConnection validates that a TLS connection can be made
// to the server identified by the target URL (only the host portion is used)
// by authenticating the client using the provided certificate. The ctx may
// specify a timeout or deadline for the TLS connection check.
func ValidateClientAuthTLSConnection(ctx context.Context, cert *tls.Certificate, targetURL string) error {
	parsed, err := url.Parse(targetURL)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "parse url")
	}

	dialer := &tls.Dialer{
		Config: &tls.Config{
			GetClientCertificate: func(reqInfo *tls.CertificateRequestInfo) (*tls.Certificate, error) {
				return cert, nil
			},
			ServerName:         parsed.Hostname(),
			ClientSessionCache: tls.NewLRUClientSessionCache(-1),
		},
	}

	conn, err := dialer.DialContext(ctx, "tcp", getHostPort(parsed))
	if err != nil {
		return ctxerr.Wrap(ctx, err, "TLS dial")
	}
	defer conn.Close()

	if _, err = conn.Read(make([]byte, 1024)); err != nil {
		return ctxerr.Wrap(ctx, err, "read from TLS connection")
	}
	return nil
}

func getHostPort(u *url.URL) string {
	host, port := u.Hostname(), u.Port()
	if port == "" {
		// the dialer accepts a port number or a service name, so using the scheme
		// as port results in the default port for that service (e.g. 443 for
		// https).
		return net.JoinHostPort(host, u.Scheme)
	}
	return net.JoinHostPort(host, port)
}

// Certificate holds a loaded TLS certificate and its raw parts.
type Certificate struct {
	Crt    tls.Certificate
	RawCrt []byte
	RawKey []byte
}

// LoadClientCertificateFromFiles loads a TLS client certificate from PEM cert and key file paths.
//
// Returns (nil, nil) if both files do not exist.
func LoadClientCertificateFromFiles(crtPath, keyPath string) (*Certificate, error) {
	checkFileExists := func(filePath string) (bool, error) {
		switch s, err := os.Stat(filePath); {
		case err == nil:
			return !s.IsDir(), nil
		case errors.Is(err, os.ErrNotExist):
			return false, nil
		default:
			return false, err
		}
	}

	if (crtPath != "") != (keyPath != "") {
		return nil, fmt.Errorf(
			"both crt path and key path must be set: crt=%t, key=%t", crtPath != "", keyPath != "",
		)
	}
	if crtPath == "" {
		return nil, nil
	}

	crtExists, err := checkFileExists(crtPath)
	if err != nil {
		return nil, err
	}
	keyExists, err := checkFileExists(keyPath)
	if err != nil {
		return nil, err
	}

	if crtExists != keyExists {
		return nil, fmt.Errorf(
			"both crt and key files must exist: %s: %t, %s: %t",
			crtPath, crtExists, keyPath, keyExists,
		)
	}
	if !crtExists {
		return nil, nil
	}

	crtBytes, err := os.ReadFile(crtPath)
	if err != nil {
		return nil, err
	}
	keyBytes, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, err
	}
	crt, err := parseFullClientCertificate(crtBytes, keyBytes)
	if err != nil {
		return nil, err
	}

	return &Certificate{
		Crt:    crt,
		RawCrt: crtBytes,
		RawKey: keyBytes,
	}, nil
}

// LoadClientCertificate loads a client certificate from the given PEM cert and key strings.
//
// Returns (nil, nil) if both values are empty.
func LoadClientCertificate(crt, key string) (*tls.Certificate, error) {
	if (crt != "") != (key != "") {
		return nil, fmt.Errorf(
			"both crt and key must be set: crt=%t, key=%t", crt != "", key != "",
		)
	}
	if crt == "" {
		return nil, nil
	}

	cert, err := parseFullClientCertificate([]byte(crt), []byte(key))
	if err != nil {
		return nil, err
	}

	return &cert, nil
}

func parseFullClientCertificate(crt, key []byte) (tls.Certificate, error) {
	cert, err := tls.X509KeyPair(crt, key)
	if err != nil {
		return tls.Certificate{}, err
	}
	// tls.X509KeyPair does not store the parsed certificate leaf.
	// To reduce per-handshake processing, we parse it here.
	//
	// From Adam Langley:
	//	The Leaf member is only needed for clients doing client-authentication.
	//	This is rare compared to the common case of loading certificates for serving.
	// 	In the latter case, the parsed form isn't needed because the server just sends
	// 	the blob to the client and doesn't generally care what's in it.
	parsedLeaf, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("parse leaf certificate: %w", err)
	}
	cert.Leaf = parsedLeaf
	return cert, nil
}
