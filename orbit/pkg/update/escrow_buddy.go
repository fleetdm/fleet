package update

import (
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

// EscrowBuddyRunner sets up [Escrow Buddy][1] to rotate FileVault keys on
// macOS without user interaction. This runner:
//
// - Ensures Escrow Buddy is added as a target for the update runner, so the
// authorization plugin is downloaded and installed.
// - Shells out to call `defaults` to configure Escrow Buddy according to
// server instructions provided via notifications.
//
// [1]: https://github.com/macadmins/escrow-buddy
type EscrowBuddyRunner struct {
	// updateRunner is the wrapped Runner where Escrow Buddy will be set as
	// a target.
	updateRunner *Runner
	// runCmdFunc can be set in tests to mock the command executed to
	// configure Escrow Buddy
	runCmdFunc func(cmd string, args ...string) error
	// runMu guards runs to prevent multiple Run calls happening at the
	// same time.
	runMu sync.Mutex
	// lastRun is used to guarantee that the run interval is enforced
	lastRun time.Time
	// interval defines how often Run is allowed to perform work
	interval time.Duration
}

// NewEscrowBuddyRunner returns a new instance configured with the provided values
func NewEscrowBuddyRunner(runner *Runner, interval time.Duration) fleet.OrbitConfigReceiver {
	return &EscrowBuddyRunner{updateRunner: runner, interval: interval}
}

func (e *EscrowBuddyRunner) Run(cfg *fleet.OrbitConfig) error {
	log.Debug().Msgf("EscrowBuddyRunner: notification: %t", cfg.Notifications.RotateDiskEncryptionKey)

	if e.updateRunner == nil {
		log.Info().Msg("EscrowBuddyRunner: received nil UpdateRunner, this probably indicates that updates are turned off. Skipping any actions related to Disk encryption")
		return nil
	}

	if !e.runMu.TryLock() {
		log.Debug().Msg("EscrowBuddyRunner: a previous instance is currently running, returning early")
		return nil
	}

	defer e.runMu.Unlock()
	if time.Since(e.lastRun) < e.interval {
		log.Debug().Msgf("EscrowBuddyRunner: last run (%v) is less than the configured interval (%v), returning early", e.lastRun, e.interval)
		return nil
	}

	updaterHasTarget := e.updateRunner.HasRunnerOptTarget("escrowBuddy")
	// if the notification is false, it could mean that we shouldn't do
	// anything at all (eg: MDM is not configured) or that this host
	// doesn't need to rotate the key.
	//
	// if Escrow Buddy is a TUF target, it means that we tried to rotate
	// the key before, and we must disable it to keep the local state as
	// instructed by the server.
	if !cfg.Notifications.RotateDiskEncryptionKey {
		if updaterHasTarget {
			log.Debug().Msg("EscrowBuddyRunner: disabling disk encryption rotation")
			e.lastRun = time.Now()
			return e.setGenerateNewKeyTo(false)
		}

		log.Debug().Msg("EscrowBuddyRunner: skipping any actions related to disk encryption")
		return nil
	}

	runnerHasLocalHash := e.updateRunner.HasLocalHash("escrowBuddy")
	if !updaterHasTarget || !runnerHasLocalHash {
		log.Info().Msg("refreshing the update runner config with Escrow Buddy targets and hashes")
		log.Debug().Msgf("updater has target: %t, runner has local hash: %t", updaterHasTarget, runnerHasLocalHash)
		if err := e.setTargetsAndHashes(); err != nil {
			return fmt.Errorf("setting Escrow Buddy targets and hashes: %w", err)
		}
	}

	// Some macOS updates and upgrades reset the authorization database to its default state
	// which will deactivate Escrow Buddy and prevent FileVault key generation upon next login.
	log.Debug().Msg("EscrowBuddyRunner: re-enable Escrow Buddy in the authorization database")
	if err := e.setAuthDBSetup(); err != nil {
		return fmt.Errorf("failed to re-enable Escrow Buddy in the authorization database, err: %w", err)
	}

	log.Debug().Msg("EscrowBuddyRunner: enabling disk encryption rotation")
	if err := e.setGenerateNewKeyTo(true); err != nil {
		return fmt.Errorf("enabling disk encryption rotation: %w", err)
	}

	e.lastRun = time.Now()
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

func (e *EscrowBuddyRunner) setAuthDBSetup() error {
	log.Debug().Msg("ready to re-enable Escrow Buddy in the authorization database")
	cmd := "/Library/Security/SecurityAgentPlugins/Escrow\\ Buddy.bundle/Contents/Resources/AuthDBSetup.sh"
	fn := e.runCmdFunc
	if fn == nil {
		fn = runCmdCollectErr
	}
	return fn("sh", "-c", cmd)
}
