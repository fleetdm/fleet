package cryptoutil

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"math/big"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateSubjectKeyID(t *testing.T) {
	ecKey, err := ecdsa.GenerateKey(elliptic.P224(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		testName string
		pub      crypto.PublicKey
	}{
		{"RSA", &rsa.PublicKey{N: big.NewInt(123), E: 65537}},
		{"ECDSA", ecKey.Public()},
	} {
		test := test
		t.Run(test.testName, func(t *testing.T) {
			t.Parallel()
			ski, err := GenerateSubjectKeyID(test.pub)
			if err != nil {
				t.Fatal(err)
			}
			if len(ski) != 20 {
				t.Fatalf("unexpected subject public key identifier length: %d", len(ski))
			}
			ski2, err := GenerateSubjectKeyID(test.pub)
			if err != nil {
				t.Fatal(err)
			}
			if !testSKIEq(ski, ski2) {
				t.Fatal("subject key identifier generation is not deterministic")
			}
		})
	}
}

func testSKIEq(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

func TestParsePrivateKey(t *testing.T) {
	t.Parallel()
	// nil block not allowed
	_, err := ParsePrivateKey(nil, "APNS private key")
	assert.ErrorContains(t, err, "failed to decode")

	// encrypted pkcs8 not supported
	pkcs8Encrypted, err := os.ReadFile("testdata/pkcs8-encrypted.key")
	require.NoError(t, err)
	_, err = ParsePrivateKey(pkcs8Encrypted, "APNS private key")
	assert.ErrorContains(t, err, "failed to parse APNS private key of type ENCRYPTED PRIVATE KEY")

	// X25519 pkcs8 not supported
	pkcs8Encrypted, err = os.ReadFile("testdata/pkcs8-x25519.key")
	require.NoError(t, err)
	_, err = ParsePrivateKey(pkcs8Encrypted, "APNS private key")
	assert.ErrorContains(t, err, "unmarshaled PKCS8 APNS private key is not")

	// In this test, the pkcs1 key and pkcs8 keys are the same key, just different formats
	pkcs1, err := os.ReadFile("testdata/pkcs1.key")
	require.NoError(t, err)
	pkcs1Key, err := ParsePrivateKey(pkcs1, "APNS private key")
	require.NoError(t, err)

	pkcs8, err := os.ReadFile("testdata/pkcs8-rsa.key")
	require.NoError(t, err)
	pkcs8Key, err := ParsePrivateKey(pkcs8, "APNS private key")
	require.NoError(t, err)

	assert.Equal(t, pkcs1Key, pkcs8Key)
}
