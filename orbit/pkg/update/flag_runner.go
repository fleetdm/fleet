package update

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"github.com/fleetdm/fleet/v4/server/service"
	"github.com/rs/zerolog/log"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"
)

// FlagRunner is a specialized runner to periodically check and update flags from Fleet
// It is designed with Execute and Interrupt functions to be compatible with oklog/run
//
// It uses an OrbitClient, along with FlagUpdateOptions to connect to Fleet
type FlagRunner struct {
	orbitClient *service.OrbitClient
	opt         FlagUpdateOptions
	cancel      chan struct{}
}

// FlagUpdateOptions is options provided for the flag update runner
type FlagUpdateOptions struct {
	// CheckInterval is the interval to check for updates
	CheckInterval time.Duration
	// RootDir is the root directory for orbit state
	RootDir string
	// OrbitNodeKey is the orbit node key for the enrolled host
	OrbitNodeKey string
}

// NewFlagRunner creates a new runner with provided options
// The runner must be started with Execute
func NewFlagRunner(orbitClient *service.OrbitClient, opt FlagUpdateOptions) (*FlagRunner, error) {
	r := &FlagRunner{
		orbitClient: orbitClient,
		opt:         opt,
		cancel:      make(chan struct{}, 1),
	}
	return r, nil
}

// Execute starts the loop checking for updates
func (r *FlagRunner) Execute() error {
	log.Debug().Msg("starting flag updater")

	ticker := time.NewTicker(r.opt.CheckInterval)
	defer ticker.Stop()

	// Run until cancel or returning an error
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
	flagsJSON, err := r.orbitClient.GetConfig(r.opt.OrbitNodeKey)
	if err != nil {
		return false, fmt.Errorf("error getting flags from fleet %w", err)
	}

	osqueryFlagMapFromFleet, err := getFlagsFromJSON(flagsJSON)
	if err != nil {
		return false, fmt.Errorf("error parsing flags %w", err)
	}

	// compare both flags, if they are equal, nothing to do
	if flagFileExists && reflect.DeepEqual(osqueryFlagMapFromFile, osqueryFlagMapFromFleet) {
		return false, nil
	}

	// flags are not equal, write the fleet flags to disk
	err = writeFlagFile(r.opt.RootDir, osqueryFlagMapFromFleet)
	if err != nil {
		return false, fmt.Errorf("error writing flags to disk %w", err)
	}
	return true, nil
}

// getFlagsFromJSON converts the json of the type below
// {"number": 5, "string": "str", "boolean": true}
// to a map[string]string
// this map will get compared and written to the filesystem and passed to osquery
// this only supports simple key:value pairs and not nested structures
func getFlagsFromJSON(flags json.RawMessage) (map[string]string, error) {
	result := make(map[string]string)

	var data map[string]interface{}
	err := json.Unmarshal([]byte(flags), &data)
	if err != nil {
		return nil, err
	}

	for k, v := range data {
		result["--"+k] = fmt.Sprintf("%v", v)
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

// readFlagFile reads and parses the osquery.flags file on disk
// and returns a map[string]string, of the form:
// {"--foo":"bar","--value":"5"}
// this only supports simple key:value pairs and not nested structures
func readFlagFile(rootDir string) (map[string]string, error) {
	flagfile := filepath.Join(rootDir, "osquery.flags")
	bytes, err := os.ReadFile(flagfile)
	if err != nil {
		return nil, fmt.Errorf("reading flagfile %s failed: %w", flagfile, err)
	}
	result := make(map[string]string)
	lines := strings.Split(strings.TrimSpace(string(bytes)), "\n")
	for _, line := range lines {
		// skip line starting with "#" indicating that it's a comment
		if !strings.HasPrefix(line, "#") {
			// split each line by "="
			str := strings.Split(strings.TrimSpace(line), "=")
			if len(str) == 2 {
				result[str[0]] = str[1]
			}
			if len(str) == 1 {
				result[str[0]] = ""
			}
		}
	}
	return result, nil
}
