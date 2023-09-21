package update

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"github.com/fleetdm/fleet/v4/orbit/pkg/logging"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/rs/zerolog/log"
)

// FlagRunner is a specialized runner to periodically check and update flags from Fleet
// It is designed with Execute and Interrupt functions to be compatible with oklog/run
//
// It uses an OrbitConfigFetcher (which may be the OrbitClient with additional middleware), along
// with FlagUpdateOptions to connect to Fleet
type FlagRunner struct {
	configFetcher OrbitConfigFetcher
	opt           FlagUpdateOptions
	cancel        chan struct{}
}

// FlagUpdateOptions is options provided for the flag update runner
type FlagUpdateOptions struct {
	// CheckInterval is the interval to check for updates
	CheckInterval time.Duration
	// RootDir is the root directory for orbit state
	RootDir string
}

// NewFlagRunner creates a new runner with provided options
// The runner must be started with Execute
func NewFlagRunner(configFetcher OrbitConfigFetcher, opt FlagUpdateOptions) *FlagRunner {
	return &FlagRunner{
		configFetcher: configFetcher,
		opt:           opt,
		cancel:        make(chan struct{}),
	}
}

// Execute starts the loop checking for updates
func (r *FlagRunner) Execute() error {
	log.Debug().Msg("starting flag updater")

	ticker := time.NewTicker(r.opt.CheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-r.cancel:
			return nil
		case <-ticker.C:
			log.Debug().Msg("calling flags update")
			didUpdate, err := r.DoFlagsUpdate()
			if err != nil {
				logging.LogErrIfEnvNotSet(constant.SilenceEnrollLogErrorEnvVar, err, "flags updates failed")
			}
			if didUpdate {
				log.Info().Msg("flags updated, exiting")
				return nil
			}
			ticker.Reset(r.opt.CheckInterval)
		}
	}
}

// Interrupt is the oklog/run interrupt method that stops orbit when interrupt is received
func (r *FlagRunner) Interrupt(err error) {
	close(r.cancel)
	log.Debug().Err(err).Msg("interrupt for flags updater")
}

// DoFlagsUpdate checks for update of flags from Fleet
// It gets the flags from the Fleet server, and compares them to locally stored flagfile (if it exists)
// If the flag comparison from disk and server are not equal, it writes the flags to disk, and returns true
func (r *FlagRunner) DoFlagsUpdate() (bool, error) {
	flagFileExists := true

	// first off try and read osquery.flags from disk
	osqueryFlagMapFromFile, err := readFlagFile(r.opt.RootDir)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return false, err
		}
		// flag file may not exist on disk on first "boot"
		flagFileExists = false
	}

	// next GetConfig from Fleet API
	config, err := r.configFetcher.GetConfig()
	if err != nil {
		return false, fmt.Errorf("error getting flags from fleet: %w", err)
	}
	if len(config.Flags) == 0 {
		// command_line_flags not set in YAML, nothing to do
		return false, nil
	}

	osqueryFlagMapFromFleet, err := getFlagsFromJSON(config.Flags)
	if err != nil {
		return false, fmt.Errorf("error parsing flags: %w", err)
	}

	// compare both flags, if they are equal, nothing to do
	if flagFileExists && reflect.DeepEqual(osqueryFlagMapFromFile, osqueryFlagMapFromFleet) {
		return false, nil
	}

	// flags are not equal, write the fleet flags to disk
	err = writeFlagFile(r.opt.RootDir, osqueryFlagMapFromFleet)
	if err != nil {
		return false, fmt.Errorf("error writing flags to disk: %w", err)
	}
	return true, nil
}

// ExtensionRunner is a specialized runner to periodically check and update flags from Fleet
// It is designed with Execute and Interrupt functions to be compatible with oklog/run
//
// It uses an an OrbitConfigFetcher (which may be the OrbitClient with additional middleware), along
// with ExtensionUpdateOptions and updateRunner to connect to Fleet.
type ExtensionRunner struct {
	configFetcher OrbitConfigFetcher
	opt           ExtensionUpdateOptions
	cancel        chan struct{}
	updateRunner  *Runner
}

// ExtensionUpdateOptions is options provided for the extensions fetch/update runner
type ExtensionUpdateOptions struct {
	// CheckInterval is the interval to check for updates
	CheckInterval time.Duration
	// RootDir is the root directory for orbit state
	RootDir string
}

// NewExtensionConfigUpdateRunner creates a new runner with provided options
// The runner must be started with Execute
func NewExtensionConfigUpdateRunner(configFetcher OrbitConfigFetcher, opt ExtensionUpdateOptions, updateRunner *Runner) *ExtensionRunner {
	return &ExtensionRunner{
		configFetcher: configFetcher,
		opt:           opt,
		cancel:        make(chan struct{}),
		updateRunner:  updateRunner,
	}
}

