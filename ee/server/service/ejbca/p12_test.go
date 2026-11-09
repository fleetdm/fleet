package ejbca

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"os"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/ee/server/service/ejbca/ejbcatest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"software.sslmate.com/src/go-pkcs12"
)

// pemBlocks returns the type and DER bytes of each PEM block.
func pemBlocks(t *testing.T, data string) (types []string, ders [][]byte) {
	rest := []byte(data)
	for len(rest) > 0 {
		var block *pem.Block
		block, rest = pem.Decode(rest)
		require.NotNil(t, block)
		types = append(types, block.Type)
		ders = append(ders, block.Bytes)
	}
	return types, ders
}

func TestParseClientP12(t *testing.T) {
	t.Parallel()

	// A client certificate for the end entity "KS User" exported by EJBCA 9.6.3 (Community) through its REST keystore endpoint.
	t.Run("exported by EJBCA", func(t *testing.T) {
		p12, err := os.ReadFile("testdata/ejbca-9.6.3.p12")
		require.NoError(t, err)

		creds, err := ParseClientP12(t.Context(), p12, "KsPw1234")
		require.NoError(t, err)
		assert.Equal(t, "KS User", creds.Leaf.Subject.CommonName)

		_, err = tls.X509KeyPair([]byte(creds.CertPEM), []byte(creds.KeyPEM))
		assert.NoError(t, err)
	})

	ca := ejbcatest.NewCA(t, "CA")
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	ecKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	leafFor := func(pub any) *x509.Certificate {
		return ca.Issue(t, &x509.Certificate{
			Subject:   pkix.Name{CommonName: "Fleet"},
			NotBefore: time.Now().Add(-time.Hour),
			NotAfter:  time.Now().Add(time.Hour),
		}, pub)
	}
	rsaLeaf, ecLeaf := leafFor(&rsaKey.PublicKey), leafFor(&ecKey.PublicKey)

	for _, tc := range []struct {
		name     string
		encoder  *pkcs12.Encoder
		key      any
		leaf     *x509.Certificate
		password string
	}{
		{"RSA key with modern encryption", pkcs12.Modern, rsaKey, rsaLeaf, "secret"},
		{"RSA key with legacy encryption", pkcs12.Legacy, rsaKey, rsaLeaf, "secret"},
		{"ECDSA key with modern encryption", pkcs12.Modern, ecKey, ecLeaf, "secret"},
		{"non-ASCII password", pkcs12.Modern, ecKey, ecLeaf, "pässwörd€密码"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p12, err := tc.encoder.Encode(tc.key, tc.leaf, []*x509.Certificate{ca.Cert}, tc.password)
			require.NoError(t, err)

			creds, err := ParseClientP12(t.Context(), p12, tc.password)
			require.NoError(t, err)
			assert.Equal(t, tc.leaf.Raw, creds.Leaf.Raw)
			_, ders := pemBlocks(t, creds.CertPEM)
			require.Len(t, ders, 2)
			assert.Equal(t, tc.leaf.Raw, ders[0])
			assert.Equal(t, ca.Cert.Raw, ders[1])
			_, keyDERs := pemBlocks(t, creds.KeyPEM)
			parsed, err := x509.ParsePKCS8PrivateKey(keyDERs[0])
			require.NoError(t, err)
			assert.True(t, tc.key.(interface{ Equal(crypto.PrivateKey) bool }).Equal(parsed))
		})
	}

	// The chain doesn't have to put the leaf first, and the leaf is the certificate for the key.
	t.Run("leaf not first", func(t *testing.T) {
		otherCA := ejbcatest.NewCA(t, "Other CA")
		p12, err := pkcs12.Modern.Encode(rsaKey, ca.Cert, []*x509.Certificate{otherCA.Cert, rsaLeaf}, "secret")
		require.NoError(t, err)
		creds, err := ParseClientP12(t.Context(), p12, "secret")
		require.NoError(t, err)
		assert.Equal(t, rsaLeaf.Raw, creds.Leaf.Raw)
		_, ders := pemBlocks(t, creds.CertPEM)
		require.Len(t, ders, 3)
		assert.Equal(t, [][]byte{rsaLeaf.Raw, ca.Cert.Raw, otherCA.Cert.Raw}, ders, "the leaf first, and the others in their order")
	})

	valid, err := pkcs12.Modern.Encode(rsaKey, rsaLeaf, nil, "secret")
	require.NoError(t, err)
	mismatched, err := pkcs12.Modern.Encode(ecKey, rsaLeaf, nil, "secret")
	require.NoError(t, err)
	for _, tc := range []struct {
		name     string
		p12      []byte
		password string
		wantErr  string
	}{
		{"wrong password", valid, "wrong", msgP12BadPassword},
		{"key that doesn't match the certificate", mismatched, "secret", msgP12Mismatch},
		{"BER encoded", append([]byte{0x30, 0x80}, valid[2:]...), "secret", msgP12BER},
		{"not a PKCS#12 file", []byte("not a PKCS#12 file"), "secret", "Couldn't read the PKCS#12 file"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			creds, err := ParseClientP12(t.Context(), tc.p12, tc.password)
			require.ErrorContains(t, err, tc.wantErr)
			require.Nil(t, creds)
		})
	}
}
