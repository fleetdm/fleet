package update

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"github.com/fleetdm/fleet/v4/server/service"
	"github.com/rs/zerolog/log"
)

// OrbitConfigFetcher allows fetching Orbit configuration.
type OrbitConfigFetcher interface {
	// GetConfig returns the Orbit configuration.
	GetConfig() (*service.OrbitConfig, error)
}

// FlagRunner is a specialized runner to periodically check and update flags from Fleet
// It is designed with Execute and Interrupt functions to be compatible with oklog/run
//
// It uses an OrbitClient, along with FlagUpdateOptions to connect to Fleet
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
			log.Info().Msg("calling flags update")
			didUpdate, err := r.DoFlagsUpdate()
			if err != nil {
				log.Info().Err(err).Msg("flags updates failed")
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
// It uses an OrbitClient, along with ExtensionUpdateOptions and updateRunner to connect to Fleet
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
			log.Info().Msg("calling /config API to fetch/update extensions")
			didExtensionsUpdate, err := r.DoExtensionConfigUpdate()
			if err != nil {
				log.Info().Err(err).Msg("ext update failed")
			}
			if didExtensionsUpdate {
				log.Info().Msg("successfully updated/fetched extensions from /config API")
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
func (r *ExtensionRunner) DoExtensionConfigUpdate() (bool, error) {
	// call "/config" API endpoint to grab orbit configs from Fleet
	config, err := r.configFetcher.GetConfig()
	if err != nil {
		return false, fmt.Errorf("extensionsUpdate: error getting extensions config from fleet: %w", err)
	}

	extensionAutoLoadFile := filepath.Join(r.opt.RootDir, "extensions.load")
	if len(config.Extensions) == 0 {
		// Extensions from Fleet is empty
		// this can be either because of:
		// 1. the default state, where no extensions are configured to begin with, or
		// 2. extensions were previously were configured, but now are deleted and reverted to empty state

		// Handle case 1, where our autoload file does not exist, so there is nothing to update and no error
		stat, err := os.Stat(extensionAutoLoadFile)
		if errors.Is(err, os.ErrNotExist) {
			log.Debug().Msg(extensionAutoLoadFile + " not found, nothing to update")
			return false, nil
		}

		if stat.Size() > 0 {
			// handle case 2: create/truncate the extensions.load file and let the runner interrupt, so that
			// osquery can't startup without the extensions that were previously loaded
			_, err := os.Create(extensionAutoLoadFile)
			if err != nil {
				return false, fmt.Errorf("extensionsUpdate: error creating file %s, %w", extensionAutoLoadFile, err)
			}
			// we want to return true here, and restart with the empty extensions.load file
			// so that we "unload" the previously loaded extensions
			return true, nil
		}
		return false, nil
	}

	type ExtensionInfo struct {
		Platform string `json:"platform"`
		Channel  string `json:"channel"`
		FileName string `json:"file_name"`
	}

	var data map[string]ExtensionInfo
	err = json.Unmarshal(config.Extensions, &data)
	if err != nil {
		return false, fmt.Errorf("error unmarshing json extensions config from fleet: %w", err)
	}

	var sb strings.Builder
	for k, v := range data {
		// we don't want path traversal and the like in the filename
		if strings.Contains(v.FileName, "..") || strings.Contains(v.FileName, "/") {
			log.Info().Msgf("invalid characters found in filename (%s) for extension (%s): skipping", v.FileName, k)
			continue
		}

		// add "extensions/" as a prefix to the target name, since that's the namespace we expect for extensions on TUF
		target := "extensions/" + k

		// update our view of targets
		r.updateRunner.UpdateRunnerOptTargets(target)
		r.updateRunner.updater.SetExtentionsTargetInfo(target, v.Platform, v.Channel, v.FileName)

		path := filepath.Join(r.updateRunner.updater.opt.RootDirectory, "bin", "extensions", k, v.Platform, v.Channel, v.FileName)

		meta, err := r.updateRunner.updater.Lookup(target)
		if err != nil {
			return false, fmt.Errorf("unable to lookup metadata for target: %s, %w", target, err)
		}

		_, localHash, err := fileHashes(meta, path)
		if err != nil {
			// OK, not an error, expected on initial that path doesn't exist
			return false, nil
		}

		// update local hashes
		log.Info().Msgf("updating local hash(%s)=%x", target, localHash)
		r.updateRunner.localHashes[target] = localHash

		sb.WriteString(path + "\n")
	}
	if err := os.WriteFile(extensionAutoLoadFile, []byte(sb.String()), constant.DefaultFileMode); err != nil {
		return false, fmt.Errorf("error writing extensions autoload file: %w", err)
	}

	// we don't want to return true, because we don't want to restart
	// UpdateAction() will fetch the new targets and restart for us if needed
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
