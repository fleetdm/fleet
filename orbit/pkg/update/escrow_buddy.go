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

	// runCmdFunc can be set in tests to mock the command executed to
	// configure Escrow Buddy
	runCmdFunc func(cmd string, args ...string) error

	cmdMu    sync.Mutex
	lastRun  time.Time
	interval time.Duration
}

func ApplyEscrowBuddyRunnerMiddleware(runner *Runner, interval time.Duration) fleet.OrbitConfigReceiver {
	return &EscrowBuddyRunner{updateRunner: runner, interval: interval}
}

func (e *EscrowBuddyRunner) Run(cfg *fleet.OrbitConfig) error {
	log.Debug().Msgf("running escrow buddy middleware, notification: %t", cfg.Notifications.RotateDiskEncryptionKey)

	if e.updateRunner == nil {
		log.Debug().Msg("escrow buddy received nil UpdateRunner, this probably indicates that updates are turned off. Skipping any actions related to Disk encryption")
		return nil
	}

	if !e.cmdMu.TryLock() {
		log.Debug().Msg("a previous instance of EscrowBuddyRunner is currently running, returning early")
		return nil
	}

	defer e.cmdMu.Unlock()
	if time.Since(e.lastRun) < e.interval {
		log.Debug().Msgf("last run (%v) of EscrowBuddyRunner is less than the configured interval (%v), returning early", e.lastRun, e.interval)
		return nil
	}

	updaterHasTarget := e.updateRunner.HasRunnerOptTarget("escrowBuddy")
	if cfg.Notifications.RotateDiskEncryptionKey {
		runnerHasLocalHash := e.updateRunner.HasLocalHash("escrowBuddy")
		if !updaterHasTarget || !runnerHasLocalHash {
			log.Info().Msg("refreshing the update runner config with Escrow Buddy targets and hashes")
			log.Debug().Msgf("updater has target: %t, runner has local hash: %t", updaterHasTarget, runnerHasLocalHash)
			if err := e.setTargetsAndHashes(); err != nil {
				return err
			}
		}

		if err := e.setGenerateNewKeyTo(true); err != nil {
			return err
		}

		e.lastRun = time.Now()

		return nil
	}

	if updaterHasTarget {
		return e.setGenerateNewKeyTo(false)
	}

	return nil
}

func (e *EscrowBuddyRunner) setTargetsAndHashes() error {
	e.updateRunner.AddRunnerOptTarget("escrowBuddy")
	e.updateRunner.updater.SetTargetInfo("escrowBuddy", EscrowBuddyMacOSTarget)
	// we don't want to keep escrowBuddy as a target if we failed to update the
	// cached hashes in the runner.
	if err := e.updateRunner.StoreLocalHash("escrowBuddy"); err != nil {
		log.Debug().Msgf("removing escrowBuddy from target options, error updating local hashes: %s", err)
		e.updateRunner.RemoveRunnerOptTarget("escrowBuddy")
		e.updateRunner.updater.RemoveTargetInfo("escrowBuddy")
		return err
	}
	return nil
}

func (e *EscrowBuddyRunner) setGenerateNewKeyTo(enabled bool) error {
	log.Debug().Msgf("running defaults write to configure Escrow Buddy with value %t", enabled)
	cmd := fmt.Sprintf("defaults write /Library/Preferences/com.netflix.Escrow-Buddy.plist GenerateNewKey -bool %t", enabled)
	fn := e.runCmdFunc
	if fn == nil {
		fn = runCmdCollectErr
	}
	return fn("sh", "-c", cmd)
}
