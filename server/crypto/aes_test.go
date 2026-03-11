package crypto

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncryptDecryptAESGCM(t *testing.T) {
	// 32-byte key for AES-256 (simple test key to avoid false positive secret scanning warnings)
	key := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

	tests := []struct {
		name      string
		plaintext []byte
	}{
		{"empty", []byte{}},
		{"short", []byte("hello")},
		{"medium", []byte("this is a longer test message for encryption")},
		{"with special chars", []byte("hello\x00world\ntest\ttab")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encrypted, err := EncryptAESGCM(tt.plaintext, key)
			require.NoError(t, err, "EncryptAESGCM()")

			// Encrypted should be longer than plaintext (nonce + auth tag)
			assert.Greater(t, len(encrypted), len(tt.plaintext), "encrypted length should be > plaintext length")

			decrypted, err := DecryptAESGCM(encrypted, key)
			require.NoError(t, err, "DecryptAESGCM()")

			assert.True(t, bytes.Equal(decrypted, tt.plaintext), "DecryptAESGCM() result mismatch")
		})
	}
}

func TestDecryptAESGCM_InvalidKey(t *testing.T) {
	key := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	wrongKey := "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"

	plaintext := []byte("secret message")
	encrypted, err := EncryptAESGCM(plaintext, key)
	require.NoError(t, err, "EncryptAESGCM()")

	_, err = DecryptAESGCM(encrypted, wrongKey)
	assert.Error(t, err, "DecryptAESGCM() with wrong key should fail")
}

func TestDecryptAESGCM_MalformedCiphertext(t *testing.T) {
	key := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

	tests := []struct {
		name       string
		ciphertext []byte
	}{
		{"empty", []byte{}},
		{"one byte", []byte{0x01}},
		{"too short for nonce", []byte("short")},
		{"exactly nonce size", make([]byte, 12)}, // GCM nonce is 12 bytes, but no ciphertext
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := DecryptAESGCM(tt.ciphertext, key)
			assert.Error(t, err, "DecryptAESGCM() with %s ciphertext should fail", tt.name)
		})
	}
}

func TestEncryptAESGCM_InvalidKeyLength(t *testing.T) {
	tests := []struct {
		name    string
		keyLen  int
		wantErr bool
	}{
		{"empty key", 0, true},
		{"too short", 5, true},
		{"AES-128 rejected", 16, true},
		{"AES-192 rejected", 24, true},
		{"AES-256 accepted", 32, false},
		{"too long", 64, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := strings.Repeat("a", tt.keyLen)

			_, err := EncryptAESGCM([]byte("test"), key)
			if tt.wantErr {
				assert.Error(t, err, "EncryptAESGCM() should fail")
			} else {
				assert.NoError(t, err, "EncryptAESGCM() should succeed")
			}
		})
	}
}

func TestDecryptAESGCM_InvalidKeyLength(t *testing.T) {
	// First encrypt with valid key
	validKey := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	encrypted, err := EncryptAESGCM([]byte("test"), validKey)
	require.NoError(t, err, "EncryptAESGCM()")

	tests := []struct {
		name   string
		keyLen int
	}{
		{"empty key", 0},
		{"too short", 5},
		{"AES-128 rejected", 16},
		{"AES-192 rejected", 24},
		{"too long", 64},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := strings.Repeat("b", tt.keyLen)

			_, err := DecryptAESGCM(encrypted, key)
			assert.Error(t, err, "DecryptAESGCM() with %d-byte key should fail", tt.keyLen)
		})
	}
}

func TestEncryptAESGCM_UniqueNonces(t *testing.T) {
	key := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	plaintext := []byte("same message")

	encrypted1, err := EncryptAESGCM(plaintext, key)
	require.NoError(t, err, "first EncryptAESGCM()")

	encrypted2, err := EncryptAESGCM(plaintext, key)
	require.NoError(t, err, "second EncryptAESGCM()")

	// Same plaintext should produce different ciphertext due to random nonce
	assert.NotEqual(t, encrypted1, encrypted2, "encrypting same plaintext twice should produce different ciphertext")
}
