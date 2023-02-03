package update

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

func TestNudgeConfigFetcher(t *testing.T) {
	u := &Updater{opt: Options{Targets: make(map[string]TargetInfo)}}
	r := &Runner{updater: u}

	cfg := &fleet.OrbitConfig{}
	var f OrbitConfigFetcher
	f = &dummyConfigFetcher{cfg: cfg}
	f = ApplyNudgeConfigFetcherMiddleware(f, r)

	// nudge is not added to targets if nudge config is not present
	cfg.NudgeConfig = nil
	gotCfg, err := f.GetConfig()
	require.NoError(t, err)
	require.Equal(t, cfg, gotCfg)
	targets := r.updater.opt.Targets
	require.Len(t, targets, 0)

	// nudge is added to targets when nudge config is present
	cfg.NudgeConfig = &fleet.NudgeConfig{}
	gotCfg, err = f.GetConfig()
	require.NoError(t, err)
	require.Equal(t, cfg, gotCfg)
	targets = r.updater.opt.Targets
	require.Len(t, targets, 1)

	expected := TargetInfo{
		Platform:             "macos",
		Channel:              "stable",
		TargetFile:           "nudge.app.tar.gz",
		ExtractedExecSubPath: []string{"Nudge.app", "Contents", "MacOS", "Nudge"},
	}
	ti, ok := targets["nudge"]
	require.True(t, ok)
	require.EqualValues(t, expected, ti)
}
