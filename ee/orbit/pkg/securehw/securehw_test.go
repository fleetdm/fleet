//go:build !windows

package securehw

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-tpm/tpm2/transport/simulator"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

func TestLoadTPMKeyFile(t *testing.T) {
	// Create a temporary directory for test files
	tempDir := t.TempDir()
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()

	// Create a TPM simulator
	sim, err := simulator.OpenSimulator()
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, sim.Close())
	})

	t.Run("missing key file", func(t *testing.T) {
		// Create tpm2SecureHW instance with non-existent key file path
		hw := &tpm2SecureHW{
			device:      sim,
			logger:      logger,
			keyFilePath: filepath.Join(tempDir, "non_existent_key.pem"),
		}

		// Try to load the key file
		privateKey, publicKey, err := hw.loadTPMKeyFile()

		// Should return ErrKeyNotFound
		require.Error(t, err)
		var keyNotFoundErr ErrKeyNotFound
		require.ErrorAs(t, err, &keyNotFoundErr)
		require.Nil(t, privateKey)
		require.Nil(t, publicKey)
	})

	t.Run("invalid key file format", func(t *testing.T) {
		// Create a file with invalid content
		invalidKeyPath := filepath.Join(tempDir, "invalid_key.pem")
		err = os.WriteFile(invalidKeyPath, []byte("this is not a valid TPM key file"), 0600)
		require.NoError(t, err)

		// Create tpm2SecureHW instance
		hw := &tpm2SecureHW{
			device:      sim,
			logger:      logger,
			keyFilePath: invalidKeyPath,
		}

		// Try to load the invalid key file
		privateKey, publicKey, err := hw.loadTPMKeyFile()

		// Should return an error about decoding
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to decode keyfile")
		require.Nil(t, privateKey)
		require.Nil(t, publicKey)
	})

	t.Run("empty key file", func(t *testing.T) {
		// Create an empty file
		emptyKeyPath := filepath.Join(tempDir, "empty_key.pem")
		err = os.WriteFile(emptyKeyPath, []byte{}, 0600)
		require.NoError(t, err)

		// Create tpm2SecureHW instance
		hw := &tpm2SecureHW{
			device:      sim,
			logger:      logger,
			keyFilePath: emptyKeyPath,
		}

		// Try to load the empty key file
		privateKey, publicKey, err := hw.loadTPMKeyFile()

		// Should return an error about decoding
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to decode keyfile")
		require.Nil(t, privateKey)
		require.Nil(t, publicKey)
	})

	t.Run("PEM formatted but not TPM key", func(t *testing.T) {
		// Create a file with valid PEM but not a TPM key
		pemKeyPath := filepath.Join(tempDir, "not_tpm_key.pem")
		pemContent := `-----BEGIN CERTIFICATE-----
MIIBkTCB+wIJAKHHIG...
-----END CERTIFICATE-----`
		err = os.WriteFile(pemKeyPath, []byte(pemContent), 0600)
		require.NoError(t, err)

		// Create tpm2SecureHW instance
		hw := &tpm2SecureHW{
			device:      sim,
			logger:      logger,
			keyFilePath: pemKeyPath,
		}

		// Try to load the PEM file that's not a TPM key
		privateKey, publicKey, err := hw.loadTPMKeyFile()

		// Should return an error about decoding
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to decode keyfile")
		require.Nil(t, privateKey)
		require.Nil(t, publicKey)
	})
}
