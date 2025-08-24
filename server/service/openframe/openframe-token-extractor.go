package openframe

import (
	"os"

	"github.com/rs/zerolog/log"
)

type OpenframeTokenExtractor struct {
	encryptionService *OpenframeEncryptionService
	tokenFilePath     string
}

func NewOpenframeTokenExtractor(encryptionService *OpenframeEncryptionService, tokenFilePath string) *OpenframeTokenExtractor {
	log.Info().Msgf("Token file path: %s", tokenFilePath)
	return &OpenframeTokenExtractor{
		encryptionService: encryptionService,
		tokenFilePath:     tokenFilePath,
	}
}

func (te *OpenframeTokenExtractor) ExtractToken() (string, error) {
	// Read the encrypted token from file
	encryptedData, err := os.ReadFile(te.tokenFilePath)
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
