package packaging

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/rs/zerolog/log"
)

// Notarize will notarize using rcodesign with App Store Connect API keys.
// Note that the provided path must be a .zip, .dmg or .pkg.
func Notarize(path, apiKeyID, apiKeyIssuer, apiKeyContent string) error {
	keyPath, err := writeAPIKeys(apiKeyIssuer, apiKeyID, apiKeyContent)
	if err != nil {
		return fmt.Errorf("write API keys: %w", err)
	}
	defer os.Remove(keyPath)

	// Create a temporary directory for rcodesign output
	tmpDir, err := os.MkdirTemp("", "rcodesign-")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	log.Info().Str("path", path).Msg("submitting file for notarization with App Store Connect API")

	// Use rcodesign to notarize (without stapling since we can't staple a zip)
	out, err := exec.Command("rcodesign",
		"notarize",
		path,
		"--api-issuer", apiKeyIssuer,
		"--api-key", apiKeyID,
	).CombinedOutput()
	if err != nil {
		log.Error().Str("output", string(out)).Msg("rcodesign notarize failed")
		return fmt.Errorf("rcodesign notarize: %w", err)
	}

	log.Info().Str("output", string(out)).Msg("notarization completed with App Store Connect API")
	return nil
}
