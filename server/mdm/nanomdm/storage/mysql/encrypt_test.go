package mysql

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// AES-256 requires a 32-byte key.
const testKey = "aaaaaaaaaabbbbbbbbbbccccccccccdd"

func TestEncryptDecryptRoundTrip(t *testing.T) {
	plaintext := []byte("my-secret-value-1234")
	encrypted, err := encrypt(plaintext, testKey)
	require.NoError(t, err)
	assert.NotEqual(t, plaintext, encrypted, "encrypted should differ from plaintext")

	decrypted, err := decrypt(encrypted, testKey)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestDecryptWrongKey(t *testing.T) {
	encrypted, err := encrypt([]byte("secret"), testKey)
	require.NoError(t, err)

	_, err = decrypt(encrypted, "zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz")
	require.Error(t, err, "decrypting with wrong key should fail")
}

func TestDecryptCiphertextTooShort(t *testing.T) {
	_, err := decrypt([]byte("short"), testKey)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ciphertext too short")
}

func TestDecryptOrPassthrough(t *testing.T) {
	t.Run("decrypts encrypted data", func(t *testing.T) {
		plaintext := []byte("bootstrap-token-value")
		encrypted, err := encrypt(plaintext, testKey)
		require.NoError(t, err)

		result := decryptOrPassthrough(encrypted, testKey)
		assert.Equal(t, plaintext, result)
	})

	t.Run("passes through cleartext data", func(t *testing.T) {
		cleartext := []byte("legacy-cleartext-value")
		result := decryptOrPassthrough(cleartext, testKey)
		assert.Equal(t, cleartext, result)
	})

	t.Run("empty data returns empty", func(t *testing.T) {
		result := decryptOrPassthrough([]byte{}, testKey)
		assert.Empty(t, result)
	})

	t.Run("nil data returns nil", func(t *testing.T) {
		result := decryptOrPassthrough(nil, testKey)
		assert.Nil(t, result)
	})

	t.Run("empty key returns data as-is", func(t *testing.T) {
		data := []byte("some-data")
		result := decryptOrPassthrough(data, "")
		assert.Equal(t, data, result)
	})
}
