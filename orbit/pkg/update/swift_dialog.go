package update

import (
	"sync"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/rs/zerolog/log"
)

type SwiftDialogDownloader struct {
	UpdateRunner                   *Runner
	triggeredSetupExperienceUpdate bool
	runMu                          sync.Mutex
}

type SwiftDialogDownloaderOptions struct {
	// UpdateRunner is the wrapped Runner where swiftDialog will be set as a target. It is responsible for
	// actually ensuring that swiftDialog is installed and updated via the designated TUF server.
	UpdateRunner *Runner
}

func ApplySwiftDialogDownloaderMiddleware(
	runner *Runner,
) fleet.OrbitConfigReceiver {
	return &SwiftDialogDownloader{UpdateRunner: runner}
}

func (s *SwiftDialogDownloader) Run(cfg *fleet.OrbitConfig) error {
	log.Debug().Msg("running swiftDialog installer middleware")

	if cfg == nil {
		log.Debug().Msg("SwiftDialogDownloader received nil config")
		return nil
	}

	if s.UpdateRunner == nil {
		log.Debug().Msg("SwiftDialogDownloader received nil UpdateRunner, this probably indicates that updates are turned off. Skipping any actions related to swiftDialog")
		return nil
	}

	if !s.runMu.TryLock() {
		log.Debug().Msg("SwiftDialogDownloader: a previous instance is currently running, returning early")
		return nil
	}
	defer s.runMu.Unlock()

	// For #25928 we are going to always install swiftDialog as a target
	updaterHasTarget := s.UpdateRunner.HasRunnerOptTarget("swiftDialog")
	runnerHasLocalHash := s.UpdateRunner.HasLocalHash("swiftDialog")
	if !updaterHasTarget || !runnerHasLocalHash {
		log.Info().Msg("refreshing the update runner config with swiftDialog targets and hashes")
		log.Debug().Msgf("updater has target: %t, runner has local hash: %t", updaterHasTarget, runnerHasLocalHash)
		s.UpdateRunner.AddRunnerOptTarget("swiftDialog")
		s.UpdateRunner.updater.SetTargetInfo("swiftDialog", SwiftDialogMacOSTarget)
		// we don't want to keep swiftDialog as a target if we failed to update the
		// cached hashes in the runner.
		if err := s.UpdateRunner.StoreLocalHash("swiftDialog"); err != nil {
			log.Debug().Msgf("removing swiftDialog from target options, error updating local hashes: %s", err)
			s.UpdateRunner.RemoveRunnerOptTarget("swiftDialog")
			s.UpdateRunner.updater.RemoveTargetInfo("swiftDialog")
			return err
		}
	}

	// If we're running setup experience and we have the hashes we need to make sure we trigger
	// an immediate update to get SwiftDialog installed and usable.
	if cfg.Notifications.RunSetupExperience && !s.triggeredSetupExperienceUpdate {
		s.triggeredSetupExperienceUpdate = true
		log.Debug().Msg("SwiftDialogDownloader: triggering update to install swiftDialog immediately during setup experience")
		_, err := s.UpdateRunner.UpdateAction()
		if err != nil {
			return err
		}
	}

	return nil
}
