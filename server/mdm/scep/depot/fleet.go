package depot

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
)

// NewSCEPCACertKey creates a self-signed CA certificate for use with SCEP and
// returns the certificate and its private key.
func NewSCEPCACertKey() (*x509.Certificate, *rsa.PrivateKey, error) {
	key, err := newPrivateKey()
	if err != nil {
		return nil, nil, err
	}

	caCert := NewCACert(
		WithYears(10),
		WithCommonName("Fleet"),
	)

	crtBytes, err := caCert.SelfSign(rand.Reader, key.Public(), key)
	if err != nil {
		return nil, nil, err
	}

	cert, err := x509.ParseCertificate(crtBytes)
	if err != nil {
		return nil, nil, err
	}

	return cert, key, nil
}

// Note Apple rejects CSRs if the key size is not 2048.
const rsaKeySize = 2048

// newPrivateKey creates an RSA private key
func newPrivateKey() (*rsa.PrivateKey, error) {
	return rsa.GenerateKey(rand.Reader, rsaKeySize)
}
