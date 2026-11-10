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
	_, err := Decrypt([]byte("some-long-enough-data-for-nonce-check"), "too-short")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "create new cipher")
}

func TestDecryptCiphertextTooShort(t *testing.T) {
	_, err := Decrypt([]byte("short"), testKey)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ciphertext too short")
}

func TestDecryptOrPassthrough(t *testing.T) {
	t.Run("decrypts encrypted data", func(t *testing.T) {
		plaintext := []byte("bootstrap-token-value")
		encrypted, err := Encrypt(plaintext, testKey)
		require.NoError(t, err)

		result := DecryptOrPassthrough(encrypted, testKey)
		assert.Equal(t, plaintext, result)
	})

	t.Run("passes through cleartext data", func(t *testing.T) {
		cleartext := []byte("legacy-cleartext-value")
		result := DecryptOrPassthrough(cleartext, testKey)
		assert.Equal(t, cleartext, result)
	})

	t.Run("empty data returns empty", func(t *testing.T) {
		result := DecryptOrPassthrough([]byte{}, testKey)
		assert.Empty(t, result)
	})

	t.Run("nil data returns nil", func(t *testing.T) {
		result := DecryptOrPassthrough(nil, testKey)
		assert.Nil(t, result)
	})

	t.Run("empty key returns data as-is", func(t *testing.T) {
		data := []byte("some-data")
		result := DecryptOrPassthrough(data, "")
		assert.Equal(t, data, result)
	})
}
