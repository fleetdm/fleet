package update

import (
	"sync/atomic"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/rs/zerolog/log"
)

type DiskEncryptionRunner struct {
	isRunning atomic.Bool

	// updateRunner is the wrapped Runner where Escrow Buddy will be set as
	// a target.
	updateRunner *Runner

	// configureFn can be set in tests to mock the command executed to
	// configure Escrow Buddy
	configureFn func() error
}

func ApplyDiskEncryptionRunnerMiddleware(runner *Runner) fleet.OrbitConfigReceiver {
	return &DiskEncryptionRunner{updateRunner: runner}
}

func (d *DiskEncryptionRunner) Run(cfg *fleet.OrbitConfig) error {
	log.Debug().Msgf("running disk encryption fetcher middleware, notification: %v, isIdle: %v", cfg.Notifications.RotateDiskEncryptionKey, d.isRunning.Load())

	if d.updateRunner == nil {
		log.Debug().Msg("DiskEncryptionRunner received nil UpdateRunner, this probably indicates that updates are turned off. Skipping any actions related to Disk encryption")
		return nil
	}

	if cfg.Notifications.RotateDiskEncryptionKey && !d.isRunning.Swap(true) {
		defer d.isRunning.Store(false)

		updaterHasTarget := d.updateRunner.HasRunnerOptTarget("escrow-buddy")
		runnerHasLocalHash := d.updateRunner.HasLocalHash("escrow-buddy")
		if !updaterHasTarget || !runnerHasLocalHash {
			log.Info().Msg("refreshing the update runner config with Escrow Buddy targets and hashes")
			log.Debug().Msgf("updater has target: %t, runner has local hash: %t", updaterHasTarget, runnerHasLocalHash)
			return d.setTargetsAndHashes()
		}

		return d.configure()
	}

	return nil
}

func (d *DiskEncryptionRunner) setTargetsAndHashes() error {
	d.updateRunner.AddRunnerOptTarget("escrow-buddy")
	d.updateRunner.updater.SetTargetInfo("escrow-buddy", EscrowBuddyMacOSTarget)
	// we don't want to keep escrow-buddy as a target if we failed to update the
	// cached hashes in the runner.
	if err := d.updateRunner.StoreLocalHash("escrow-buddy"); err != nil {
		log.Debug().Msgf("removing escrow-buddy from target options, error updating local hashes: %s", err)
		d.updateRunner.RemoveRunnerOptTarget("escrow-buddy")
		d.updateRunner.updater.RemoveTargetInfo("escrow-buddy")
		return err
	}
	return nil
}

func (d *DiskEncryptionRunner) configure() error {
	fn := d.configureFn
	if fn == nil {
		fn = func() error {
			/// TODO: test and ensure we don't enter a loop re-enabling escrow buddy while encryption is in process
			cmd := `defaults write /Library/Preferences/com.netflix.Escrow-Buddy.plist GenerateNewKey -bool true`
			return runCmdCollectErr("sh", "-c", cmd)
		}
	}
	return fn()
}
