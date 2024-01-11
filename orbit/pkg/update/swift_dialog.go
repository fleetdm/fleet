package update

import (
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/rs/zerolog/log"
)

type SwiftDialogDownloader struct {
	Fetcher      OrbitConfigFetcher
	UpdateRunner *Runner
}

type SwiftDialogDownloaderOptions struct {
	// UpdateRunner is the wrapped Runner where swiftDialog will be set as a target. It is responsible for
	// actually ensuring that swiftDialog is installed and updated via the designated TUF server.
	UpdateRunner *Runner
}

func ApplySwiftDialogDownloaderMiddleware(
	f OrbitConfigFetcher,
	runner *Runner,
) OrbitConfigFetcher {
	return &SwiftDialogDownloader{Fetcher: f, UpdateRunner: runner}
}

func (s *SwiftDialogDownloader) GetConfig() (*fleet.OrbitConfig, error) {
	log.Debug().Msg("running swiftDialog installer middleware")
	cfg, err := s.Fetcher.GetConfig()
	if err != nil {
		return nil, err
	}

	if cfg == nil {
		log.Debug().Msg("SwiftDialogDownloader received nil config")
		return nil, nil
	}

	if s.UpdateRunner == nil {
		log.Debug().Msg("SwiftDialogDownloader received nil UpdateRunner, this probably indicates that updates are turned off. Skipping any actions related to swiftDialog")
		return cfg, nil
	}

	if !cfg.Notifications.NeedsMDMMigration && !cfg.Notifications.RenewEnrollmentProfile {
		return cfg, nil
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
			return cfg, err
		}
	}

	return cfg, nil
}
