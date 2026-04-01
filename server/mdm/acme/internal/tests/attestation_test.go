package tests

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/mdm/acme/internal/service"
	"github.com/fleetdm/fleet/v4/server/mdm/acme/internal/types"
	"github.com/fxamacker/cbor/v2"
)

// Attestation test helpers

func generateTestAttestationCA(t *testing.T) (*x509.Certificate, *ecdsa.PrivateKey) {
	key := generateTestKey(t)
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
		t.Fatalf("Failed to create test attestation CA certificate: %v", err)
	}
	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		t.Fatalf("Failed to parse test attestation CA certificate: %v", err)
	}
	return cert, key
}

func buildAttestationLeafCert(t *testing.T, ca *x509.Certificate, caKey *ecdsa.PrivateKey, serial, token string) *x509.Certificate {
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

	certDER, err := x509.CreateCertificate(rand.Reader, template, ca, generateTestKey(t).Public(), caKey)
	if err != nil {
		t.Fatalf("Failed to create attestation leaf certificate: %v", err)
	}
	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		t.Fatalf("Failed to parse attestation leaf certificate: %v", err)
	}
	return cert
}

// The leaf cert should always be the first
func buildAppleDeviceAttestationPayload(t *testing.T, certs ...*x509.Certificate) any {
	x5c := make([][]byte, len(certs))
	for i, cert := range certs {
		x5c[i] = cert.Raw
	}

	appleAttest := types.AppleDeviceAttestationStatement{
		X5C: x5c,
	}

	appleAttestCbor, err := cbor.Marshal(appleAttest)
	if err != nil {
		t.Fatalf("Failed to marshal Apple device attestation statement to CBOR: %v", err)
	}

	attObj := types.AttestationObject{
		Format:               "apple",
		AttestationStatement: appleAttestCbor,
	}
	attObjCbor, err := cbor.Marshal(attObj)
	if err != nil {
		t.Fatalf("Failed to marshal attestation object to CBOR: %v", err)
	}

	base64Encoded := base64.RawURLEncoding.EncodeToString(attObjCbor)

	// finally embed in the top-level json payload
	return struct {
		AttObj string `json:"attObj"`
	}{
		AttObj: base64Encoded,
	}
}
