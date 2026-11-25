package crypto

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// AES-256 requires a 32-byte key.
const testKey = "aaaaaaaaaabbbbbbbbbbccccccccccdd"

func TestEncryptDecryptRoundTrip(t *testing.T) {
	plaintext := []byte("my-secret-value-1234")
	encrypted, err := Encrypt(plaintext, testKey)
	require.NoError(t, err)
	assert.NotEqual(t, plaintext, encrypted, "encrypted should differ from plaintext")
	assert.Equal(t, byte(encryptionVersion), encrypted[0], "first byte should be the version prefix")

	decrypted, err := Decrypt(encrypted, testKey)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestDecryptWrongKey(t *testing.T) {
	encrypted, err := Encrypt([]byte("secret"), testKey)
	require.NoError(t, err)

	_, err = Decrypt(encrypted, "zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz")
	require.Error(t, err, "decrypting with wrong key should fail")
}

func TestEncryptInvalidKeyLength(t *testing.T) {
	_, err := Encrypt([]byte("data"), "too-short")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "create new cipher")
}

func TestDecryptInvalidKeyLength(t *testing.T) {
	// Needs version prefix to get past version check
	data := append([]byte{encryptionVersion}, []byte("some-long-enough-data-for-nonce")...)
	_, err := Decrypt(data, "too-short")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "create new cipher")
}

func TestDecryptCiphertextTooShort(t *testing.T) {
	_, err := Decrypt([]byte{encryptionVersion}, testKey)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ciphertext too short")
}

func TestDecryptNoVersionPrefix(t *testing.T) {
	_, err := Decrypt([]byte("no-version-prefix-data-here"), testKey)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown version byte")
}

func TestDecryptOrPassthrough(t *testing.T) {
	t.Run("decrypts encrypted data", func(t *testing.T) {
		plaintext := []byte("bootstrap-token-value")
		encrypted, err := Encrypt(plaintext, testKey)
		require.NoError(t, err)

		result, err := DecryptOrPassthrough(encrypted, testKey)
		require.NoError(t, err)
		assert.Equal(t, plaintext, result)
	})

	t.Run("passes through cleartext data", func(t *testing.T) {
		cleartext := []byte("legacy-cleartext-value")
		result, err := DecryptOrPassthrough(cleartext, testKey)
		require.NoError(t, err)
		assert.Equal(t, cleartext, result)
	})

	t.Run("empty data returns empty", func(t *testing.T) {
		result, err := DecryptOrPassthrough([]byte{}, testKey)
		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("nil data returns nil", func(t *testing.T) {
		result, err := DecryptOrPassthrough(nil, testKey)
		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("empty key returns data as-is", func(t *testing.T) {
		data := []byte("some-data")
		result, err := DecryptOrPassthrough(data, "")
		require.NoError(t, err)
		assert.Equal(t, data, result)
	})

	t.Run("wrong key on versioned data returns error", func(t *testing.T) {
		encrypted, err := Encrypt([]byte("secret"), testKey)
		require.NoError(t, err)

		_, err = DecryptOrPassthrough(encrypted, "zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz")
		require.Error(t, err, "should error, not silently return garbage")
	})

	t.Run("corrupted versioned data returns error", func(t *testing.T) {
		encrypted, err := Encrypt([]byte("secret"), testKey)
		require.NoError(t, err)

		// Corrupt a byte in the ciphertext
		encrypted[len(encrypted)-1] ^= 0xFF
		_, err = DecryptOrPassthrough(encrypted, testKey)
		require.Error(t, err, "should error on corrupted data")
	})
}
