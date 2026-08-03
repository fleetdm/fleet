// Package perfconfig persists saved osquery-perf run configurations to
// <config-dir>/perf-configs.json. Ported from src-tauri/src/perf_configs.rs.
//
// v1 stores everything (including enroll_secret and the SCEP challenge) as
// plain text — these are dev-only credentials for local fleet-perf
// simulation, the same security boundary as ~/.fleet/config and the rest
// of the Hangar settings file.
//
// All functions take an explicit directory so they're hermetically
// testable; the service layer resolves the real config dir via the paths
// package.
package perfconfig

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

const fileName = "perf-configs.json"

// Config is one saved osquery-perf run configuration.
type Config struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	ServerURL    string `json:"server_url"`
	EnrollSecret string `json:"enroll_secret"`
	// OSCounts is per-template host counts. A Go map marshals with
	// alphabetically-sorted keys, matching the Rust BTreeMap (diff-friendly,
	// stable rewrites).
	OSCounts         map[string]uint32 `json:"os_counts"`
	MDMEnabled       bool              `json:"mdm_enabled"`
	MDMProb          float64           `json:"mdm_prob"`
	MDMSCEPChallenge string            `json:"mdm_scep_challenge"`
	StartPeriod      string            `json:"start_period"`
	QueryInterval    string            `json:"query_interval"`
	ConfigInterval   string            `json:"config_interval"`
	// CreatedAtMS is server-stamped on first save and preserved across
	// updates. UpdatedAtMS is bumped on every save.
	CreatedAtMS uint64 `json:"created_at_ms"`
	UpdatedAtMS uint64 `json:"updated_at_ms"`
}

type configsFile struct {
	Configs []Config `json:"configs"`
}

func path(dir string) string { return filepath.Join(dir, fileName) }

// List returns all saved configs, or an empty slice if the file is absent.
func List(dir string) ([]Config, error) { return readAll(dir) }

func readAll(dir string) ([]Config, error) {
	b, err := os.ReadFile(path(dir))
	if errors.Is(err, fs.ErrNotExist) {
		return []Config{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", fileName, err)
	}
	var f configsFile
	if err := json.Unmarshal(b, &f); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", fileName, err)
	}
	if f.Configs == nil {
		f.Configs = []Config{}
	}
	return f.Configs, nil
}

func writeAll(dir string, configs []Config) error {
	if configs == nil {
		// A nil slice marshals to JSON null; the Rust app emits [] for an
		// empty list, so normalize to keep the file format identical.
		configs = []Config{}
	}
	b, err := json.MarshalIndent(configsFile{Configs: configs}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path(dir), b, 0o644)
}

// Save upserts cfg: if a config with the same ID exists it's overwritten in
// place (preserving the original CreatedAtMS); otherwise it's appended.
// nowMS is stamped into UpdatedAtMS (and CreatedAtMS for a brand-new
// record). Returns the saved record with timestamps applied.
func Save(dir string, cfg Config, nowMS uint64) (Config, error) {
	all, err := readAll(dir)
	if err != nil {
		return Config{}, err
	}
	for i := range all {
		if all[i].ID == cfg.ID {
			cfg.CreatedAtMS = all[i].CreatedAtMS
			cfg.UpdatedAtMS = nowMS
			all[i] = cfg
			if err := writeAll(dir, all); err != nil {
				return Config{}, err
			}
			return cfg, nil
		}
	}
	if cfg.CreatedAtMS == 0 {
		cfg.CreatedAtMS = nowMS
	}
	cfg.UpdatedAtMS = nowMS
	all = append(all, cfg)
	if err := writeAll(dir, all); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// Delete removes the config with the given ID (a no-op if absent).
func Delete(dir, id string) error {
	all, err := readAll(dir)
	if err != nil {
		return err
	}
	out := make([]Config, 0, len(all))
	for _, c := range all {
		if c.ID != id {
			out = append(out, c)
		}
	}
	return writeAll(dir, out)
}
