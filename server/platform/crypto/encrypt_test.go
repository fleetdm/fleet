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
	assert.Equal(t, encryptionVersion, encrypted[0], "first byte should be the version prefix")

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

