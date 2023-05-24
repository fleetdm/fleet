package update

import (
	"sync"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/rs/zerolog/log"
)

type runCmdFunc func() error

// RenewEnrollmentProfileConfigFetcher is a kind of middleware that wraps an
// OrbitConfigFetcher and detects if the fleet server sent a notification to
// renew the enrollment profile. If so, it runs the command (as root) to
// bootstrap the renewal of the profile on the device (the user still needs to
// execute some manual steps to accept the new profile).
//
// It ensures only one renewal command is executed at any given time, and that
// it doesn't re-execute the command until a certain amount of time has passed.
type RenewEnrollmentProfileConfigFetcher struct {
	// Fetcher is the OrbitConfigFetcher that will be wrapped. It is responsible
	// for actually returning the orbit configuration or an error.
	Fetcher OrbitConfigFetcher
	// Frequency is the minimum amount of time that must pass between two
	// executions of the profile renewal command.
	Frequency time.Duration

	// for tests, to be able to mock command execution. If nil, will use
	// runRenewEnrollmentProfile.
	runCmdFn runCmdFunc

	// ensures only one command runs at a time, protects access to lastRun
	cmdMu   sync.Mutex
	lastRun time.Time
}

func ApplyRenewEnrollmentProfileConfigFetcherMiddleware(fetcher OrbitConfigFetcher, frequency time.Duration) OrbitConfigFetcher {
	return &RenewEnrollmentProfileConfigFetcher{Fetcher: fetcher, Frequency: frequency}
}

// GetConfig calls the wrapped Fetcher's GetConfig method, and if the fleet
// server set the renew enrollment profile flag to true, executes the command
// to renew the enrollment profile.
func (h *RenewEnrollmentProfileConfigFetcher) GetConfig() (*fleet.OrbitConfig, error) {
	cfg, err := h.Fetcher.GetConfig()

	// TODO: download and use swiftDialog following the same patterns we
	// use for Nudge.
	//
	// updaterHasTarget := h.UpdateRunner.HasRunnerOptTarget("swiftDialog")
	// runnerHasLocalHash := h.UpdateRunner.HasLocalHash("swiftDialog")
	// if !updaterHasTarget || !runnerHasLocalHash {
	//         log.Info().Msg("refreshing the update runner config with swiftDialog targets and hashes")
	//         log.Debug().Msgf("updater has target: %t, runner has local hash: %t", updaterHasTarget, runnerHasLocalHash)
	//         return cfg, h.setTargetsAndHashes()
	// }

	if err == nil && cfg.Notifications.RenewEnrollmentProfile {
		if h.cmdMu.TryLock() {
			defer h.cmdMu.Unlock()

			// Note that the macOS notification popup will be shown periodically
			// until the Fleet server gets notified that the device is now properly
			// enrolled (after the user's manual steps, and osquery reporting the
			// updated mdm enrollment).
			// See https://github.com/fleetdm/fleet/pull/9409#discussion_r1084382455
			if time.Since(h.lastRun) > h.Frequency {
				fn := h.runCmdFn
				if fn == nil {
					fn = runRenewEnrollmentProfile
				}
				if err := fn(); err != nil {
					log.Info().Err(err).Msg("calling /usr/bin/profiles to renew enrollment profile failed")
				} else {
					h.lastRun = time.Now()
					log.Info().Msg("successfully called /usr/bin/profiles to renew enrollment profile")
				}
			} else {
				log.Debug().Msg("skipped calling /usr/bin/profiles to renew enrollment profile, last run was too recent")
			}
		}
	}
	return cfg, err
}
