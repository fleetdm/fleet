package update

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"github.com/fleetdm/fleet/v4/server/service"
	"github.com/rs/zerolog/log"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"
)

type FlagRunner struct {
	orbitClient *service.Client
	opt         FlagUpdateOptions
	cancel      chan struct{}
}

type FlagUpdateOptions struct {
	CheckInterval time.Duration
	RootDir       string
	OrbitNodeKey  string
}

func NewFlagRunner(orbitClient *service.Client, opt FlagUpdateOptions) (*FlagRunner, error) {
	r := &FlagRunner{
		orbitClient: orbitClient,
		opt:         opt,
		cancel:      make(chan struct{}, 1),
	}
	return r, nil
}

func (r *FlagRunner) Execute() error {
	log.Debug().Msg("starting flag updater")

	ticker := time.NewTicker(r.opt.CheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-r.cancel:
			return nil
		case <-ticker.C:
			didUpdate, err := r.DoFlagsUpdate()
			if err != nil {
				log.Info().Err(err).Msg("flags updates failed " + err.Error())
			}
			if didUpdate {
				log.Info().Msg("flags updated, exiting")
				return nil
			}
		}
	}
}

func (r *FlagRunner) Interrupt(err error) {
	r.cancel <- struct{}{}
	log.Debug().Err(err).Msg("interrupt for flags updater")
}

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

	// next GetFlags from Fleet API
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
		log.Error().Msg("Error writing flags to disk " + err.Error())
		return false, fmt.Errorf("error writing flags to disk %w", err)
	}
	return true, nil
}

// getFlagsFromJSON converts the json of the type below
// {
//	  "number": 5,
//	  "string": "str",
//	  "boolean": true
//	}
// to a map[string]string
// this map will get compared and written to the filesystem and passed to osquery
// this only supports simple key:value pairs and not nested structures
func getFlagsFromJSON(flags json.RawMessage) (map[string]string, error) {
	result := make(map[string]string)

	var data map[string]interface{}
	err := json.Unmarshal([]byte(flags), &data)
	if err != nil {
		log.Info().Msg(err.Error())
		return nil, err
	}

	for k, v := range data {
		switch t := v.(type) {
		case string:
			result["--"+k] = t
		case bool:
			result["--"+k] = strconv.FormatBool(t)
		case float64:
			result["--"+k] = fmt.Sprint(t)
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
	if err := ioutil.WriteFile(flagfile, []byte(sb.String()), constant.DefaultFileMode); err != nil {
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
	bytes, err := ioutil.ReadFile(flagfile)
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
