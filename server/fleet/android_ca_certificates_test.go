package fleet

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func generateTestCert(t *testing.T, isCA bool, commonName string, issuerKey *rsa.PrivateKey) ([]byte, []byte) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	signingKey := key
	if issuerKey != nil {
		signingKey = issuerKey
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject: pkix.Name{
			CommonName:   commonName,
			Organization: []string{"Fleet Test Org"},
		},
		Issuer: pkix.Name{
			CommonName:   commonName,
			Organization: []string{"Fleet Test Org"},
		},
		NotBefore:             time.Now().Add(-1 * time.Hour).Truncate(time.Second),
		NotAfter:              time.Now().Add(24 * time.Hour).Truncate(time.Second),
		IsCA:                  isCA,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
	}

	certDer, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, signingKey)
	require.NoError(t, err)

	certPem := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDer,
	})

	privPem := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})

	return certPem, privPem
}

func TestValidateAndParseAndroidCACertificate(t *testing.T) {
	t.Run("valid single cert", func(t *testing.T) {
		certPem, _ := generateTestCert(t, true, "Root CA", nil)
		parsed, err := ValidateAndParseAndroidCACertificate(certPem)
		require.NoError(t, err)
		assert.Equal(t, "CN=Root CA,O=Fleet Test Org", parsed.Subject)
		assert.Equal(t, "Root CA", parsed.CommonName)
		assert.Len(t, parsed.SHA256Fingerprint, 64)
		assert.Len(t, parsed.Certificates, 1)
	})

	t.Run("rejects file with private key", func(t *testing.T) {
		certPem, privPem := generateTestCert(t, true, "Root CA with Key", nil)
		combined := append([]byte(nil), certPem...)
		combined = append(combined, privPem...)
		_, err := ValidateAndParseAndroidCACertificate(combined)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must not contain private keys")
	})

	t.Run("rejects empty cert", func(t *testing.T) {
		_, err := ValidateAndParseAndroidCACertificate([]byte{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be empty")
	})

	t.Run("rejects cert exceeding max size", func(t *testing.T) {
		huge := make([]byte, MaxAndroidCACertificateSize+1)
		_, err := ValidateAndParseAndroidCACertificate(huge)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "exceeds maximum allowed size")
	})

	t.Run("rejects invalid certificate format", func(t *testing.T) {
		_, err := ValidateAndParseAndroidCACertificate([]byte("not a cert"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Invalid certificate format")
	})

	t.Run("valid cert bundle", func(t *testing.T) {
		cert1, _ := generateTestCert(t, true, "Root CA 1", nil)
		cert2, _ := generateTestCert(t, true, "Intermediate CA 2", nil)
		bundle := append([]byte(nil), cert1...)
		bundle = append(bundle, cert2...)

		parsed, err := ValidateAndParseAndroidCACertificate(bundle)
		require.NoError(t, err)
		assert.Equal(t, "CN=Root CA 1,O=Fleet Test Org", parsed.Subject)
		assert.Len(t, parsed.Certificates, 2)
	})
}

func TestHostAndroidCACertificateToHostMDMProfile(t *testing.T) {
	hac := &HostAndroidCACertificate{
		HostUUID:          "test-host-uuid",
		CertificateName:   "Corp Root CA",
		CACertificateID:   42,
		Status:            MDMDeliveryVerified,
		Detail:            "installed ok",
		SHA256Fingerprint: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
	}

	prof := hac.ToHostMDMProfile()
	assert.Equal(t, "test-host-uuid", prof.HostUUID)
	assert.Equal(t, "Corp Root CA", prof.Name)
	assert.Equal(t, "android", prof.Platform)
	assert.Equal(t, AndroidCACertificateProfileID, prof.ProfileUUID)
	assert.Equal(t, uint(42), *prof.CertificateTemplateID)
	assert.Equal(t, string(MDMDeliveryVerified), *prof.Status)
	assert.Equal(t, "installed ok", prof.Detail)
}
