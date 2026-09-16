package tuf

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fleetdm/fleet/tools/hangar/internal/processes"
	"github.com/fleetdm/fleet/tools/hangar/internal/settings"
)

func envMap(pairs []processes.EnvPair) map[string]string {
	m := make(map[string]string, len(pairs))
	for _, p := range pairs {
		m[p.Key] = p.Value
	}
	return m
}

func TestEnvForFullSelection(t *testing.T) {
	cfg := settings.TufConfig{
		Platforms:    []string{"macos", "windows", "windows-arm64", "linux", "linux-arm64"},
		FleetURL:     "https://andrey.ngrok.app",
		TufURL:       "https://tuf.andrey.ngrok.app",
		EnrollSecret: "sekret",
		FleetDesktop: true,
		Debug:        true,
	}
	m := envMap(EnvFor(cfg))

	if m["SYSTEMS"] != "macos windows windows-arm64 linux linux-arm64" {
		t.Errorf("SYSTEMS = %q", m["SYSTEMS"])
	}
	for _, g := range []string{
		"GENERATE_PKG", "GENERATE_MSI", "GENERATE_MSI_ARM64",
		"GENERATE_DEB", "GENERATE_RPM", "GENERATE_DEB_ARM64", "GENERATE_RPM_ARM64",
	} {
		if m[g] != "1" {
			t.Errorf("expected %s=1, got %q", g, m[g])
		}
	}
	// URLs set for every package-type prefix.
	for _, tp := range []string{"PKG", "DEB", "RPM", "MSI", "PKG_TAR_ZST"} {
		if m[tp+"_FLEET_URL"] != cfg.FleetURL {
			t.Errorf("%s_FLEET_URL = %q, want %q", tp, m[tp+"_FLEET_URL"], cfg.FleetURL)
		}
		if m[tp+"_TUF_URL"] != cfg.TufURL {
			t.Errorf("%s_TUF_URL = %q, want %q", tp, m[tp+"_TUF_URL"], cfg.TufURL)
		}
	}
	if m["ENROLL_SECRET"] != "sekret" {
		t.Errorf("ENROLL_SECRET = %q", m["ENROLL_SECRET"])
	}
	if m["FLEET_DESKTOP"] != "1" || m["DEBUG"] != "1" {
		t.Errorf("expected FLEET_DESKTOP=1 DEBUG=1, got %q %q", m["FLEET_DESKTOP"], m["DEBUG"])
	}
	// Hangar runs the file-server itself, so the build must skip main.sh's.
	if m["SKIP_SERVER"] != "1" {
		t.Errorf("expected SKIP_SERVER=1, got %q", m["SKIP_SERVER"])
	}
}

func TestFileServerArgs(t *testing.T) {
	got := FileServerArgs()
	want := []string{"run", "./tools/file-server", "8081", "test_tuf/repository"}
	if len(got) != len(want) {
		t.Fatalf("FileServerArgs = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("FileServerArgs = %v, want %v", got, want)
		}
	}
}

func TestEnvForSubsetAndToggles(t *testing.T) {
	cfg := settings.TufConfig{
		Platforms: []string{"macos", "bogus"}, // unknown key ignored
	}
	m := envMap(EnvFor(cfg))
	if m["SYSTEMS"] != "macos" {
		t.Errorf("SYSTEMS = %q, want macos (bogus ignored)", m["SYSTEMS"])
	}
	if m["GENERATE_PKG"] != "1" {
		t.Error("expected GENERATE_PKG=1")
	}
	if _, ok := m["GENERATE_MSI"]; ok {
		t.Error("GENERATE_MSI should be absent for macos-only")
	}
	// Toggles off → keys absent.
	if _, ok := m["FLEET_DESKTOP"]; ok {
		t.Error("FLEET_DESKTOP should be absent when disabled")
	}
	if _, ok := m["DEBUG"]; ok {
		t.Error("DEBUG should be absent when disabled")
	}
}

func TestDeleteAssets(t *testing.T) {
	repo := t.TempDir()
	assets := filepath.Join(repo, AssetsDir)
	if err := os.MkdirAll(filepath.Join(assets, "repository"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := DeleteAssets(repo); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(assets); !os.IsNotExist(err) {
		t.Errorf("expected %s removed, stat err = %v", assets, err)
	}
	// Idempotent: deleting again is a no-op.
	if err := DeleteAssets(repo); err != nil {
		t.Errorf("second delete should be a no-op, got %v", err)
	}
	// Empty repo path is rejected.
	if err := DeleteAssets(""); err == nil {
		t.Error("expected error for empty repo path")
	}
}

func TestMainScriptPath(t *testing.T) {
	if got := MainScriptPath("/x/fleet"); got != "/x/fleet/tools/tuf/test/main.sh" {
		t.Errorf("MainScriptPath = %q", got)
	}
}
