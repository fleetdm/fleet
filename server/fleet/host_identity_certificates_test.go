package fleet

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHostIdentityCertificate_PublicKey(t *testing.T) {
	t.Run("P256", func(t *testing.T) {
		testUnmarshalPublicKeyWithCurve(t, elliptic.P256())
	})

	t.Run("P384", func(t *testing.T) {
		testUnmarshalPublicKeyWithCurve(t, elliptic.P384())
	})

	t.Run("unsupported curve", func(t *testing.T) {
		// Generate a key with P521 curve (unsupported)
		key, err := ecdsa.GenerateKey(elliptic.P521(), rand.Reader)
		require.NoError(t, err)

		// Try to create raw public key - should fail
		_, err = CreateECDSAPublicKeyRaw(&key.PublicKey)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported curve")
	})

	t.Run("invalid format - missing 0x04 prefix", func(t *testing.T) {
		// Generate a P256 key but remove the 0x04 prefix
		key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		require.NoError(t, err)

		// Create raw bytes without 0x04 prefix, padded to 32 bytes each
		xBytes := make([]byte, 32)
		yBytes := make([]byte, 32)
		key.X.FillBytes(xBytes)
		key.Y.FillBytes(yBytes)
		pubKeyRaw := xBytes
		pubKeyRaw = append(pubKeyRaw, yBytes...)

		cert := &HostIdentityCertificate{
			PublicKeyRaw: pubKeyRaw,
		}

		_, err = cert.UnmarshalPublicKey()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported EC point format")
	})

	t.Run("invalid format - wrong prefix", func(t *testing.T) {
		// Generate a P256 key but use wrong prefix
		key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		require.NoError(t, err)

		// Create raw bytes with wrong prefix (0x03 instead of 0x04)
		pubKeyRaw, err := CreateECDSAPublicKeyRaw(&key.PublicKey)
		require.NoError(t, err)
		pubKeyRaw[0] = 0x03

		cert := &HostIdentityCertificate{
			PublicKeyRaw: pubKeyRaw,
		}

		_, err = cert.UnmarshalPublicKey()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported EC point format")
	})

	t.Run("empty public key", func(t *testing.T) {
		cert := &HostIdentityCertificate{
			PublicKeyRaw: []byte{},
		}

		_, err := cert.UnmarshalPublicKey()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported EC point format")
	})

	t.Run("unsupported key length", func(t *testing.T) {
		// Create a key with unsupported length (not 65 or 97 bytes)
		pubKeyRaw := make([]byte, 33) // 33 bytes total (0x04 + 16 + 16)
		pubKeyRaw[0] = 0x04

		cert := &HostIdentityCertificate{
			PublicKeyRaw: pubKeyRaw,
		}

		_, err := cert.UnmarshalPublicKey()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unknown curve")
	})
}

func testUnmarshalPublicKeyWithCurve(t *testing.T, curve elliptic.Curve) {
	// Generate a key with the specified curve
	originalKey, err := ecdsa.GenerateKey(curve, rand.Reader)
	require.NoError(t, err)

	pubKeyRaw, err := CreateECDSAPublicKeyRaw(&originalKey.PublicKey)
	require.NoError(t, err)

	// Create HostIdentityCertificate with the raw public key
	cert := &HostIdentityCertificate{
		PublicKeyRaw: pubKeyRaw,
	}

	// Unmarshal the public key
	unmarshaledKey, err := cert.UnmarshalPublicKey()
	require.NoError(t, err)
	require.NotNil(t, unmarshaledKey)

	// Verify the unmarshaled key matches the original
	assert.Equal(t, curve, unmarshaledKey.Curve)
	assert.True(t, originalKey.X.Cmp(unmarshaledKey.X) == 0, "X coordinates should match")
	assert.True(t, originalKey.Y.Cmp(unmarshaledKey.Y) == 0, "Y coordinates should match")
	assert.True(t, originalKey.PublicKey.Equal(unmarshaledKey), "Public keys should be equal")
}
