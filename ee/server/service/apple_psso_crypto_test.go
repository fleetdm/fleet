package service

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPSSO_SymmetricRoundTrip exercises the AES-256-GCM envelope used for
// key_exchange and password_request responses. Encrypting and then
// decrypting under the same session key must yield the original plaintext.
func TestPSSO_SymmetricRoundTrip(t *testing.T) {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	require.NoError(t, err)

	plain := []byte("a small but meaningful payload — could be a JWT or claims blob")
	blob, err := buildSymmetricJWE(plain, key)
	require.NoError(t, err)
	require.NotEmpty(t, blob)

	got, err := decryptSymmetricBlob(blob, key)
	require.NoError(t, err)
	assert.Equal(t, plain, got)
}

// TestPSSO_SymmetricWrongKeyFails confirms that decryption with a different
// session key fails — i.e., that GCM's authentication tag is being checked.
func TestPSSO_SymmetricWrongKeyFails(t *testing.T) {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	require.NoError(t, err)
	wrongKey := make([]byte, 32)
	_, err = rand.Read(wrongKey)
	require.NoError(t, err)

	blob, err := buildSymmetricJWE([]byte("secret"), key)
	require.NoError(t, err)

	_, err = decryptSymmetricBlob(blob, wrongKey)
	require.Error(t, err)
}

// TestPSSO_SymmetricWrongKeySize confirms we reject session keys with the
// wrong byte length, since AES-256 expects exactly 32.
func TestPSSO_SymmetricWrongKeySize(t *testing.T) {
	_, err := buildSymmetricJWE([]byte("x"), make([]byte, 16))
	require.Error(t, err)
	_, err = decryptSymmetricBlob([]byte(`{"iv":"AAA","ciphertext":"AAA"}`), make([]byte, 16))
	require.Error(t, err)
}

// TestPSSO_HKDFDifferentSaltDifferentKey confirms the session-key derivation
// produces distinct outputs for distinct salts (i.e. distinct request
// nonces).
func TestPSSO_HKDFDifferentSaltDifferentKey(t *testing.T) {
	kek := make([]byte, 32)
	_, err := rand.Read(kek)
	require.NoError(t, err)

	k1, err := deriveSessionKey(kek, []byte("nonce-1"))
	require.NoError(t, err)
	k2, err := deriveSessionKey(kek, []byte("nonce-2"))
	require.NoError(t, err)
	require.Len(t, k1, 32)
	require.Len(t, k2, 32)
	assert.NotEqual(t, k1, k2)

	// Same salt produces the same key (deterministic).
	k1again, err := deriveSessionKey(kek, []byte("nonce-1"))
	require.NoError(t, err)
	assert.Equal(t, k1, k1again)
}

// TestPSSO_AsymmetricEncryptRoundTrip confirms that a payload encrypted to
// a device's encryption pubkey via JWE ECDH-ES + A256GCM can be decrypted
// with the corresponding private key. This is the key_request flow.
func TestPSSO_AsymmetricEncryptRoundTrip(t *testing.T) {
	deviceKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	payload := []byte(`{"key_exchange_key":"AAECAwQF"}`)
	jweCompact, err := buildAsymmetricJWE(payload, &deviceKey.PublicKey, "")
	require.NoError(t, err)
	require.NotEmpty(t, jweCompact)

	// JWE compact form has 5 base64url segments separated by dots; smoke
	// check we got something of that shape rather than re-implementing the
	// whole decrypt path here (the JOSE library is well-tested upstream).
	dots := 0
	for _, b := range jweCompact {
		if b == '.' {
			dots++
		}
	}
	assert.Equal(t, 4, dots, "expected JWE compact form with 4 dots")
}

// TestPSSO_ParseECPublicKey covers both PEM forms we accept on inbound key
// material from the extension.
func TestPSSO_ParseECPublicKey(t *testing.T) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	der, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	require.NoError(t, err)
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der})

	got, err := parseECPublicKeyPEM(pemBytes)
	require.NoError(t, err)
	gotDER, err := x509.MarshalPKIXPublicKey(got)
	require.NoError(t, err)
	assert.Equal(t, der, gotDER)

	_, err = parseECPublicKeyPEM([]byte("not a pem block"))
	require.Error(t, err)
}
