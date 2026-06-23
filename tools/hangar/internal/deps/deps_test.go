package deps

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtractVersion(t *testing.T) {
	cases := []struct{ in, want string }{
		{"git version 2.39.5", "2.39.5"},
		{"go version go1.26.3 darwin/arm64", "1.26.3"},
		{"v24.10.0", "24.10.0"},
		{"Homebrew 4.2.1", "4.2.1"},
		{"Docker version 24.0.6, build ed223bc", "24.0.6"},
		{"1.5", "1.5.0"}, // padded
		{"no version here", ""},
		{"", ""},
	}
	for _, c := range cases {
		if got := extractVersion(c.in); got != c.want {
			t.Errorf("extractVersion(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestSatisfies(t *testing.T) {
	if v := satisfies("24.16.0", "^24.10.0"); v == nil || !*v {
		t.Errorf("24.16.0 ^24.10.0 = %v, want true", v)
	}
	if v := satisfies("23.0.0", "^24.10.0"); v == nil || *v {
		t.Errorf("23.0.0 ^24.10.0 = %v, want false", v)
	}
	if v := satisfies("garbage", "^24.10.0"); v != nil {
		t.Errorf("unparseable version should be nil, got %v", *v)
	}
	if v := satisfies("24.0.0", "@@@bad"); v != nil {
		t.Errorf("unparseable constraint should be nil, got %v", *v)
	}
}

func TestReadEngines(t *testing.T) {
	dir := t.TempDir()
	pkg := filepath.Join(dir, "package.json")

	os.WriteFile(pkg, []byte(`{"engines":{"node":"^24.10.0","npm":">=10"}}`), 0o644)
	if v, ok := readEngines(pkg, "node"); !ok || v != "^24.10.0" {
		t.Errorf("readEngines node = %q,%v", v, ok)
	}

	os.WriteFile(pkg, []byte(`{"name":"x"}`), 0o644)
	if _, ok := readEngines(pkg, "node"); ok {
		t.Error("missing engines should report not-found")
	}

	if _, ok := readEngines(filepath.Join(dir, "nope.json"), "node"); ok {
		t.Error("missing file should report not-found")
	}
}

func TestRequiredNodeVersion(t *testing.T) {
	// No repo → fallback.
	if got := requiredNodeVersion(""); got != "^24.10.0" {
		t.Errorf("empty repo = %q, want fallback", got)
	}

	// Repo with engines.node → that value.
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"engines":{"node":"^99.0.0"}}`), 0o644)
	if got := requiredNodeVersion(dir); got != "^99.0.0" {
		t.Errorf("repo engines = %q, want ^99.0.0", got)
	}

	// Repo without engines → fallback.
	bare := t.TempDir()
	os.WriteFile(filepath.Join(bare, "package.json"), []byte(`{}`), 0o644)
	if got := requiredNodeVersion(bare); got != "^24.10.0" {
		t.Errorf("repo without engines = %q, want fallback", got)
	}
}

// Smoke test: the real checklist runs and includes the expected rows. git
// and go are present on any dev machine that built this.
func TestCheckDependenciesSmoke(t *testing.T) {
	report := CheckDependencies("", false)
	ids := map[string]DepCheck{}
	for _, c := range report.Checks {
		ids[c.ID] = c
	}
	for _, want := range []string{"xcode-clt", "brew", "git", "go", "node-version-manager", "node", "yarn", "docker"} {
		if _, ok := ids[want]; !ok {
			t.Errorf("checklist missing %q", want)
		}
	}
	if c, ok := ids["git"]; ok && !c.Installed {
		t.Error("git should be detected as installed in the test environment")
	}
	if c, ok := ids["node"]; ok {
		if c.Required == nil || *c.Required == "" {
			t.Error("node check should carry a required version")
		}
	}
}
