package testhelpers

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/fleetdm/fleet/v4/server/mdm/acme/internal/service"
	"github.com/fleetdm/fleet/v4/server/mdm/acme/internal/types"
	"github.com/fxamacker/cbor/v2"
)

// GenerateTestKey generates an ECDSA P-256 key pair and returns the private key and public JWK.
func GenerateTestKey() (*ecdsa.PrivateKey, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	return key, err
}

func GenerateTestAttestationCA() (*x509.Certificate, *ecdsa.PrivateKey, error) {
	key, err := GenerateTestKey()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate test key: %w", err)
	}
	template := &x509.Certificate{
		Subject:               pkix.Name{CommonName: "Test Attestation CA"},
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
		IsCA:                  true,
		BasicConstraintsValid: true,
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
	}
	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create test attestation CA certificate: %w", err)
	}
	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse test attestation CA certificate: %w", err)
	}
	return cert, key, nil
}

func BuildAttestationLeafCert(ca *x509.Certificate, caKey *ecdsa.PrivateKey, serial, token string) (*x509.Certificate, error) {
	template := &x509.Certificate{
		Subject:               pkix.Name{CommonName: "Test Attestation Leaf"},
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
		BasicConstraintsValid: true,
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
	}

	hashedNonce := sha256.Sum256([]byte(token))

	template.ExtraExtensions = []pkix.Extension{
		{
			Id:    service.OIDAppleSerialNumber,
			Value: []byte(serial),
		},
		{
			Id:    service.OIDAppleNonce,
			Value: hashedNonce[:],
		},
	}

	key, err := GenerateTestKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate key for attestation leaf cert: %w", err)
	}
	certDER, err := x509.CreateCertificate(rand.Reader, template, ca, key.Public(), caKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create attestation leaf certificate: %w", err)
	}
	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return nil, fmt.Errorf("failed to parse attestation leaf certificate: %w", err)
	}
	return cert, nil
}

// The leaf cert should always be the first
func BuildAppleDeviceAttestationPayload(certs ...*x509.Certificate) (any, error) {
	x5c := make([][]byte, len(certs))
	for i, cert := range certs {
		x5c[i] = cert.Raw
	}

	appleAttest := types.AppleDeviceAttestationStatement{
		X5C: x5c,
	}

	appleAttestCbor, err := cbor.Marshal(appleAttest)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Apple device attestation statement to CBOR: %w", err)
	}

	attObj := types.AttestationObject{
		Format:               "apple",
		AttestationStatement: appleAttestCbor,
	}
	attObjCbor, err := cbor.Marshal(attObj)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal attestation object to CBOR: %w", err)
	}

	base64Encoded := base64.RawURLEncoding.EncodeToString(attObjCbor)

	// finally embed in the top-level json payload
	return struct {
		AttObj string `json:"attObj"`
	}{
		AttObj: base64Encoded,
	}, nil
}

// generateCSRDER creates a base64 URL encoded DER-encoded ECDSA CSR with the given common name.
func GenerateCSRDER(commonName string) (string, *ecdsa.PrivateKey, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return "", nil, fmt.Errorf("failed to generate key for CSR: %w", err)
	}
	template := &x509.CertificateRequest{
		Subject: pkix.Name{
			CommonName: commonName,
		},
	}
	csrDER, err := x509.CreateCertificateRequest(rand.Reader, template, key)
	if err != nil {
		return "", nil, fmt.Errorf("failed to create CSR: %w", err)
	}

	// base64 URL encode the DER csr as per the RFC 7.4 spec
	encoded := base64.RawURLEncoding.EncodeToString(csrDER)
	return encoded, key, nil
}
