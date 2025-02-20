package update

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"github.com/fleetdm/fleet/v4/orbit/pkg/execuser"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/rs/zerolog/log"
)

const (
	nudgeConfigFile     = "nudge-config.json"
	nudgeConfigFileMode = os.FileMode(constant.DefaultWorldReadableFileMode)
)

// NudgeConfigReceiver is a kind of middleware that wraps an OrbitConfigFetcher and a Runner.
// It checks the config supplied by the wrapped OrbitConfigFetcher to detects whether the Fleet
// server has supplied a Nudge config. If so, it sets Nudge as a target on the wrapped Runner.
type NudgeConfigReceiver struct {
	opt NudgeConfigFetcherOptions
	// ensures only one command runs at a time, protects access to lastRun
	cmdMu   sync.Mutex
	lastRun time.Time

	// launchErr is set if Nudge fails to launch. If launchErr is set, we won't try to
	// launch Nudge again.
	launchErr *nudgeLaunchErr
}

type NudgeConfigFetcherOptions struct {
	// UpdateRunner is the wrapped Runner where Nudge will be set as a target. It is responsible for
	// actually ensuring that Nudge is installed and updated via the designated TUF server.
	UpdateRunner *Runner
	// RootDir is where the Nudge configuration will be stored
	RootDir string
	// Interval is the minimum amount of time that must pass to launch
	// Nudge
	Interval time.Duration
	// runNudgeFn can be set in tests to mock the command executed to
	// run Nudge.
	runNudgeFn func(execPath, configPath string) error
}

func ApplyNudgeConfigReceiverMiddleware(opt NudgeConfigFetcherOptions) fleet.OrbitConfigReceiver {
	return &NudgeConfigReceiver{opt: opt}
}

// GetConfig calls the wrapped Fetcher's GetConfig method, and detects if the
// Fleet server has supplied a Nudge config.
//
// If a Nudge config is supplied, it:
//
// - ensures that Nudge is installed and updated via the designated TUF server.
// - ensures that Nudge is opened at an interval given by n.frequency with the
// provided config.
func (n *NudgeConfigReceiver) Run(cfg *fleet.OrbitConfig) error {
	log.Debug().Msg("running nudge config fetcher middleware")

	if cfg == nil {
		log.Debug().Msg("NudgeConfigReceiver received nil config")
		return nil
	}

	if n.opt.UpdateRunner == nil {
		log.Debug().Msg("NudgeConfigReceiver received nil UpdateRunner, this probably indicates that updates are turned off. Skipping any actions related to Nudge")
		return nil
	}

	if cfg.NudgeConfig == nil {
		log.Debug().Msg("empty nudge config, removing nudge as target")
		// TODO(roberto): by early returning and removing the target from the
		// runner/updater we ensure Nudge won't be opened/updated again
		// but we don't actually remove the file from disk. We
		// knowingly decided to do this as a post MVP optimization.
		n.opt.UpdateRunner.RemoveRunnerOptTarget("nudge")
		n.opt.UpdateRunner.updater.RemoveTargetInfo("nudge")
		return nil
	}

	updaterHasTarget := n.opt.UpdateRunner.HasRunnerOptTarget("nudge")
	runnerHasLocalHash := n.opt.UpdateRunner.HasLocalHash("nudge")
	if !updaterHasTarget || !runnerHasLocalHash {
		log.Info().Msg("refreshing the update runner config with Nudge targets and hashes")
		log.Debug().Msgf("updater has target: %t, runner has local hash: %t", updaterHasTarget, runnerHasLocalHash)
		return n.setTargetsAndHashes()
	}

	if err := n.configure(*cfg.NudgeConfig); err != nil {
		log.Info().Err(err).Msg("nudge configuration")
		return err
	}

	if err := n.launch(); err != nil {
		log.Info().Err(err).Msg("nudge launch")
		return err
	}

	return nil
}

func (n *NudgeConfigReceiver) setTargetsAndHashes() error {
	n.opt.UpdateRunner.AddRunnerOptTarget("nudge")
	n.opt.UpdateRunner.updater.SetTargetInfo("nudge", NudgeMacOSTarget)
	// we don't want to keep nudge as a target if we failed to update the
	// cached hashes in the runner.
	if err := n.opt.UpdateRunner.StoreLocalHash("nudge"); err != nil {
		log.Debug().Msgf("removing nudge from target options, error updating local hashes: %s", err)
		n.opt.UpdateRunner.RemoveRunnerOptTarget("nudge")
		n.opt.UpdateRunner.updater.RemoveTargetInfo("nudge")
		return err
	}
	return nil
}

