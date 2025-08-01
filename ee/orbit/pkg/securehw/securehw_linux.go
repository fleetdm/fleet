//go:build linux

package securehw

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"github.com/google/go-tpm/tpm2/transport/linuxtpm"
	"github.com/rs/zerolog"
)

const tpm20DevicePath = "/dev/tpmrm0"

// Creates a new SecureHW instance using TPM 2.0 for Linux.
// It attempts to open the TPM device using the provided configuration.
func newSecureHW(metadataDir string, logger zerolog.Logger) (SecureHW, error) {
	if metadataDir == "" {
		return nil, errors.New("required metadata directory not set")
	}

	logger.Info().Msg("opening TPM 2.0 resource manager")

	// Open the TPM 2.0 resource manager, which
	// - Provides managed access to TPM resources, allowing multiple applications to share the TPM safely.
	// - Used by the TPM2 Access Broker and Resource Manager (tpm2-abrmd or the kernel resource manager).
	device, err := linuxtpm.Open(tpm20DevicePath)
	if err != nil {
		return nil, ErrSecureHWUnavailable{
			Message: fmt.Sprintf("failed to open TPM 2.0 device %q: %s", tpm20DevicePath, err.Error()),
		}
	}

	logger.Info().Str("device_path", tpm20DevicePath).Msg("successfully opened TPM 2.0 resource manager")

	return &tpm2SecureHW{
		device:      device,
		logger:      logger.With().Str("component", "securehw-tpm").Logger(),
		keyFilePath: filepath.Join(metadataDir, constant.FleetHTTPSignatureTPMKeyFileName),
	}, nil
}
