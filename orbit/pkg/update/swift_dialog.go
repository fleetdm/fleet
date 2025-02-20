package update

import (
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/rs/zerolog/log"
)

type SwiftDialogDownloader struct {
	UpdateRunner *Runner
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

	// TODO: we probably want to ensure that swiftDialog is always installed if we're going to be
	// using it offline.
	if !cfg.Notifications.NeedsMDMMigration && !cfg.Notifications.RenewEnrollmentProfile && !cfg.Notifications.RunSetupExperience {
		log.Debug().Msg("skipping swiftDialog update")
		return nil
	}

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

		if cfg.Notifications.RunSetupExperience {
			// Then update immediately, since we need to get swiftDialog quickly to show the setup
			// experience
			_, err := s.UpdateRunner.UpdateAction()
			if err != nil {
				return err
			}
		}
	}

	return nil
}
