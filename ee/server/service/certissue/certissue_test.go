package certissue

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"software.sslmate.com/src/go-pkcs12"
)

func TestUPNExtension(t *testing.T) {
	// The encoding OpenSSL produces for these two UPNs, which EJBCA accepts:
	// openssl req -addext "subjectAltName=otherName:1.3.6.1.4.1.311.20.2.3;UTF8:user@example.com,otherName:1.3.6.1.4.1.311.20.2.3;UTF8:b@c.d"
	const openSSLEncoding = "3039a020060a2b060104018237140203a0120c1075736572406578616d706c652e636f6da015060a2b060104018237140203a0070c" +
		"056240632e64"

	ext, ok := UPNExtension([]string{"user@example.com", "", "b@c.d"})
	require.True(t, ok)
	assert.True(t, ext.Id.Equal(OIDSubjectAltName))
	assert.Equal(t, openSSLEncoding, hex.EncodeToString(ext.Value))

	_, ok = UPNExtension([]string{"", " "})
	assert.False(t, ok)

	// And the UPNs are read back from the encoding OpenSSL produces.
	der, err := hex.DecodeString(openSSLEncoding)
	require.NoError(t, err)
	cert := &x509.Certificate{Extensions: []pkix.Extension{{Id: OIDSubjectAltName, Value: der}}}
	assert.Equal(t, []string{"user@example.com", "b@c.d"}, UPNs(cert))
}

func TestNewCSR(t *testing.T) {
	for _, tc := range []struct {
		name string
		upns []string
	}{
		{"common name only", nil},
		{"with UPN", []string{"user@example.com"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			key, csrPEM, err := NewCSR(t.Context(), "host-1", tc.upns)
			require.NoError(t, err)
			block, _ := pem.Decode([]byte(csrPEM))
			require.NotNil(t, block)
			require.Equal(t, "CERTIFICATE REQUEST", block.Type)
			csr, err := x509.ParseCertificateRequest(block.Bytes)
			require.NoError(t, err)
			require.NoError(t, csr.CheckSignature())
			assert.Equal(t, "host-1", csr.Subject.CommonName)
			assert.True(t, key.PublicKey.Equal(csr.PublicKey))
			// Reuse the certificate parser to read the UPNs back out of the CSR extensions.
			assert.Equal(t, tc.upns, UPNs(&x509.Certificate{Extensions: csr.Extensions}))
		})
	}
}

func TestEncodePKCS12(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	cert := selfSigned(t, key)

	data, password, err := EncodePKCS12(t.Context(), key, cert)
	require.NoError(t, err)
	require.NotEmpty(t, password)
	gotKey, gotCert, err := pkcs12.Decode(data, password)
	require.NoError(t, err)
	assert.True(t, key.Equal(gotKey))
	assert.True(t, cert.Equal(gotCert))
}

func selfSigned(t *testing.T, key *rsa.PrivateKey) *x509.Certificate {
	t.Helper()
	template := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "host-1"}}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)
	cert, err := x509.ParseCertificate(der)
	require.NoError(t, err)
	return cert
}
