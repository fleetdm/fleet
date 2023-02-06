package update

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"github.com/fleetdm/fleet/v4/pkg/secure"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

func TestNudgeConfigFetcherAddNudge(t *testing.T) {
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

func TestNudgeConfigFetcherRemoveNudge(t *testing.T) {
	// setup mock updater with test directory to mock nudge install
	rootDir := t.TempDir()
	nudgePath := filepath.Join(rootDir, binDir, "nudge")
	require.NoError(t, secure.MkdirAll(nudgePath, constant.DefaultDirMode))
	require.DirExists(t, nudgePath)
	nudgeInfo := TargetInfo{
		Platform:             "macos",
		Channel:              "stable",
		TargetFile:           "nudge.app.tar.gz",
		ExtractedExecSubPath: []string{"Nudge.app", "Contents", "MacOS", "Nudge"},
	}
	u := &Updater{opt: Options{RootDirectory: rootDir, Targets: map[string]TargetInfo{"nudge": nudgeInfo}}}

	// setup mock runner with test channels to mock runner interrupt
	cancel := make(chan struct{})
	done := make(chan struct{})
	var canceled bool
	go func() {
		select {
		case <-cancel:
			canceled = true
		case <-time.After(1 * time.Second):
			canceled = false
		}
		close(done)
	}()
	r := &Runner{cancel: cancel, opt: RunnerOptions{Targets: []string{"nudge"}}, updater: u}

	// settup config fetcher
	cfg := &fleet.OrbitConfig{}
	var f OrbitConfigFetcher
	f = &dummyConfigFetcher{cfg: cfg}
	f = ApplyNudgeConfigFetcherMiddleware(f, r)

	// if nudge config is present
	cfg.NudgeConfig = &fleet.NudgeConfig{}
	gotCfg, err := f.GetConfig()
	require.NoError(t, err)
	require.Equal(t, cfg, gotCfg)
	targets := r.updater.opt.Targets
	require.Len(t, targets, 1)
	gotInfo, ok := targets["nudge"]
	require.True(t, ok)
	require.EqualValues(t, nudgeInfo, gotInfo)
	require.DirExists(t, nudgePath)
	require.False(t, canceled)

	// if nudge config is not present, nudge is removed from targets and filesystem
	cfg.NudgeConfig = nil
	gotCfg, err = f.GetConfig()
	require.NoError(t, err)
	require.Equal(t, cfg, gotCfg)
	targets = r.updater.opt.Targets
	require.Len(t, targets, 0)

	<-done
	require.NoDirExists(t, nudgePath)
	require.True(t, canceled)
}
