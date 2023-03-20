package update

import (
	"github.com/fleetdm/fleet/v4/orbit/pkg/useraction"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/rs/zerolog/log"
)

const maxRetries = 2

type DiskEncryptionRunner struct {
	Fetcher OrbitConfigFetcher

	isIdle bool
}

func ApplyDiskEncryptionRunnerMiddleware(f OrbitConfigFetcher) DiskEncryptionRunner {
	return DiskEncryptionRunner{Fetcher: f}
}

func (d DiskEncryptionRunner) GetConfig() (*fleet.OrbitConfig, error) {
	cfg, err := d.Fetcher.GetConfig()
	if err != nil {
		log.Info().Err(err).Msg("calling GetConfig from DiskEncryptionFetcher")
		return nil, err
	}

	log.Debug().Msgf("running disk encryption fetcher middleware, notification: %v, isIdle: %v", cfg.Notifications.RotateDiskEncryptionKey, d.isIdle)

	if cfg.Notifications.RotateDiskEncryptionKey && !d.isIdle {
		d.isIdle = true
		go func() {
			defer func() { d.isIdle = false }()
			if err := useraction.RotateDiskEncryptionKey(maxRetries); err != nil {
				log.Error().Err(err).Msg("rotating encryption key")
			}
		}()
	}

	return cfg, nil
}
