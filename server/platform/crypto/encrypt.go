// Package crypto provides shared encryption utilities for Fleet.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
)

// encryptionVersion is prepended to all newly encrypted values so we can
// distinguish encrypted data from legacy cleartext on read.
const encryptionVersion byte = 0x01

// Encrypt encrypts plainText using AES-GCM with the given key.
// The key must be 16, 24, or 32 bytes (AES-128, AES-192, AES-256).
// The returned ciphertext is prefixed with a version byte, followed by
// the nonce and the GCM-sealed ciphertext.
func Encrypt(plainText []byte, key string) ([]byte, error) {
	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return nil, fmt.Errorf("create new cipher: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create new gcm: %w", err)
	}

	nonce := make([]byte, aesGCM.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}

	sealed := aesGCM.Seal(nonce, nonce, plainText, nil)
	// Prepend version byte so DecryptOrPassthrough can distinguish
	// encrypted values from legacy cleartext.
	result := make([]byte, 1+len(sealed))
	result[0] = encryptionVersion
	copy(result[1:], sealed)
	return result, nil
}

// Decrypt decrypts data that was encrypted with Encrypt using the same key.
// The first byte must be the version prefix written by Encrypt.
func Decrypt(encrypted []byte, key string) ([]byte, error) {
	if len(encrypted) < 2 {
		return nil, errors.New("decrypting: ciphertext too short")
	}
	if encrypted[0] != encryptionVersion {
		return nil, fmt.Errorf("decrypting: unknown version byte 0x%02x", encrypted[0])
	}
	// Strip version byte.
	payload := encrypted[1:]

	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return nil, fmt.Errorf("create new cipher: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create new gcm: %w", err)
	}

	nonceSize := aesGCM.NonceSize()
	if len(payload) < nonceSize {
		return nil, errors.New("decrypting: ciphertext too short")
	}

	nonce, ciphertext := payload[:nonceSize], payload[nonceSize:]

	decrypted, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypting: %w", err)
	}

	return decrypted, nil
}

// DecryptOrPassthrough decrypts data that starts with the version prefix.
// If the data does not start with the version prefix, it is treated as
// legacy cleartext and returned as-is. If the data starts with the version
// prefix but decryption fails (wrong key, corrupted data), an error is
// returned instead of silently passing through garbage.
func DecryptOrPassthrough(data []byte, key string) ([]byte, error) {
	if len(key) == 0 || len(data) == 0 {
		return data, nil
	}
	if data[0] != encryptionVersion {
		// Legacy cleartext, return as-is.
		return data, nil
	}
	// Data has the version prefix, so it must decrypt or fail.
	return Decrypt(data, key)
}
