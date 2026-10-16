package installers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/tools/hangar/internal/fleetctl"
	"github.com/fleetdm/fleet/tools/hangar/internal/settings"
)

func TestSlug(t *testing.T) {
	cases := map[string]string{
		"rc":             "rc",
		"Server 2 (n-1)": "server-2-n-1",
		"💻 Workstations": "workstations",
		"  --Main--  ":   "main",
		"📱🔐":             "",
		"Prev v4.92.3":   "prev-v4-92-3",
	}
	for in, want := range cases {
		if got := Slug(in); got != want {
			t.Errorf("Slug(%q) = %q, want %q", in, got, want)
		}
	}
	if got := ServerSlug(settings.ServerProfile{ID: "s2", Name: "🚀"}); got != "s2" {
		t.Errorf("ServerSlug fallback = %q, want s2", got)
	}
}

func TestFileNames(t *testing.T) {
	def := settings.DefaultInstallerServer()
	var got []string
	for _, j := range Jobs("rc", "/out/rc", def) {
		got = append(got, filepath.Base(j.OutFile))
	}
	want := []string{
		"fleetd-rc-macos-desktop.pkg",
		"fleetd-rc-windows-amd64-desktop.msi",
		"fleetd-rc-windows-arm64-desktop.msi",
		"fleetd-rc-linux-amd64-desktop.deb",
		"fleetd-rc-linux-arm64-desktop.deb",
		"fleetd-rc-linux-amd64-desktop.rpm",
		"fleetd-rc-linux-arm64-desktop.rpm",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("default names\n got %v\nwant %v", got, want)
	}

	deb, _ := PlatformByKey("deb-arm64")
	pkg, _ := PlatformByKey("macos")
	cfg := def
	cfg.FleetDesktop = false
	if n := FileName("prev", deb, cfg); n != "fleetd-prev-linux-arm64.deb" {
		t.Errorf("no desktop: %q", n)
	}
	cfg = def
	cfg.UpdateSource = settings.UpdateSourceLocalTUF
	cfg.OrbitChannel = "edge"
	if n := FileName("rc", pkg, cfg); n != "fleetd-rc-macos-tuf-edge-desktop.pkg" {
		t.Errorf("tuf+edge: %q", n)
	}
	cfg = def
	cfg.OrbitChannel, cfg.DesktopChannel, cfg.OsquerydChannel = "edge", "edge", "edge"
	if n := FileName("rc", pkg, cfg); n != "fleetd-rc-macos-edge-desktop.pkg" {
		t.Errorf("all edge: %q", n)
	}
	cfg = def
	cfg.OrbitChannel, cfg.OsquerydChannel = "1.61.0", "edge"
	if n := FileName("rc", pkg, cfg); n != "fleetd-rc-macos-orbit-1.61.0-osquery-edge-desktop.pkg" {
		t.Errorf("mixed channels: %q", n)
	}
}

func TestJobsSubsetAndEncode(t *testing.T) {
	cfg := settings.DefaultInstallerServer()
	cfg.Platforms = []string{"rpm-arm64", "bogus", "macos"} // order follows Platforms
	jobs := Jobs("rc", "/o", cfg)
	if len(jobs) != 2 || jobs[0].Key != "macos" || jobs[1].Key != "rpm-arm64" {
		t.Fatalf("jobs = %+v", jobs)
	}
	enc := EncodeJobs(jobs)
	wantEnc := "macos\tpkg\t-\t/o/fleetd-rc-macos-desktop.pkg\nrpm-arm64\trpm\tarm64\t/o/fleetd-rc-linux-arm64-desktop.rpm\n"
	if enc != wantEnc {
		t.Errorf("EncodeJobs = %q", enc)
	}
}

func TestCommonArgs(t *testing.T) {
	in := BuildInputs{FleetURL: "https://fleet.example", EnrollSecret: "s3cret", UpdateURL: "https://tuf.example", UpdateRoots: `[{"k":1}]`}
	def := settings.DefaultInstallerServer()
	got := CommonArgs(def, in)
	want := []string{"--fleet-url=https://fleet.example", "--enroll-secret=s3cret", "--disable-open-folder", "--fleet-desktop"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("default args %v", got)
	}

	cfg := def
	cfg.FleetDesktop = false
	cfg.Debug = true
	cfg.DesktopChannel = "edge"
	cfg.UpdateSource = settings.UpdateSourceLocalTUF
	got = CommonArgs(cfg, in)
	want = []string{
		"--fleet-url=https://fleet.example", "--enroll-secret=s3cret", "--disable-open-folder", "--debug",
		"--desktop-channel=edge", "--update-url=https://tuf.example", `--update-roots=[{"k":1}]`,
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("custom args\n got %v\nwant %v", got, want)
	}
}