// Execute starts the loop checking for updates
func (r *ExtensionRunner) Execute() error {
	log.Debug().Msg("starting extension runner")

	ticker := time.NewTicker(r.opt.CheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-r.cancel:
			return nil
		case <-ticker.C:
			log.Debug().Msg("calling /config API to fetch/update extensions")
			extensionsCleared, err := r.DoExtensionConfigUpdate()
			if err != nil {
				logging.LogErrIfEnvNotSet(constant.SilenceEnrollLogErrorEnvVar, err, "ext update failed")
			}
			if extensionsCleared {
				log.Info().Msg("extensions were cleared on the server")
				return nil
			}
		}
		ticker.Reset(r.opt.CheckInterval)
	}
}

// Interrupt is the oklog/run interrupt method that stops orbit when interrupt is received
func (r *ExtensionRunner) Interrupt(err error) {
	close(r.cancel)
	log.Debug().Err(err).Msg(("interrupt extension runner"))
}

// DoExtensionConfigUpdate calls the /config API endpoint to grab extensions from Fleet
// It parses the extensions, computes the local hash, and writes the binary path to extension.load file
//
// It returns a (bool, error), where bool indicates whether orbit should restart
// It only returns (true, nil) when extensions were previously configured and now are cleared
func (r *ExtensionRunner) DoExtensionConfigUpdate() (bool, error) {
	// call "/config" API endpoint to grab orbit configs from Fleet
	config, err := r.configFetcher.GetConfig()
	if err != nil {
		// we do not want orbit to restart
		return false, fmt.Errorf("extensionsUpdate: error getting extensions config from fleet: %w", err)
	}

	extensionAutoLoadFile := filepath.Join(r.opt.RootDir, "extensions.load")
	if len(config.Extensions) == 0 {
		// Extensions from Fleet is empty
		// this can be either because of:
		// 1. the default state, where no extensions are configured to begin with, or
		// 2. extensions were previously configured, but now are deleted and reverted to empty state
		switch stat, err := os.Stat(extensionAutoLoadFile); {
		// Handle case 1, where our autoload file does not exist, so there is nothing to update and no error
		case errors.Is(err, os.ErrNotExist):
			log.Debug().Msg(extensionAutoLoadFile + " not found, nothing to update")
			// we do not want orbit to restart
			return false, nil
		case err == nil:
			// handle case 2: create/truncate the extensions.load file and let the runner interrupt, so that
			// osquery can't startup without the extensions that were previously loaded
			// WriteFile will create the file if it doesn't exist, and it handles Close for us
			if stat.Size() > 0 {
				err := os.WriteFile(extensionAutoLoadFile, []byte(""), constant.DefaultFileMode)
				if err != nil {
					// we do not want orbit to restart
					return false, fmt.Errorf("extensionsUpdate: error creating file %s, %w", extensionAutoLoadFile, err)
				}
				// we want to return true here, and restart with the empty extensions.load file
				// so that we "unload" the previously loaded extensions
				return true, nil
			}
			// we do not want orbit to restart
			return false, nil
		default:
			// we do not want orbit to restart, just log the error
			return false, fmt.Errorf("stat file: %s", extensionAutoLoadFile)
		}
	}

	log.Debug().Str("extensions", string(config.Extensions)).Msg("received extensions configuration")

	var extensions fleet.Extensions
	err = json.Unmarshal(config.Extensions, &extensions)
	if err != nil {
		// we do not want orbit to restart
		return false, fmt.Errorf("error unmarshing json extensions config from fleet: %w", err)
	}

	// Filter out extensions not targeted to this OS.
	extensions.FilterByHostPlatform(runtime.GOOS)

	var sb strings.Builder
	for extensionName, extensionInfo := range extensions {
		// infer filename from extension name
		// osquery enforces .ext, so we just add that
		// we expect filename to match extension name
		filename := extensionName + ".ext"

		// All Windows executables must end with `.exe`.
		if runtime.GOOS == "windows" {
			filename = filename + ".exe"
		}

		// we don't want path traversal and the like in the filename
		if strings.Contains(filename, "..") || strings.Contains(filename, "/") || strings.Contains(filename, "\\") {
			log.Info().Msgf("invalid characters found in filename (%s) for extension (%s): skipping", filename, extensionName)
			continue
		}

		// add "extensions/" as a prefix to the targetName, since that's the namespace we expect for extensions on TUF
		targetName := "extensions/" + extensionName
		platform := extensionInfo.Platform
		channel := extensionInfo.Channel

		rootDir := r.updateRunner.updater.opt.RootDirectory

		// update our view of targets
		r.updateRunner.AddRunnerOptTarget(targetName)
		r.updateRunner.updater.SetTargetInfo(targetName, TargetInfo{Platform: platform, Channel: channel, TargetFile: filename})

		// the full path to where the extension would be on disk, for e.g. for extension name "hello_world"
		// the path is: <root-dir>/bin/extensions/hello_world/<platform>/<channel>/hello_world.ext on macOS/Linux
		// and <root-dir>/bin/extensions/hello_world/<platform>/<channel>/hello_world.ext.exe on Windows.
		path := filepath.Join(rootDir, "bin", "extensions", extensionName, platform, channel, filename)

		if err := r.updateRunner.updater.UpdateMetadata(); err != nil {
			// Consider this a non-fatal error since it will be common to be offline
			// or otherwise unable to retrieve the metadata.
			return false, fmt.Errorf("update metadata: %w", err)
		}

		if err := r.updateRunner.StoreLocalHash(targetName); err != nil {
			// we do not want orbit to restart
			return false, fmt.Errorf("unable to lookup metadata for target: %s, %w", targetName, err)
		}

		sb.WriteString(path + "\n")
	}
	if err := os.WriteFile(extensionAutoLoadFile, []byte(sb.String()), constant.DefaultFileMode); err != nil {
		return false, fmt.Errorf("error writing extensions autoload file: %w", err)
	}

	// we do not want orbit to restart
	// runner.UpdateAction() will fetch the new targets and restart for us if needed
	return false, nil
}

