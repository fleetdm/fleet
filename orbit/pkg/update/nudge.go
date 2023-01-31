package update

import (
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/rs/zerolog/log"
)

// NudgeConfigFetcher is a kind of middleware that wraps an OrbitConfigFetcher and a Runner.
// It checks the config supplied by the wrapped OrbitConfigFetcher to detects whether the Fleet
// server has supplied a Nudge config. If so, it sets Nudge as a target on the wrapped Runner.
type NudgeConfigFetcher struct {
	// Fetcher is the OrbitConfigFetcher that will be wrapped. It is responsible
	// for actually returning the orbit configuration or an error.
	Fetcher OrbitConfigFetcher

	// UpdateRunner is the wrapped Runner where Nudge will be set as a target. It is responsible for
	// actually ensuring that Nudge is installed and updated via the designated TUF server.
	UpdateRunner *Runner
}

func ApplyNudgeConfigFetcherMiddleware(f OrbitConfigFetcher, u *Runner) OrbitConfigFetcher {
	return &NudgeConfigFetcher{Fetcher: f, UpdateRunner: u}
}

// GetConfig calls the wrapped Fetcher's GetConfig method, and detects if the
// Fleet server has supplied a Nudge config. If so, it ensures that Nudge is
// installed and updated via the designated TUF server.
func (n *NudgeConfigFetcher) GetConfig() (*fleet.OrbitConfig, error) {
	cfg, err := n.Fetcher.GetConfig()
	if err != nil {
		log.Info().Err(err).Msg("calling GetConfig from NudgeConfigFetcher")
		return nil, err
	}

	if cfg != nil && cfg.NudgeConfig != nil {
		var found bool
		for _, t := range n.UpdateRunner.opt.Targets {
			if t == "nudge" {
				found = true
			}
		}

		if !found {
			log.Info().Msg("adding nudge as target")
			n.UpdateRunner.UpdateRunnerOptTargets("nudge")
			n.UpdateRunner.updater.SetExtensionsTargetInfo(
				"nudge",
				"macos",
				"stable",
				"nudge.app.tar.gz",
				[]string{"Nudge.app", "Contents", "MacOS", "Nudge"},
			)
		}
	}

	return cfg, err
}
