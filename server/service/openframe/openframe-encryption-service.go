package openframe

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"fmt"

	"github.com/rs/zerolog/log"
)

type OpenframeEncryptionService struct {
	encryptionKey string
}

func NewOpenframeEncryptionService(encryptionKey string) *OpenframeEncryptionService {
	return &OpenframeEncryptionService{
		encryptionKey: encryptionKey,
	}
}

func (es *OpenframeEncryptionService) Encrypt(data []byte) (string, error) {
	block, err := aes.NewCipher([]byte(es.encryptionKey))
	if err != nil {
		log.Error().Err(err).Msg("Error creating cipher")
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		log.Error().Err(err).Msg("Error creating GCM")
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	ciphertext := gcm.Seal(nonce, nonce, data, nil)

	// Encode to base64
	base64Token := base64.StdEncoding.EncodeToString(ciphertext)
	return base64Token, nil
}

func (es *OpenframeEncryptionService) Decrypt(data string) ([]byte, error) {
	// Decode base64 string to bytes
	encryptedData, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		log.Error().Err(err).Msg("Error decoding base64 data")
		return nil, err
	}

	block, err := aes.NewCipher([]byte(es.encryptionKey))
	if err != nil {
		log.Error().Err(err).Msg("Error creating cipher")
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	if len(encryptedData) < gcm.NonceSize() {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce := encryptedData[:gcm.NonceSize()]
	ciphertext := encryptedData[gcm.NonceSize():]

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}
