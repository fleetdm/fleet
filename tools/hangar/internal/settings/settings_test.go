package settings

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefault(t *testing.T) {
	d := Default()
	if !d.FleetServe.Premium || !d.FleetServe.Debug || !d.FleetServe.Logging {
		t.Errorf("serve flags should default true: %+v", d.FleetServe)
	}
	if d.PythonServer.Port != 8000 {
		t.Errorf("python port = %d, want 8000", d.PythonServer.Port)
	}
	if d.Theme != ThemeSystem {
		t.Errorf("theme = %q, want system", d.Theme)
	}
	if d.FirstRunComplete {
		t.Error("first_run_complete should default false")
	}
}

func TestLoadMissingFileIsDefault(t *testing.T) {
	got, err := Load(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if got.PythonServer.Port != 8000 || !got.FleetServe.Premium {
		t.Errorf("missing file should yield defaults, got %+v", got)
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	rp := "/Users/tester/fleet"
	in := Default()
	in.RepoPath = &rp
	in.FirstRunComplete = true
	in.Theme = ThemeDark
	in.FleetServe.Premium = false
	in.FleetServe.Env = []EnvVar{{Key: "FOO", Value: "bar", Enabled: false}}

	if err := Save(dir, in); err != nil {
		t.Fatal(err)
	}
	got, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got.RepoPath == nil || *got.RepoPath != rp {
		t.Errorf("repo_path round-trip failed: %v", got.RepoPath)
	}
	if got.Theme != ThemeDark || got.FleetServe.Premium != false {
		t.Errorf("round-trip mismatch: theme=%q premium=%v", got.Theme, got.FleetServe.Premium)
	}
	if len(got.FleetServe.Env) != 1 || got.FleetServe.Env[0].Enabled != false {
		t.Errorf("env round-trip failed: %+v", got.FleetServe.Env)
	}
}

// An existing file missing the serve flags must load them as true (serde
// default parity), and an EnvVar row missing "enabled" must default true.
func TestLoadDefaultsForMissingFields(t *testing.T) {
	dir := t.TempDir()
	partial := `{
  "repo_path": "/x",
  "fleet_serve": { "config_path": null, "env": [ { "key": "A", "value": "1" } ] }
}`
	if err := os.WriteFile(filepath.Join(dir, fileName), []byte(partial), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !got.FleetServe.Premium || !got.FleetServe.Debug || !got.FleetServe.Logging {
		t.Errorf("missing serve flags should default true, got %+v", got.FleetServe)
	}
	if got.PythonServer.Port != 8000 {
		t.Errorf("missing python_server should default port 8000, got %d", got.PythonServer.Port)
	}
	if got.Theme != ThemeSystem {
		t.Errorf("missing theme should default system, got %q", got.Theme)
	}
	if len(got.FleetServe.Env) != 1 || !got.FleetServe.Env[0].Enabled {
		t.Errorf("env row missing 'enabled' should default true, got %+v", got.FleetServe.Env)
	}
}

// An explicit false in the file must win over the true default.
func TestLoadExplicitFalseWins(t *testing.T) {
	dir := t.TempDir()
	data := `{ "fleet_serve": { "premium": false, "debug": false, "logging_debug": false } }`
	if err := os.WriteFile(filepath.Join(dir, fileName), []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got.FleetServe.Premium || got.FleetServe.Debug || got.FleetServe.Logging {
		t.Errorf("explicit false should win, got %+v", got.FleetServe)
	}
}

func TestSavedJSONKeys(t *testing.T) {
	dir := t.TempDir()
	if err := Save(dir, Default()); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, fileName))
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{
		"\"repo_path\"", "\"fleetctl_path\"", "\"gitops_dir\"", "\"first_run_complete\"",
		"\"python_server\"", "\"fleet_serve\"", "\"logging_debug\"", "\"favorite_crons\"",
	} {
		if !strings.Contains(string(raw), key) {
			t.Errorf("saved settings missing key %s", key)
		}
	}
}
