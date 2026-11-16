// Package crypto provides shared encryption utilities for Fleet.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
)

// Encrypt encrypts plainText using AES-GCM with the given key.
// The key must be 16, 24, or 32 bytes (AES-128, AES-192, AES-256).
// The returned ciphertext has the nonce prepended.
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

	return aesGCM.Seal(nonce, nonce, plainText, nil), nil
}

// Decrypt decrypts data that was encrypted with Encrypt using the same key.
func Decrypt(encrypted []byte, key string) ([]byte, error) {
	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return nil, fmt.Errorf("create new cipher: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create new gcm: %w", err)
	}

	nonceSize := aesGCM.NonceSize()
	if len(encrypted) < nonceSize {
		return nil, fmt.Errorf("decrypting: ciphertext too short")
	}

	nonce, ciphertext := encrypted[:nonceSize], encrypted[nonceSize:]

	decrypted, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypting: %w", err)
	}

	return decrypted, nil
}

// DecryptOrPassthrough attempts to decrypt the data. If decryption fails
// (e.g. because the data is legacy cleartext), it returns the data as-is.
func DecryptOrPassthrough(data []byte, key string) []byte {
	if len(key) == 0 || len(data) == 0 {
		return data
	}
	decrypted, err := Decrypt(data, key)
	if err != nil {
		return data
	}
	return decrypted
}
