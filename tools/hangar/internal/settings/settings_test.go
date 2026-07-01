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

// A fresh install (missing file) must come back with exactly one enabled
// server on the canonical ports, active.
func TestLoadMissingFileSynthesizesServer(t *testing.T) {
	got, err := Load(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Servers) != 1 {
		t.Fatalf("want 1 server, got %d", len(got.Servers))
	}
	s := got.Servers[0]
	if s.ID != "s1" || !s.Enabled || s.WorktreePath != nil {
		t.Errorf("default server unexpected: %+v", s)
	}
	if s.Ports != (ServerPorts{Server: 8080, MySQL: 3306, Redis: 6379, S3: 9000, S3Console: 9001}) {
		t.Errorf("server 1 should use canonical ports, got %+v", s.Ports)
	}
	if s.ComposeProject != "fleet" {
		t.Errorf("server 1 compose project = %q, want fleet", s.ComposeProject)
	}
	if got.ActiveServerID != "s1" {
		t.Errorf("active server = %q, want s1", got.ActiveServerID)
	}
}

// A legacy single-server file (repo_path + fleet_serve, no servers array) must
// migrate into one server: repo_path becomes the worktree, and the saved serve
// config carries over.
func TestLoadMigratesLegacySingleServer(t *testing.T) {
	dir := t.TempDir()
	legacy := `{
  "repo_path": "/Users/tester/fleet",
  "first_run_complete": true,
  "fleet_serve": { "premium": false, "debug": true, "logging_debug": true, "env": [ { "key": "A", "value": "1" } ] }
}`
	if err := os.WriteFile(filepath.Join(dir, fileName), []byte(legacy), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Servers) != 1 {
		t.Fatalf("want 1 migrated server, got %d", len(got.Servers))
	}
	s := got.Servers[0]
	if s.WorktreePath == nil || *s.WorktreePath != "/Users/tester/fleet" {
		t.Errorf("legacy repo_path should become worktree, got %v", s.WorktreePath)
	}
	if s.FleetServe.Premium != false {
		t.Errorf("legacy serve premium=false should carry over, got %+v", s.FleetServe)
	}
	if len(s.FleetServe.Env) != 1 || s.FleetServe.Env[0].Key != "A" {
		t.Errorf("legacy serve env should carry over, got %+v", s.FleetServe.Env)
	}
	if got.ActiveServerID != "s1" {
		t.Errorf("active server = %q, want s1", got.ActiveServerID)
	}
}

// A file already in multi-server shape must round-trip untouched (migration is
// idempotent) and must NOT be clobbered by the legacy fields.
func TestLoadMultiServerRoundTrip(t *testing.T) {
	dir := t.TempDir()
	wp1, wp2 := "/a/fleet", "/a/fleet-n1"
	in := Default()
	migrate(&in) // shape it like a loaded config
	in.Servers = []ServerProfile{
		{ID: "s1", Name: "main", Color: "green", WorktreePath: &wp1, Ports: DefaultPortsForIndex(0), ComposeProject: "fleet", FleetServe: FleetServeConfig{Premium: true, Env: []EnvVar{}}, Enabled: true},
		{ID: "s2", Name: "n-1", Color: "purple", WorktreePath: &wp2, Ports: DefaultPortsForIndex(1), ComposeProject: "fleet-s2", FleetServe: FleetServeConfig{Premium: false, Env: []EnvVar{}}, Enabled: true},
	}
	in.ActiveServerID = "s2"
	if err := Save(dir, in); err != nil {
		t.Fatal(err)
	}
	got, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Servers) != 2 {
		t.Fatalf("want 2 servers, got %d", len(got.Servers))
	}
	if got.ActiveServerID != "s2" {
		t.Errorf("active server = %q, want s2", got.ActiveServerID)
	}
	if got.Servers[1].Ports.MySQL != 3326 || got.Servers[1].ComposeProject != "fleet-s2" {
		t.Errorf("server 2 ports/project wrong: %+v", got.Servers[1])
	}
}

// A dangling active_server_id (points at a removed server) is repaired to the
// first server.
func TestLoadRepairsDanglingActiveServer(t *testing.T) {
	dir := t.TempDir()
	data := `{ "servers": [ { "id": "s1", "name": "main", "enabled": true } ], "active_server_id": "s9" }`
	if err := os.WriteFile(filepath.Join(dir, fileName), []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got.ActiveServerID != "s1" {
		t.Errorf("dangling active server should fall back to s1, got %q", got.ActiveServerID)
	}
	// A server with only id/name/enabled set must get ports/project/color backfilled.
	if got.Servers[0].Ports.MySQL != 3306 || got.Servers[0].ComposeProject != "fleet" || got.Servers[0].Color == "" {
		t.Errorf("partial server not backfilled: %+v", got.Servers[0])
	}
}

func TestDefaultPortsForIndex(t *testing.T) {
	cases := []struct {
		i                                   int
		server, mysql, redis, s3, s3console uint16
	}{
		{0, 8080, 3306, 6379, 9000, 9001},
		{1, 8090, 3326, 6389, 9020, 9011},
		{2, 8100, 3346, 6399, 9040, 9021},
	}
	for _, c := range cases {
		got := DefaultPortsForIndex(c.i)
		want := ServerPorts{Server: c.server, MySQL: c.mysql, Redis: c.redis, S3: c.s3, S3Console: c.s3console}
		if got != want {
			t.Errorf("ports[%d] = %+v, want %+v", c.i, got, want)
		}
	}
}

func TestNextServerProfile(t *testing.T) {
	// From one server we get s2 on the offset block.
	one := []ServerProfile{defaultServer(0)}
	p, ok := NextServerProfile(one)
	if !ok {
		t.Fatal("expected a profile for slot 2")
	}
	if p.ID != "s2" || p.Ports.MySQL != 3326 || p.ComposeProject != "fleet-s2" {
		t.Errorf("next profile wrong: %+v", p)
	}
	// At the cap we get false.
	full := []ServerProfile{defaultServer(0), defaultServer(1), defaultServer(2)}
	if _, ok := NextServerProfile(full); ok {
		t.Errorf("expected no profile past the cap of %d", MaxServers)
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
