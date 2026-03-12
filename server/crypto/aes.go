// Package crypto provides cryptographic utilities for Fleet.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
)

// EncryptAESGCM encrypts plaintext using AES-256-GCM with the given key.
// The key must be 32 bytes for AES-256.
// Returns [nonce || ciphertext].
func EncryptAESGCM(plainText []byte, key string) ([]byte, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("invalid key size: got %d bytes, AES-256 requires exactly 32 bytes", len(key))
	}

	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create gcm: %w", err)
	}

	nonce := make([]byte, aesGCM.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}

	return aesGCM.Seal(nonce, nonce, plainText, nil), nil
}

// DecryptAESGCM decrypts ciphertext that was encrypted with EncryptAESGCM.
// The key must be the same 32-byte key used for encryption.
func DecryptAESGCM(encrypted []byte, key string) ([]byte, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("invalid key size: got %d bytes, AES-256 requires exactly 32 bytes", len(key))
	}

	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create gcm: %w", err)
	}

	nonceSize := aesGCM.NonceSize()
	if len(encrypted) < nonceSize+1 {
		return nil, fmt.Errorf("malformed ciphertext: length %d is less than minimum required %d", len(encrypted), nonceSize+1)
	}

	nonce, ciphertext := encrypted[:nonceSize], encrypted[nonceSize:]
	return aesGCM.Open(nil, nonce, ciphertext, nil)
}
