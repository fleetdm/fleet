package update

import (
	"fmt"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/rs/zerolog/log"
)

// NudgeConfigFetcher is a kind of middleware that wraps an OrbitConfigFetcher and detects if the
// Fleet server has supplied a Nudge config. If so, it ensures that Nudge is installed and updated
// via the designated TUF server.
type NudgeConfigFetcher struct {
	// Fetcher is the OrbitConfigFetcher that will be wrapped. It is responsible
	// for actually returning the orbit configuration or an error.
	Fetcher OrbitConfigFetcher

	UpdateRunner *Runner

	// // for tests, to be able to mock command execution. If nil, will use
	// // runRenewEnrollmentProfile.
	// runCmdFn func() error

	// // ensures only one command runs at a time, protects access to lastRun
	// cmdMu   sync.Mutex
	// lastRun time.Time
}

func ApplyNudgeConfigFetcherMiddleware(f OrbitConfigFetcher, u *Runner) OrbitConfigFetcher {
	return &NudgeConfigFetcher{Fetcher: f, UpdateRunner: u}
}

// GetConfig calls the wrapped Fetcher's GetConfig method, and detects if the
// Fleet server has supplied a Nudge config. If so, it ensures that Nudge is
// installed and updated via the designated TUF server.
func (n *NudgeConfigFetcher) GetConfig() (*fleet.OrbitConfig, error) {
	log.Info().Msg("called into NudgeConfigFetcher")

	cfg, err := n.Fetcher.GetConfig()
	if err != nil {
		log.Info().Err(err).Msg("calling GetConfig from NudgeConfigFetcher")
		return nil, err
	}

	// if cfg != nil && len(cfg.NudgeConfig) > 0 {
	if cfg != nil {
		var found bool
		for _, t := range n.UpdateRunner.opt.Targets {
			if t == "nudge" {
				found = true
			}
		}

		log.Info().Msg(fmt.Sprint("found nudge? ", found))

		// if !found && n.cmdMu.TryLock() {
		// 	defer n.cmdMu.Unlock()
		if !found {
			log.Info().Msg("adding nudge as target")

			n.UpdateRunner.UpdateRunnerOptTargets("nudge")
			n.UpdateRunner.updater.SetExtensionsTargetInfo(
				"nudge",
				"macos-app",
				"stable",
				"nudge.app.tar.gz",
				[]string{"Nudge.app", "Contents", "MacOS", "Nudge"}, // TODO confirm
			)

			// TODO start a new runner to launch nudge?

		}
	}
	log.Info().Msg("returning from NudgeConfigFetcher")

	return cfg, err
}
