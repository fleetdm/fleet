package update

import (
	"sync/atomic"

	"github.com/fleetdm/fleet/v4/orbit/pkg/useraction"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/rs/zerolog/log"
)

const maxRetries = 2

type DiskEncryptionRunner struct {
	isRunning atomic.Bool
}

func ApplyDiskEncryptionRunnerMiddleware() fleet.OrbitConfigReceiver {
	return &DiskEncryptionRunner{}
}

func (d *DiskEncryptionRunner) Run(cfg *fleet.OrbitConfig) error {
	log.Debug().Msgf("running disk encryption fetcher middleware, notification: %v, isIdle: %v", cfg.Notifications.RotateDiskEncryptionKey, d.isRunning.Load())

	if cfg.Notifications.RotateDiskEncryptionKey && !d.isRunning.Swap(true) {
		go func() {
			defer d.isRunning.Store(false)
			if err := useraction.RotateDiskEncryptionKey(maxRetries); err != nil {
				log.Error().Err(err).Msg("rotating encryption key")
			}
		}()
	}

	return nil
}
