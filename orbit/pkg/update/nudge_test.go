package update

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestNudge(t *testing.T) {
	testingSuite := new(nudgeTestSuite)
	testingSuite.s = &testingSuite.Suite
	suite.Run(t, testingSuite)
}

type nudgeTestSuite struct {
	suite.Suite
	withTUF
}

func (s *nudgeTestSuite) TestNudgeConfigFetcherAddNudge() {
	t := s.T()
	tmpDir := t.TempDir()
	updater := &Updater{
		client: s.client,
		opt:    Options{Targets: make(map[string]TargetInfo), RootDirectory: tmpDir},
	}
	runner := &Runner{updater: updater, localHashes: make(map[string][]byte)}
	interval := time.Minute
	cfg := &fleet.OrbitConfig{}
	nudgePath := "nudge/macos/stable/nudge.app.tar.gz"
	runNudgeFn := func(execPath, configPath string) error {
		return nil
	}

	var f OrbitConfigFetcher = &dummyConfigFetcher{cfg: cfg}
	f = ApplyNudgeConfigFetcherMiddleware(f, NudgeConfigFetcherOptions{
		UpdateRunner: runner,
		RootDir:      tmpDir,
		Interval:     interval,
		runNudgeFn:   runNudgeFn,
	})
	configPath := filepath.Join(tmpDir, nudgeConfigFile)

	// nudge is not added to targets if nudge config is not present
	cfg.NudgeConfig = nil
	gotCfg, err := f.GetConfig()
	require.NoError(t, err)
	require.Equal(t, cfg, gotCfg)
	targets := runner.updater.opt.Targets
	require.Len(t, targets, 0)

	// set the config
	cfg.NudgeConfig, err = fleet.NewNudgeConfig(fleet.MacOSUpdates{MinimumVersion: optjson.SetString("11"), Deadline: optjson.SetString("2022-01-04")})
	require.NoError(t, err)

	// there's an error when the remote repo doesn't have the target yet
	gotCfg, err = f.GetConfig()
	require.ErrorContains(t, err, "tuf: file not found")
	require.Equal(t, cfg, gotCfg)

	// add nuge to the remote
	s.addRemoteTarget(nudgePath)

	// nudge is added to targets when nudge config is present
	gotCfg, err = f.GetConfig()
	require.NoError(t, err)
	require.Equal(t, cfg, gotCfg)
	targets = runner.updater.opt.Targets
	require.Len(t, targets, 1)
	ti, ok := targets["nudge"]
	require.True(t, ok)
	require.EqualValues(t, NudgeMacOSTarget, ti)

	// override the custom check since we don't really have an executable
	ti.CustomCheckExec = func(path string) error {
		require.Contains(t, path, "/Nudge.app/Contents/MacOS/Nudge")
		return nil
	}
	runner.updater.opt.Targets["nudge"] = ti

	// trigger an update check
	updated, err := runner.UpdateAction()
	require.NoError(t, err)
	require.True(t, updated)

	// doesn't re-update after an update
	gotCfg, err = f.GetConfig()
	require.NoError(t, err)
	require.Equal(t, cfg, gotCfg)
	updated, err = runner.UpdateAction()
	require.NoError(t, err)
	require.False(t, updated)

	// runner hashes are updated
	b, ok := runner.localHashes["nudge"]
	require.True(t, ok)
	require.NotEmpty(t, b)

	// a config is created on the next run after install
	gotCfg, err = f.GetConfig()
	require.NoError(t, err)
	require.Equal(t, cfg, gotCfg)
	configBytes, err := os.ReadFile(configPath)
	require.NoError(t, err)
	var savedConfig fleet.NudgeConfig
	err = json.Unmarshal(configBytes, &savedConfig)
	require.NoError(t, err)
	require.Equal(t, cfg.NudgeConfig, &savedConfig)

	// config on disk changes if the config from the server changes
	cfg.NudgeConfig.OSVersionRequirements[0].RequiredMinimumOSVersion = "13.1.1"
	gotCfg, err = f.GetConfig()
	require.NoError(t, err)
	require.Equal(t, cfg, gotCfg)
	configBytes, err = os.ReadFile(configPath)
	require.NoError(t, err)
	savedConfig = fleet.NudgeConfig{}
	err = json.Unmarshal(configBytes, &savedConfig)
	require.NoError(t, err)
	require.Equal(t, cfg.NudgeConfig, &savedConfig)

	// config permissions are always validated and set to the right value
	err = os.Chmod(configPath, constant.DefaultFileMode)
	require.NoError(t, err)
	gotCfg, err = f.GetConfig()
	require.NoError(t, err)
	require.Equal(t, cfg, gotCfg)
	fileInfo, err := os.Stat(configPath)
	require.NoError(t, err)
	require.Equal(t, fileInfo.Mode(), nudgeConfigFileMode)

	configBytes, err = os.ReadFile(configPath)
	require.NoError(t, err)
	savedConfig = fleet.NudgeConfig{}
	err = json.Unmarshal(configBytes, &savedConfig)
	require.NoError(t, err)
	require.Equal(t, cfg.NudgeConfig, &savedConfig)

	// nudge is removed from targets when the config config is present
	cfg.NudgeConfig = nil
	gotCfg, err = f.GetConfig()
	require.NoError(t, err)
	require.Equal(t, cfg, gotCfg)
	targets = runner.updater.opt.Targets
	require.Empty(t, targets)
	ti, ok = targets["nudge"]
	require.False(t, ok)
	require.Empty(t, ti)
}
