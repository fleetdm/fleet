// Package scep_ca implements functionality to create and handle a SCEP CA certificate.
package scep_ca

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"

	"github.com/micromdm/scep/v2/depot"
)

// Create creates a self-signed CA certificate and returns the certificate and its private key.
func Create(years int, cn, org, orgUnit, country string) (certPEM []byte, keyPEM []byte, err error) {
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}
	caCert := depot.NewCACert(
		depot.WithYears(years),
		depot.WithCommonName(cn),
		depot.WithOrganization(org),
		depot.WithOrganizationalUnit(orgUnit),
		depot.WithCountry(country),
	)
	crtBytes, err := caCert.SelfSign(rand.Reader, &privKey.PublicKey, privKey)
	if err != nil {
		return nil, nil, err
	}
	crt, err := x509.ParseCertificate(crtBytes)
	if err != nil {
		return nil, nil, err
	}
	pemCertBlock := &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: crt.Raw,
	}
	certPEM = pem.EncodeToMemory(pemCertBlock)
	pemKeyBlock := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privKey),
	}
	keyPEM = pem.EncodeToMemory(pemKeyBlock)
	return certPEM, keyPEM, nil
}

// Load loads the SCEP CA certificate and key from the raw PEM bytes.
func Load(pemCert, pemKey []byte) (*x509.Certificate, *rsa.PrivateKey, error) {
	certBlock, _ := pem.Decode(pemCert)
	if certBlock.Type != "CERTIFICATE" {
		return nil, nil, fmt.Errorf("PEM block not a certificate: %s", certBlock.Type)
	}
	crt, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return nil, nil, err
	}
	keyBlock, _ := pem.Decode(pemKey)
	if keyBlock.Type != "RSA PRIVATE KEY" {
		return nil, nil, fmt.Errorf("PEM block not a rsa private key: %s", keyBlock.Type)
	}
	key, err := x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
	if err != nil {
		return nil, nil, err
	}
	return crt, key, nil
}
