package openframe

import (
	"os"

	"github.com/rs/zerolog/log"
)

const (
	filePath = "/etc/openframe/token.txt"
)

type OpenframeTokenExtractor struct {
	encryptionService *OpenframeEncryptionService
}

func NewOpenframeTokenExtractor(encryptionService *OpenframeEncryptionService) *OpenframeTokenExtractor {
	return &OpenframeTokenExtractor{
		encryptionService: encryptionService,
	}
}

func (te *OpenframeTokenExtractor) ExtractToken() (string, error) {
	// Read the encrypted token from file
	encryptedData, err := os.ReadFile(filePath)
	if err != nil {
		log.Error().Err(err).Msg("Error reading token file")
		return "", err
	}

	// Decrypt the data
	decryptedData, err := te.encryptionService.Decrypt(string(encryptedData))
	if err != nil {
		log.Error().Err(err).Msg("Error decrypting data")
		return "", err
	}

	token := string(decryptedData)
	return token, nil
}