func (n *NudgeConfigReceiver) configure(nudgeCfg fleet.NudgeConfig) error {
	jsonCfg, err := json.Marshal(nudgeCfg)
	if err != nil {
		return err
	}

	cfgFile := filepath.Join(n.opt.RootDir, nudgeConfigFile)
	writeConfig := func() error {
		return os.WriteFile(cfgFile, jsonCfg, constant.DefaultWorldReadableFileMode)
	}

	fileInfo, err := os.Stat(cfgFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return writeConfig()
		}
		return err
	}

	// ensure the config file has the right permissions, a call to
	// WriteFile preserves existing permissions if the file already exists,
	// and previous versions of orbit set the permissions of this file to
	// constant.DefaultFileMode.
	if fileInfo.Mode() != nudgeConfigFileMode {
		log.Info().Msgf("%s config file had wrong permissions (%v) setting permissions to %v", cfgFile, fileInfo.Mode(), nudgeConfigFileMode)
		if err := os.Chmod(cfgFile, nudgeConfigFileMode); err != nil {
			return fmt.Errorf("ensuring permissions of config file, chmod %q: %w", cfgFile, err)
		}
	}

	// this not only an optimization, but mostly a safeguard: if the file
	// has been tampered and contains very large contents, we don't
	// want to load them into memory.
	if fileInfo.Size() != int64(len(jsonCfg)) {
		log.Debug().Msg("configuring nudge: local file has different size than remote, writing remote config")
		return writeConfig()
	}

	fileBytes, err := os.ReadFile(cfgFile)
	if err != nil {
		return err
	}

	if !bytes.Equal(fileBytes, jsonCfg) {
		log.Debug().Msg("configuring nudge: local file is different than remote, writing remote config")
		return writeConfig()
	}

	return nil
}

func (n *NudgeConfigReceiver) launch() error {
	cfgFile := filepath.Join(n.opt.RootDir, nudgeConfigFile)

	if n.cmdMu.TryLock() {
		defer n.cmdMu.Unlock()

		if time.Since(n.lastRun) > n.opt.Interval {
			nudge, err := n.opt.UpdateRunner.updater.localTarget("nudge")
			if err != nil {
				return err
			}

			// before moving forward, check that the file at the
			// path is the file we're about to open hasn't been
			// tampered with.
			meta, err := n.opt.UpdateRunner.updater.Lookup("nudge")
			if err != nil {
				return err
			}
			// if we can't find the file, or the hash doesn't match
			// make sure nudge is added as a target and the hashes
			// are refreshed
			if err := checkFileHash(meta, nudge.Path); err != nil {
				n.launchErr = nil // reset launchErr since we're dealing with a different file
				return n.setTargetsAndHashes()
			}

			// if we have a prior launch error, we won't try to launch nudge again
			if n.launchErr != nil {
				log.Info().Msgf("Nudge disabled since %s due to launch error: %v", n.launchErr.timestamp.Format("2006-01-02"), n.launchErr)
				n.lastRun = time.Now()
				return nil
			}

			fn := n.opt.runNudgeFn
			if fn == nil {
				fn = func(appPath, configPath string) error {
					// TODO(roberto): when an user selects "Later" from the
					// Nudge defer menu, the Nudge UI will be shown the
					// next time Nudge is launched. If for some reason orbit
					// restarts (eg: an update) and the user has a pending
					// OS update, we might show Nudge more than one time
					// every n.frequency.
					//
					// Note that this only happens for the "Later" option,
					// all other options behave as expected and Nudge will
					// respect the time chosen (eg: next day) and it won't
					// show up even if it's opened multiple times in that
					// interval.
					log.Info().Msg("running Nudge")
					_, err := execuser.Run(
						appPath,
						execuser.WithArg("-json-url", configPath),
					)
					return err
				}
			}

			if err := fn(nudge.DirPath, fmt.Sprintf("file://%s", cfgFile)); err != nil {
				if exitErr, ok := err.(*exec.ExitError); ok {
					launchErr := &nudgeLaunchErr{
						err:       err,
						exitCode:  exitErr.ExitCode(),
						detail:    string(exitErr.Stderr),
						cfgFile:   cfgFile,
						timestamp: time.Now(),
					}
					n.launchErr = launchErr
					return fmt.Errorf("opening Nudge with config %q: %w", cfgFile, launchErr)
				}
				return fmt.Errorf("opening Nudge with config %q: %w", cfgFile, err)
			}

			n.lastRun = time.Now()
		}
	}

	return nil
}

type nudgeLaunchErr struct {
	err       error
	exitCode  int
	detail    string
	cfgFile   string
	timestamp time.Time
}

func (e *nudgeLaunchErr) Error() string {
	return fmt.Sprintf("%v: %s", e.err, e.detail)
}
