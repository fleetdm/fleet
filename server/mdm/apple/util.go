package apple_mdm

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
)

// Note Apple rejects CSRs if the key size is not 2048.
const rsaKeySize = 2048

// newPrivateKey creates an RSA private key
func newPrivateKey() (*rsa.PrivateKey, error) {
	return rsa.GenerateKey(rand.Reader, rsaKeySize)
}

// EncodeCertPEM returns PEM-endcoded certificate data.
func EncodeCertPEM(cert *x509.Certificate) []byte {
	block := pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert.Raw,
	}
	return pem.EncodeToMemory(&block)
}

func DecodeCertPEM(encoded []byte) (*x509.Certificate, error) {
	block, _ := pem.Decode(encoded)
	if block == nil {
		return nil, errors.New("no PEM-encoded data found")
	}
	if block.Type != "CERTIFICATE" {
		return nil, fmt.Errorf("unexpected block type %s", block.Type)
	}

	return x509.ParseCertificate(block.Bytes)
}

func EncodeCertRequestPEM(cert *x509.CertificateRequest) []byte {
	pemBlock := &pem.Block{
		Type:    "CERTIFICATE REQUEST",
		Headers: nil,
		Bytes:   cert.Raw,
	}

	return pem.EncodeToMemory(pemBlock)
}

// EncodePrivateKeyPEM returns PEM-encoded private key data
func EncodePrivateKeyPEM(key *rsa.PrivateKey) []byte {
	block := pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}
	return pem.EncodeToMemory(&block)
}

// DecodePrivateKeyPEM decodes PEM-encoded private key data.
func DecodePrivateKeyPEM(encoded []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(encoded)
	if block == nil {
		return nil, errors.New("no PEM-encoded data found")
	}
	if block.Type != "RSA PRIVATE KEY" {
		return nil, fmt.Errorf("unexpected block type %s", block.Type)
	}

	return x509.ParsePKCS1PrivateKey(block.Bytes)
}
