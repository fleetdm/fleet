package tokenpki

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"time"
)

// SelfSignedRSAKeypair generates a 2048-bit RSA private key and self-signs an
// X.509 certificate using it. You can set the Common Name in cn and the
// validity duration with days.
func SelfSignedRSAKeypair(cn string, days int) (*rsa.PrivateKey, *x509.Certificate, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}
	timeNow := time.Now()
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: cn,
		},
		NotBefore: timeNow.Add(time.Minute * -10),
		NotAfter:  timeNow.Add(time.Duration(days) * 24 * time.Hour),
		KeyUsage:  x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
	}
	certBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		return nil, nil, err
	}
	cert, err := x509.ParseCertificate(certBytes)
	if err != nil {
		return nil, nil, err
	}
	return key, cert, err
}

// PEMRSAPrivateKey returns key as a PEM block.
func PEMRSAPrivateKey(key *rsa.PrivateKey) []byte {
	block := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}
	return pem.EncodeToMemory(block)
}

// RSAKeyFromPEM decodes a PEM RSA private key.
func RSAKeyFromPEM(key []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(key)
	if block.Type != "RSA PRIVATE KEY" {
		return nil, errors.New("PEM type is not RSA PRIVATE KEY")
	}
	return x509.ParsePKCS1PrivateKey(block.Bytes)
}

// PEMCertificate returns derBytes encoded as a PEM block.
func PEMCertificate(derBytes []byte) []byte {
	block := &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: derBytes,
	}
	return pem.EncodeToMemory(block)
}

// CertificateFromPEM decodes a PEM certificate.
func CertificateFromPEM(cert []byte) (*x509.Certificate, error) {
	block, _ := pem.Decode(cert)
	if block.Type != "CERTIFICATE" {
		return nil, errors.New("PEM type is not CERTIFICATE")
	}
	return x509.ParseCertificate(block.Bytes)
}
