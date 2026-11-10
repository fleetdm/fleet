package mysql

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
)

func encrypt(plainText []byte, privateKey string) ([]byte, error) {
	block, err := aes.NewCipher([]byte(privateKey))
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

func decrypt(encrypted []byte, privateKey string) ([]byte, error) {
	block, err := aes.NewCipher([]byte(privateKey))
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

// decryptOrPassthrough attempts to decrypt the data. If decryption fails
// (e.g. because the data is legacy cleartext), it returns the data as-is.
func decryptOrPassthrough(data []byte, privateKey string) []byte {
	if len(privateKey) == 0 || len(data) == 0 {
		return data
	}
	decrypted, err := decrypt(data, privateKey)
	if err != nil {
		return data
	}
	return decrypted
}