// getFlagsFromJSON converts a json document of the form
// `{"number": 5, "string": "str", "boolean": true}` to a map[string]string.
//
// This only supports simple key:value pairs and not nested structures.
//
// Returns an empty map if flags is nil or an empty JSON `{}`.
func getFlagsFromJSON(flags json.RawMessage) (map[string]string, error) {
	var data map[string]interface{}
	err := json.Unmarshal([]byte(flags), &data)
	if err != nil {
		return nil, err
	}
	result := make(map[string]string)
	for k, v := range data {
		switch t := v.(type) {
		case string:
			result["--"+k] = t
		case bool:
			result["--"+k] = strconv.FormatBool(t)
		case float64:
			result["--"+k] = fmt.Sprintf("%.f", v)
		default:
			result["--"+k] = fmt.Sprintf("%v", v)
		}
	}
	return result, nil
}

// writeFlagFile writes the contents of the data map as a osquery flagfile to disk
// given a map[string]string, of the form: {"--foo":"bar","--value":"5"}
// it writes the contents of key=value, one line per pair to the file
// this only supports simple key:value pairs and not nested structures
func writeFlagFile(rootDir string, data map[string]string) error {
	flagfile := filepath.Join(rootDir, "osquery.flags")
	var sb strings.Builder
	for k, v := range data {
		if k != "" && v != "" {
			sb.WriteString(k + "=" + v + "\n")
		} else if v == "" {
			sb.WriteString(k + "\n")
		}
	}
	if err := os.WriteFile(flagfile, []byte(sb.String()), constant.DefaultFileMode); err != nil {
		return fmt.Errorf("writing flagfile %s failed: %w", flagfile, err)
	}
	return nil
}

// readFlagFile reads and parses the osquery.flags file on disk of the form
//
//	--foo="bar"
//	--bar=5
//	--zoo=true
//	--verbose
//
// and returns a map[string]string:
//
//	{"--foo": "bar", "--bar": 5, "--zoo", "--verbose": ""}
//
// This only supports simple key:value pairs and not nested structures.
//
// Returns:
//   - an error if the file does not exist.
//   - an empty map if the file is empty.
func readFlagFile(rootDir string) (map[string]string, error) {
	flagfile := filepath.Join(rootDir, "osquery.flags")
	bytes, err := os.ReadFile(flagfile)
	if err != nil {
		return nil, fmt.Errorf("reading flagfile %s failed: %w", flagfile, err)
	}
	content := strings.TrimSpace(string(bytes))
	result := make(map[string]string)
	if len(content) == 0 {
		return result, nil
	}
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line := strings.TrimSpace(line)
		// skip any empty lines
		if line == "" {
			continue
		}
		// skip line starting with "#" indicating that it's a comment
		if strings.HasPrefix(line, "#") {
			continue
		}
		// split each line by "="
		str := strings.Split(line, "=")
		if len(str) == 2 {
			result[str[0]] = str[1]
		}
		if len(str) == 1 {
			result[str[0]] = ""
		}
	}
	return result, nil
}
