package service

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"io"
	"net"
	"net/url"

	"github.com/pkg/errors"
)

// Certificate returns the PEM encoded certificate chain for osqueryd TLS termination.
func (svc *Service) CertificateChain(ctx context.Context) ([]byte, error) {
	config, err := svc.AppConfig(ctx)
	if err != nil {
		return nil, err
	}

	u, err := url.Parse(config.ServerURL)
	if err != nil {
		return nil, errors.Wrap(err, "parsing serverURL")
	}

	conn, err := connectTLS(u)
	if err != nil {
		return nil, err
	}

	return chain(conn.ConnectionState(), u.Hostname())
}

func connectTLS(serverURL *url.URL) (*tls.Conn, error) {
	var hostport string
	if serverURL.Port() == "" {
		hostport = net.JoinHostPort(serverURL.Host, "443")
	} else {
		hostport = serverURL.Host
	}

	// attempt dialing twice, first with a secure conn, and then
	// if that fails, use insecure
	dial := func(insecure bool) (*tls.Conn, error) {
		conn, err := tls.Dial("tcp", hostport, &tls.Config{
			InsecureSkipVerify: insecure})
		if err != nil {
			return nil, errors.Wrap(err, "dial tls")
		}
		defer conn.Close()
		return conn, nil
	}

	var (
		conn *tls.Conn
		err  error
	)

	conn, err = dial(false)
	if err == nil {
		return conn, nil
	}
	conn, err = dial(true)
	return conn, err
}

// chain builds a PEM encoded certificate chain using the PeerCertificates
// in tls.ConnectionState. chain uses the hostname to omit the Leaf certificate
// from the chain.
func chain(cs tls.ConnectionState, hostname string) ([]byte, error) {
	buf := bytes.NewBuffer([]byte(""))

	verifyEncode := func(chain []*x509.Certificate) error {
		for _, cert := range chain {
			if len(chain) > 1 {
				// drop the leaf certificate from the chain. osqueryd does not
				// need it to establish a secure connection
				if err := cert.VerifyHostname(hostname); err == nil {
					continue
				}
			}
			if err := encodePEMCertificate(buf, cert); err != nil {
				return err
			}
		}
		return nil
	}

	// use verified chains if available(which adds the root CA), otherwise
	// use the certificate chain offered by the server (if terminated with
	// self-signed certs)
	if len(cs.VerifiedChains) != 0 {
		for _, chain := range cs.VerifiedChains {
			if err := verifyEncode(chain); err != nil {
				return nil, errors.Wrap(err, "encode verified chains pem")
			}
		}
	} else {
		if err := verifyEncode(cs.PeerCertificates); err != nil {
			return nil, errors.Wrap(err, "encode peer certificates pem")
		}
	}
	return buf.Bytes(), nil
}

func encodePEMCertificate(buf io.Writer, cert *x509.Certificate) error {
	block := &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert.Raw,
	}
	return pem.Encode(buf, block)
}
