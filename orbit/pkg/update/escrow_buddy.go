package update

import (
	"fmt"
	"sync"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/rs/zerolog/log"
)

type EscrowBuddyRunner struct {
	// updateRunner is the wrapped Runner where Escrow Buddy will be set as
	// a target.
	updateRunner *Runner

	// configureFn can be set in tests to mock the command executed to
	// configure Escrow Buddy
	configureFn func() error

	installFunc func(path string) error

	cmdMu    sync.Mutex
	lastRun  time.Time
	interval time.Duration
}

func ApplyEscrowBuddyRunnerMiddleware(runner *Runner, interval time.Duration) fleet.OrbitConfigReceiver {
	return &EscrowBuddyRunner{updateRunner: runner, interval: interval}
}

func (e *EscrowBuddyRunner) Run(cfg *fleet.OrbitConfig) error {
	log.Debug().Msgf("running disk encryption fetcher middleware, notification: %t", cfg.Notifications.RotateDiskEncryptionKey)

	if e.updateRunner == nil {
		log.Debug().Msg("DiskEncryptionRunner received nil UpdateRunner, this probably indicates that updates are turned off. Skipping any actions related to Disk encryption")
		return nil
	}

	if !e.cmdMu.TryLock() {
		log.Debug().Msg("a previous instance of DiskEncryptionRunner is currently running, returning early")
		return nil
	}

	defer e.cmdMu.Unlock()
	if time.Since(e.lastRun) < e.interval {
		log.Debug().Msgf("last run (%v) of DiskEncryptionRunner is less than the configured interval (%v), returning early", e.lastRun, e.interval)
		return nil
	}

	if cfg.Notifications.RotateDiskEncryptionKey {
		updaterHasTarget := e.updateRunner.HasRunnerOptTarget("escrowBuddy")
		runnerHasLocalHash := e.updateRunner.HasLocalHash("escrowBuddy")
		if !updaterHasTarget || !runnerHasLocalHash {
			log.Info().Msg("refreshing the update runner config with Escrow Buddy targets and hashes")
			log.Debug().Msgf("updater has target: %t, runner has local hash: %t", updaterHasTarget, runnerHasLocalHash)
			if err := e.setTargetsAndHashes(); err != nil {
				return err
			}
		}

		if err := e.setEscrowBuddyStatus(true); err != nil {
			return err
		}

		e.lastRun = time.Now()

		return nil
	}

	return e.setEscrowBuddyStatus(false)
}

func (d *DiskEncryptionRunner) setTargetsAndHashes() error {
	d.updateRunner.AddRunnerOptTarget("escrowBuddy")
	d.updateRunner.updater.SetTargetInfo("escrowBuddy", EscrowBuddyMacOSTarget)
	// we don't want to keep escrowBuddy as a target if we failed to update the
	// cached hashes in the runner.
	if err := d.updateRunner.StoreLocalHash("escrowBuddy"); err != nil {
		log.Debug().Msgf("removing escrowBuddy from target options, error updating local hashes: %s", err)
		d.updateRunner.RemoveRunnerOptTarget("escrowBuddy")
		d.updateRunner.updater.RemoveTargetInfo("escrowBuddy")
		return err
	}
	return nil
}

// func (d *DiskEncryptionRunner) configure() error {
// 		log.Debug().Msg("checking if Escrow Buddy is configured before")
// 	installed, err := isPkgInstalled("com.netflix.Escrow-Buddy")
// 	if err != nil {
// 		return err
// 	}
// 	if !installed {
// 		log.Debug().Msg("tried to configure Escrow Buddy, but it's not installed yet. Skipping until next loop")
// 		return d.setTargetsAndHashes()
// 	}
//
// 	return d.setEscrowBuddyStatus(true)
// }

// TODO: find a better name
func (d *DiskEncryptionRunner) setEscrowBuddyStatus(enabled bool) error {
	fn := d.configureFn
	if fn == nil {
		fn = func() error {
			log.Debug().Msgf("running defaults write to configure Escrow Buddy with value %t", enabled)
			cmd := fmt.Sprintf("defaults write /Library/Preferences/com.netflix.Escrow-Buddy.plist GenerateNewKey -bool %t", enabled)
			return runCmdCollectErr("sh", "-c", cmd)
		}
	}
	return fn()
}

// TODO: reconsider if we want to keep this check
// func isPkgInstalled(bundleIdentifier string) (bool, error) {
// 	cmd := exec.Command("pkgutil", "--pkg-info-plist", bundleIdentifier)
// 	out, err := cmd.CombinedOutput()
// 	log.Debug().Msgf("pkgutil --pkg-info-plist %s | out: %s, err %s", bundleIdentifier, string(out), err)
// 	if err != nil {
// 		if bytes.Contains(out, []byte("No receipt")) {
// 			return false, nil
// 		}
//
// 		return false, err
// 	}
//
// 	return true, nil
// }