func TestValidateConfig(t *testing.T) {
	cfg := settings.DefaultInstallerServer()
	if err := ValidateConfig(cfg); err != nil {
		t.Errorf("default config: %v", err)
	}
	cfg.FleetURL = "https://f\n--insecure"
	if err := ValidateConfig(cfg); err == nil {
		t.Error("newline in Fleet URL should be rejected")
	}
}

func TestMatchContext(t *testing.T) {
	ctxs := map[string]string{
		"default": "",
		"public":  "https://fleetqa.ngrok.io",
		"rc":      "https://localhost:8080",
		"prev":    "https://127.0.0.1:8090/",
	}
	order := []string{"default", "public", "rc", "prev"}
	if got := MatchContext(ctxs, order, 8080); got != "rc" {
		t.Errorf("8080 → %q", got)
	}
	if got := MatchContext(ctxs, order, 8090); got != "prev" {
		t.Errorf("8090 → %q", got)
	}
	if got := MatchContext(ctxs, order, 8100); got != "" {
		t.Errorf("8100 → %q", got)
	}
}

func TestTUFChannels(t *testing.T) {
	targets := `{"signed":{"targets":{
		"orbit/macos/stable/orbit":{},"orbit/macos/1.61.0/orbit":{},"orbit/linux/26.10/orbit":{},"orbit/linux/1.60.0/orbit":{},
		"desktop/macos/stable/desktop.app.tar.gz":{},
		"osqueryd/macos-app/5.24.0/osqueryd.app.tar.gz":{},"osqueryd/macos-app/5/osqueryd.app.tar.gz":{},"osqueryd/linux/stable/osqueryd":{},
		"nudge/macos/stable/nudge.app.tar.gz":{}, "junk":{}
	}}}`
	got, err := TUFChannels([]byte(targets))
	if err != nil {
		t.Fatal(err)
	}
	want := Channels{
		Orbit:    []string{"stable", "1.60.0", "1.61.0", "26.10"},
		Desktop:  []string{"stable"},
		Osqueryd: []string{"stable", "5", "5.24.0"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("TUFChannels\n got %+v\nwant %+v", got, want)
	}
}

func TestUpdateManifest(t *testing.T) {
	dir := t.TempDir()
	since := time.Now().Add(-time.Second)
	old := filepath.Join(dir, "fleetd-rc-macos.pkg")
	if err := os.WriteFile(old, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	// An existing entry for a file that's gone is pruned; one for a file that
	// still exists (and wasn't rebuilt) is kept.
	seed := Manifest{Files: map[string]ManifestEntry{
		"fleetd-rc-macos.pkg": {Platform: "macos", Team: "old"},
		"gone.deb":            {Platform: "deb-amd64"},
	}}
	b, _ := json.Marshal(seed)
	if err := os.WriteFile(filepath.Join(dir, ManifestName), b, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(old, since.Add(-time.Hour), since.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}

	built := filepath.Join(dir, "fleetd-rc-linux-arm64-desktop.deb")
	if err := os.WriteFile(built, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	jobs := []Job{
		{Key: "deb-arm64", OutFile: built},
		{Key: "rpm-arm64", OutFile: filepath.Join(dir, "fleetd-rc-linux-arm64-desktop.rpm")}, // failed: no file
		{Key: "macos", OutFile: old},                                                         // stale: not rebuilt
	}
	n, err := UpdateManifest(dir, jobs, since, ManifestEntry{Team: "Workstations", FleetURL: "https://f"})
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("built = %d, want 1", n)
	}
	var m Manifest
	b, _ = os.ReadFile(filepath.Join(dir, ManifestName))
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	if len(m.Files) != 2 {
		t.Fatalf("manifest files = %v", m.Files)
	}
	if e := m.Files["fleetd-rc-linux-arm64-desktop.deb"]; e.Platform != "deb-arm64" || e.Team != "Workstations" {
		t.Errorf("new entry = %+v", e)
	}
	if e := m.Files["fleetd-rc-macos.pkg"]; e.Team != "old" {
		t.Errorf("kept entry = %+v", e)
	}
}

func TestListOutputs(t *testing.T) {
	dir := t.TempDir()
	for _, n := range []string{"a.pkg", "b.deb", "manifest.json", "notes.txt"} {
		if err := os.WriteFile(filepath.Join(dir, n), []byte("xx"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, ManifestName), []byte(`{"files":{"a.pkg":{"platform":"macos","team":"Workstations"}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := ListOutputs(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("ListOutputs = %+v", got)
	}
	for _, f := range got {
		switch f.Name {
		case "a.pkg":
			if f.Build == nil || f.Build.Team != "Workstations" {
				t.Errorf("a.pkg build = %+v", f.Build)
			}
		case "b.deb":
			if f.Build != nil {
				t.Errorf("b.deb has no manifest entry, got %+v", f.Build)
			}
		}
	}
	if missing, err := ListOutputs(filepath.Join(dir, "nope")); err != nil || len(missing) != 0 {
		t.Errorf("missing dir: %v %v", missing, err)
	}
}

// fakeFleet serves the endpoints Scan reads. newRoutes controls whether the
// server knows /fleets (current) or only /teams (older releases).
func fakeFleet(t *testing.T, tier string, newRoutes bool) *httptest.Server {
	t.Helper()
	return httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer tok" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		switch r.URL.Path {
		case "/api/v1/fleet/config":
			_, _ = w.Write([]byte(`{"server_settings":{"server_url":"https://fleet.public"},"license":{"tier":"` + tier + `"}}`))
		case "/api/v1/fleet/spec/enroll_secret":
			_, _ = w.Write([]byte(`{"spec":{"secrets":[{"secret":"global"}]}}`))
		case "/api/v1/fleet/fleets":
			if !newRoutes {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			_, _ = w.Write([]byte(`{"fleets":[{"id":3,"name":"💻 Workstations","secrets":[{"secret":"ws"}]},{"id":4,"name":"Empty","secrets":[]},{"id":5,"name":"Hidden","secrets":[{"secret":"********"}]}],"teams":[]}`))
		case "/api/v1/fleet/teams":
			if tier != "premium" {
				w.WriteHeader(http.StatusPaymentRequired)
				return
			}
			_, _ = w.Write([]byte(`{"teams":[{"id":7,"name":"Old team","secrets":[{"secret":"old"}]}]}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func TestScan(t *testing.T) {
	srv := fakeFleet(t, "premium", true)
	defer srv.Close()
	c, err := NewClient(fleetctl.Credentials{Address: srv.URL, Token: "tok", TLSSkipVerify: true})
	if err != nil {
		t.Fatal(err)
	}
	res, err := c.Scan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if res.ServerURL != "https://fleet.public" || !res.Premium || len(res.Teams) != 4 {
		t.Fatalf("scan = %+v", res)
	}
	if s, _, err := SecretFor(res, nil); err != nil || s != "global" {
		t.Errorf("unassigned secret %q %v", s, err)
	}
	three, four, five, six := uint(3), uint(4), uint(5), uint(6)
	if s, name, err := SecretFor(res, &three); err != nil || s != "ws" || name != "💻 Workstations" {
		t.Errorf("team 3: %q %q %v", s, name, err)
	}
	if _, _, err := SecretFor(res, &four); err == nil || !strings.Contains(err.Error(), "no enroll secret") {
		t.Errorf("team 4 err = %v", err)
	}
	if res.Teams[3].HasSecret || !res.Teams[3].SecretHidden {
		t.Errorf("masked secret treated as usable: %+v", res.Teams[3])
	}
	if _, _, err := SecretFor(res, &five); err == nil || !strings.Contains(err.Error(), "can't read") {
		t.Errorf("team 5 (masked) err = %v", err)
	}
	if _, _, err := SecretFor(res, &six); err == nil {
		t.Error("team 6 should be missing")
	}
	// Secrets must not be serialized to the frontend.
	b, _ := json.Marshal(res)
	if strings.Contains(string(b), "ws") || strings.Contains(string(b), "global") {
		t.Errorf("scan JSON leaks a secret: %s", b)
	}
}

func TestScanOldRoutesAndFree(t *testing.T) {
	old := fakeFleet(t, "premium", false)
	defer old.Close()
	c, _ := NewClient(fleetctl.Credentials{Address: old.URL, Token: "tok", TLSSkipVerify: true})
	res, err := c.Scan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Teams) != 2 || res.Teams[1].Name != "Old team" {
		t.Errorf("old routes teams = %+v", res.Teams)
	}

	free := fakeFleet(t, "free", true)
	defer free.Close()
	c, _ = NewClient(fleetctl.Credentials{Address: free.URL, Token: "tok", TLSSkipVerify: true})
	res, err = c.Scan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if res.Premium || len(res.Teams) != 1 || res.Teams[0].Name != UnassignedName {
		t.Errorf("free scan = %+v", res)
	}

	// A server that isn't running gets an actionable message.
	down := httptest.NewServer(http.NotFoundHandler())
	downURL := down.URL
	down.Close()
	c, _ = NewClient(fleetctl.Credentials{Address: downURL, Token: "tok"})
	if _, err := c.Scan(context.Background()); err == nil || !strings.Contains(err.Error(), "no Fleet server is running") {
		t.Errorf("down server err = %v", err)
	}

	if _, err := NewClient(fleetctl.Credentials{Address: free.URL}); err == nil {
		t.Error("missing token should fail")
	}
}

// TestBuildScript runs build.sh against a stub fleetctl to check the flags it
// passes, that it keeps going after a failure, and the status lines the tab
// parses.
func TestBuildScript(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash not available")
	}
	dir := t.TempDir()
	script := filepath.Join(dir, "build.sh")
	if err := os.WriteFile(script, []byte(BuildScript), 0o755); err != nil {
		t.Fatal(err)
	}
	stub := filepath.Join(dir, "fleetctl")
	stubSrc := `#!/usr/bin/env bash
echo "args: $*" >> "` + filepath.Join(dir, "calls.log") + `"
case "$*" in
  *--type=rpm*) exit 1;;
  *--type=deb*--arch=amd64*) exit 0;; # claims success, writes nothing
  # Mimics fleetctl ignoring --outfile for arm64 .msi.
  *--type=msi*--arch=arm64*) echo new > fleet-osquery-arm64.msi; exit 0;;
esac
for a in "$@"; do case "$a" in --outfile=*) touch "${a#--outfile=}";; esac; done
`
	if err := os.WriteFile(stub, []byte(stubSrc), 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := settings.DefaultInstallerServer()
	cfg.Platforms = []string{"macos", "windows-arm64", "deb-amd64", "deb-arm64", "rpm-amd64"}
	jobs := Jobs("rc", dir, cfg)
	common := CommonArgs(cfg, BuildInputs{FleetURL: "https://f", EnrollSecret: "has space"})

	for _, n := range []string{"fleetd-rc-windows-arm64-desktop.msi", "fleetd-rc-linux-amd64-desktop.deb"} {
		if err := os.WriteFile(filepath.Join(dir, n), []byte("old\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	cmd := exec.Command("bash", script)
	cmd.Env = append(os.Environ(),
		"HANGAR_FLEETCTL="+stub,
		"HANGAR_JOBS="+EncodeJobs(jobs),
		"HANGAR_COMMON_ARGS="+strings.Join(common, "\n"),
	)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected non-zero exit with a failed job, output:\n%s", out)
	}
	for _, want := range []string{
		"==> [macos] building fleetd-rc-macos-desktop.pkg",
		"==> [macos] ok",
		"==> [windows-arm64] ok",
		"==> [deb-amd64] failed",
		"==> [deb-arm64] ok",
		"==> [rpm-amd64] failed",
		"==> done: 3 of 5 built",
	} {
		if !strings.Contains(string(out), want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
	calls, _ := os.ReadFile(filepath.Join(dir, "calls.log"))
	lines := strings.Split(strings.TrimSpace(string(calls)), "\n")
	if len(lines) != 5 {
		t.Fatalf("calls = %q", calls)
	}
	// A run that wrote nothing leaves the previous file alone.
	if b, _ := os.ReadFile(filepath.Join(dir, "fleetd-rc-linux-amd64-desktop.deb")); string(b) != "old\n" {
		t.Errorf("previous deb overwritten: %q", b)
	}
	// No temp dirs or stray files are left next to the outputs.
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if e.IsDir() {
			t.Errorf("leftover dir %s", e.Name())
		}
	}
	// The stray arm64 .msi was moved to its intended name (replacing an old
	// one) and made readable.
	msi := filepath.Join(dir, "fleetd-rc-windows-arm64-desktop.msi")
	if b, err := os.ReadFile(msi); err != nil || string(b) != "new\n" {
		t.Errorf("arm64 msi = %q, %v", b, err)
	}
	if info, err := os.Stat(msi); err == nil && info.Mode().Perm() != 0o644 {
		t.Errorf("arm64 msi mode = %v", info.Mode().Perm())
	}
	if !strings.HasPrefix(lines[0], "args: package --type=pkg --outfile=") || strings.Contains(lines[0], "--arch") {
		t.Errorf("pkg call = %q", lines[0])
	}
	if !strings.Contains(lines[3], "--type=deb --arch=arm64 --outfile=") || !strings.Contains(lines[3], "--enroll-secret=has space --disable-open-folder --fleet-desktop") {
		t.Errorf("deb call = %q", lines[3])
	}
}
