package installers

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"syscall"
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
	for target, want := range map[settings.InstallerTarget]string{
		{TeamName: "💻 Workstations"}:          "workstations",
		{TeamName: UnassignedName}:            "unassigned",
		{EnrollSecret: "typed", TeamName: ""}: ManualFleet,
		{TeamName: "📱"}:                       ManualFleet,
		{TeamName: "📱", TeamID: new(uint(9))}: "fleet-9",
	} {
		if got := FleetSlug(target); got != want {
			t.Errorf("FleetSlug(%+v) = %q, want %q", target, got, want)
		}
	}
}

func TestContextDirName(t *testing.T) {
	// Names that are already safe are used as they are.
	for _, name := range []string{"default", "qa-premium_4.94", "context"} {
		if got := ContextDirName(name); got != name {
			t.Errorf("ContextDirName(%q) = %q, want it unchanged", name, got)
		}
	}
	// Anything else is sanitized, can't add path segments, and gets a hash.
	for _, name := range []string{"../../etc", "a/b c", "..", "Load Test (100k)", "Default"} {
		got := ContextDirName(name)
		if strings.ContainsAny(got, "/\\") || strings.HasPrefix(got, ".") || got == name {
			t.Errorf("ContextDirName(%q) = %q", name, got)
		}
	}
	if got := ContextDirName("a/b c"); !strings.HasPrefix(got, "a-b-c-") {
		t.Errorf("ContextDirName(a/b c) = %q, want a-b-c-<hash>", got)
	}
	// Distinct contexts never share a folder, including on a case-insensitive disk.
	seen := map[string]string{}
	for _, name := range []string{"my ctx", "my-ctx", "my/ctx", "Prod", "prod", "PROD", "..", "context"} {
		dir := strings.ToLower(ContextDirName(name))
		if other, dup := seen[dir]; dup {
			t.Errorf("%q and %q both map to %q", other, name, dir)
		}
		seen[dir] = name
	}
}

func TestOutputDir(t *testing.T) {
	cases := []struct {
		name, fileServer, repo, context, want string
	}{
		{"file server wins", "/srv/files", "/src/fleet", "default", "/srv/files/packages/default"},
		{"repo fallback", "", "/src/fleet", "qa", "/src/fleet/packages/qa"},
		{"manual build", "/srv/files", "/src/fleet", "", "/srv/files/packages"},
	}
	for _, c := range cases {
		got, err := OutputDir(c.fileServer, c.repo, c.context)
		if err != nil || got != c.want {
			t.Errorf("%s: OutputDir = %q, %v; want %q", c.name, got, err, c.want)
		}
	}
	if got, err := OutputDir("", "/src/fleet", "../../tmp"); err != nil || filepath.Dir(got) != "/src/fleet/packages" {
		t.Errorf("context must stay one folder under packages/: %q, %v", got, err)
	}
	if _, err := OutputDir("", "", "default"); err == nil {
		t.Error("no file server dir and no repo should be an error")
	}
	if _, err := OutputDir("relative/dir", "", "default"); err == nil {
		t.Error("a relative folder should be an error")
	}
}

func TestInstallerExt(t *testing.T) {
	cases := map[string]string{
		"fleetd-x-linux-amd64.pkg.tar.zst": ".pkg.tar.zst",
		"fleetd-x-macos.pkg":               ".pkg",
		"a.msi":                            ".msi",
		"a.deb":                            ".deb",
		"a.rpm":                            ".rpm",
		"manifest.json":                    "",
		"a.tar.zst":                        "",
	}
	for name, want := range cases {
		if got := InstallerExt(name); got != want {
			t.Errorf("InstallerExt(%q) = %q, want %q", name, got, want)
		}
	}
}

func TestFileNames(t *testing.T) {
	def := settings.DefaultInstallerOptions()
	var got []string
	for _, j := range Jobs("workstations", "/out/default", def) {
		got = append(got, filepath.Base(j.OutFile))
	}
	want := []string{
		"fleetd-workstations-macos.pkg",
		"fleetd-workstations-windows-amd64.msi",
		"fleetd-workstations-windows-arm64.msi",
		"fleetd-workstations-linux-amd64.deb",
		"fleetd-workstations-linux-arm64.deb",
		"fleetd-workstations-linux-amd64.rpm",
		"fleetd-workstations-linux-arm64.rpm",
		"fleetd-workstations-linux-amd64.pkg.tar.zst",
		"fleetd-workstations-linux-arm64.pkg.tar.zst",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("default names\n got %v\nwant %v", got, want)
	}

	deb, _ := PlatformByKey("deb-arm64")
	pkg, _ := PlatformByKey("macos")
	cfg := def
	cfg.FleetDesktop = false
	cfg.EnableScripts = false
	if n := FileName("unassigned", deb, cfg); n != "fleetd-unassigned-linux-arm64-nodesktop-noscripts.deb" {
		t.Errorf("no desktop, no scripts: %q", n)
	}
	cfg = def
	cfg.UpdateSource = settings.UpdateSourceLocalTUF
	cfg.OrbitChannel = "edge"
	if n := FileName("rc", pkg, cfg); n != "fleetd-rc-macos-tuf-edge.pkg" {
		t.Errorf("tuf+edge: %q", n)
	}
	cfg = def
	cfg.OrbitChannel, cfg.DesktopChannel, cfg.OsquerydChannel = "edge", "edge", "edge"
	if n := FileName("rc", pkg, cfg); n != "fleetd-rc-macos-edge.pkg" {
		t.Errorf("all edge: %q", n)
	}
	cfg = def
	cfg.OrbitChannel, cfg.OsquerydChannel = "1.61.0", "edge"
	if n := FileName("rc", pkg, cfg); n != "fleetd-rc-macos-orbit-1.61.0-osquery-edge.pkg" {
		t.Errorf("mixed channels: %q", n)
	}
}

func TestJobsSubsetAndEncode(t *testing.T) {
	cfg := settings.DefaultInstallerOptions()
	cfg.Platforms = []string{"arch-arm64", "rpm-arm64", "bogus", "macos"} // order follows Platforms
	jobs := Jobs("rc", "/o", cfg)
	if len(jobs) != 3 || jobs[0].Key != "macos" || jobs[1].Key != "rpm-arm64" || jobs[2].Key != "arch-arm64" {
		t.Fatalf("jobs = %+v", jobs)
	}
	enc := EncodeJobs(jobs)
	wantEnc := "macos\tpkg\t-\t/o/fleetd-rc-macos.pkg\n" +
		"rpm-arm64\trpm\tarm64\t/o/fleetd-rc-linux-arm64.rpm\n" +
		"arch-arm64\tpkg.tar.zst\tarm64\t/o/fleetd-rc-linux-arm64.pkg.tar.zst\n"
	if enc != wantEnc {
		t.Errorf("EncodeJobs = %q", enc)
	}
}

func TestCommonArgs(t *testing.T) {
	in := BuildInputs{FleetURL: "https://fleet.example", EnrollSecret: "s3cret", UpdateURL: "https://tuf.example", UpdateRoots: `[{"k":1}]`}
	def := settings.DefaultInstallerOptions()
	got := CommonArgs(def, in)
	want := []string{"--fleet-url=https://fleet.example", "--enroll-secret=s3cret", "--disable-open-folder", "--fleet-desktop", "--enable-scripts"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("default args %v", got)
	}

	cfg := def
	cfg.FleetDesktop = false
	cfg.EnableScripts = false
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
	cfg := settings.DefaultInstallerOptions()
	if err := ValidateConfig(cfg, settings.InstallerTarget{FleetURL: "https://f"}); err != nil {
		t.Errorf("default config: %v", err)
	}
	if err := ValidateConfig(cfg, settings.InstallerTarget{FleetURL: "https://f\n--insecure"}); err == nil {
		t.Error("newline in Fleet URL should be rejected")
	}
}

func TestValidateArgs(t *testing.T) {
	ok := CommonArgs(settings.DefaultInstallerOptions(), BuildInputs{FleetURL: "https://f", EnrollSecret: "s"})
	if err := ValidateArgs(ok); err != nil {
		t.Errorf("valid args: %v", err)
	}
	// A server-supplied value with a line break is caught, not split into flags.
	bad := CommonArgs(settings.DefaultInstallerOptions(), BuildInputs{FleetURL: "https://f", EnrollSecret: "s\n--insecure"})
	if err := ValidateArgs(bad); err == nil || !strings.Contains(err.Error(), "--enroll-secret") {
		t.Errorf("secret with newline: %v", err)
	}
}

func TestParseExtraFlags(t *testing.T) {
	got, err := ParseExtraFlags(`--use-system-configuration --host-identifier=instance  --osquery-flagfile "/tmp/my flags.txt" --end-user-email='a b@x.io'`)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"--use-system-configuration", "--host-identifier=instance", "--osquery-flagfile", "/tmp/my flags.txt", "--end-user-email=a b@x.io"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ParseExtraFlags\n got %q\nwant %q", got, want)
	}
	if got, err := ParseExtraFlags("   "); err != nil || len(got) != 0 {
		t.Errorf("blank: %q %v", got, err)
	}
	for _, bad := range []string{
		`--fleet-url=https://x`, `--outfile x.pkg`, `--enable-scripts`, `--host-identifier "unclosed`,
		// fleetctl takes a single dash too.
		`-outfile=/elsewhere/x.pkg`, `-type deb`, `--host-identifier=instance -enroll-secret=`,
	} {
		if _, err := ParseExtraFlags(bad); err == nil {
			t.Errorf("%q should be rejected", bad)
		}
	}
	// Extra flags go last on the fleetctl command line.
	args := CommonArgs(settings.DefaultInstallerOptions(), BuildInputs{FleetURL: "https://f", EnrollSecret: "s", ExtraFlags: want[:2]})
	if args[len(args)-2] != "--use-system-configuration" || args[len(args)-1] != "--host-identifier=instance" {
		t.Errorf("args = %q", args)
	}
}

func TestServerForAddress(t *testing.T) {
	servers := []settings.ServerProfile{
		{ID: "s1", Ports: settings.ServerPorts{Server: 8080}},
		{ID: "s2", Ports: settings.ServerPorts{Server: 8090}},
	}
	cases := map[string]string{
		"https://localhost:8080":       "s1",
		"https://127.0.0.1:8090/":      "s2",
		"https://[::1]:8090":           "s2",
		"https://localhost:8100":       "",
		"https://fleetqa.ngrok.io":     "",
		"https://fleet.example:8080":   "",
		"":                             "",
		"not a url with spaces ::8080": "",
	}
	for addr, want := range cases {
		got := ""
		if srv := ServerForAddress(servers, addr); srv != nil {
			got = srv.ID
		}
		if got != want {
			t.Errorf("ServerForAddress(%q) = %q, want %q", addr, got, want)
		}
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
	write := func(path string) {
		t.Helper()
		if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// Written just now, like a package from a build that ended a moment ago,
	// so its modification time alone can't tell it from this build's output.
	old := filepath.Join(dir, "fleetd-rc-macos.pkg")
	write(old)
	msi := filepath.Join(dir, "fleetd-rc-windows-amd64.msi")
	write(msi)
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

	built := filepath.Join(dir, "fleetd-rc-linux-arm64-desktop.deb")
	jobs := []Job{
		{Key: "deb-arm64", OutFile: built},
		{Key: "rpm-arm64", OutFile: filepath.Join(dir, "fleetd-rc-linux-arm64-desktop.rpm")}, // failed: no file
		{Key: "macos", OutFile: old},                                                         // failed rebuild: old file left in place
		{Key: "windows-amd64", OutFile: msi},                                                 // rebuilt over an existing file
	}
	before := Snapshot(jobs)

	// The build: a new .deb, and the .msi replaced the way build.sh does it.
	write(built)
	tmp := filepath.Join(dir, ".work-msi")
	write(tmp)
	if err := os.Rename(tmp, msi); err != nil {
		t.Fatal(err)
	}

	n, err := UpdateManifest(dir, jobs, before, ManifestEntry{Team: "Workstations", FleetURL: "https://f"})
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Errorf("built = %d, want 2", n)
	}
	var m Manifest
	b, _ = os.ReadFile(filepath.Join(dir, ManifestName))
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	if len(m.Files) != 3 {
		t.Fatalf("manifest files = %v", m.Files)
	}
	for name, platform := range map[string]string{"fleetd-rc-linux-arm64-desktop.deb": "deb-arm64", "fleetd-rc-windows-amd64.msi": "windows-amd64"} {
		if e := m.Files[name]; e.Platform != platform || e.Team != "Workstations" {
			t.Errorf("entry for %s = %+v", name, e)
		}
	}
	if e := m.Files["fleetd-rc-macos.pkg"]; e.Team != "old" {
		t.Errorf("kept entry = %+v", e)
	}
}

func TestListOutputs(t *testing.T) {
	dir := t.TempDir()
	for _, n := range []string{"a.pkg", "b.deb", "c.pkg.tar.zst", "manifest.json", "notes.txt"} {
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
	if len(got) != 3 {
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
		// "fleet-admin" is a user with a role on a fleet only.
		auth := r.Header.Get("Authorization")
		if auth != "Bearer tok" && auth != "Bearer fleet-admin" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		switch r.URL.Path {
		case "/api/v1/fleet/config":
			_, _ = w.Write([]byte(`{"server_settings":{"server_url":"https://fleet.public"},"license":{"tier":"` + tier + `"}}`))
		case "/api/v1/fleet/spec/enroll_secret":
			if auth == "Bearer fleet-admin" {
				w.WriteHeader(http.StatusForbidden)
				return
			}
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

func TestScanHonorsURLPrefixAndCustomHeaders(t *testing.T) {
	inner := fakeFleet(t, "premium", true)
	defer inner.Close()
	// Like a server behind an access proxy on a path prefix.
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rest, ok := strings.CutPrefix(r.URL.Path, "/fleet-prefix")
		if !ok || r.Header.Get("X-Access") != "letmein" || r.Header.Get("Authorization") != "Bearer tok" {
			http.Error(w, "blocked by proxy", http.StatusForbidden)
			return
		}
		r.URL.Path = rest
		inner.Config.Handler.ServeHTTP(w, r)
	}))
	defer proxy.Close()
	c, err := NewClient(fleetctl.Credentials{
		Address: proxy.URL, Token: "tok", URLPrefix: "/fleet-prefix/",
		// Like fleetctl, a custom header can't replace the context's token.
		CustomHeaders: map[string]string{"X-Access": "letmein", "Authorization": "Bearer other"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res, err := c.Scan(context.Background()); err != nil || res.ServerURL != "https://fleet.public" {
		t.Fatalf("scan through the proxy = %+v, %v", res, err)
	}
}

func TestScanWithoutGlobalSecretAccess(t *testing.T) {
	srv := fakeFleet(t, "premium", true)
	defer srv.Close()
	c, err := NewClient(fleetctl.Credentials{Address: srv.URL, Token: "fleet-admin", TLSSkipVerify: true})
	if err != nil {
		t.Fatal(err)
	}
	res, err := c.Scan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Teams) != 4 || res.Teams[0].Name != UnassignedName || res.Teams[0].HasSecret || !res.Teams[0].SecretHidden {
		t.Fatalf("scan = %+v", res)
	}
	if _, _, err := SecretFor(res, nil); err == nil || !strings.Contains(err.Error(), "can't read") {
		t.Errorf("unassigned err = %v", err)
	}
	three := uint(3)
	if s, _, err := SecretFor(res, &three); err != nil || s != "ws" {
		t.Errorf("team 3: %q %v", s, err)
	}
}

func TestScanErrorsSayWhatToFix(t *testing.T) {
	srv := fakeFleet(t, "premium", true)
	defer srv.Close()

	c, err := NewClient(fleetctl.Credentials{Address: srv.URL, Token: "expired", TLSSkipVerify: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Scan(context.Background()); err == nil || !strings.Contains(err.Error(), "fleetctl login") {
		t.Errorf("rejected token: %v", err)
	}

	c, err = NewClient(fleetctl.Credentials{Address: srv.URL, Token: "tok"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Scan(context.Background()); err == nil || !strings.Contains(err.Error(), "tls-skip-verify") {
		t.Errorf("untrusted certificate: %v", err)
	}

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	down := "https://" + l.Addr().String()
	l.Close()
	c, err = NewClient(fleetctl.Credentials{Address: down, Token: "tok", TLSSkipVerify: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Scan(context.Background()); err == nil || !strings.Contains(err.Error(), "no Fleet server is running") {
		t.Errorf("server down: %v", err)
	}

	if _, err := NewClient(fleetctl.Credentials{Address: srv.URL}); err == nil || !strings.Contains(err.Error(), "fleetctl login") {
		t.Errorf("no token: %v", err)
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
// buildScriptCmd writes build.sh and a fleetctl stub into a new dir and
// returns a command that builds platforms with them into that dir.
func buildScriptCmd(t *testing.T, stubSrc string, platforms ...string) (*exec.Cmd, string) {
	t.Helper()
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash not available")
	}
	dir := t.TempDir()
	script := filepath.Join(dir, "build.sh")
	if err := os.WriteFile(script, []byte(BuildScript), 0o755); err != nil {
		t.Fatal(err)
	}
	stub := filepath.Join(dir, "fleetctl")
	if err := os.WriteFile(stub, []byte(stubSrc), 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := settings.DefaultInstallerOptions()
	cfg.Platforms = platforms
	common := CommonArgs(cfg, BuildInputs{FleetURL: "https://f", EnrollSecret: "has space"})
	cmd := exec.Command("bash", script)
	cmd.Env = append(os.Environ(),
		"HANGAR_FLEETCTL="+stub,
		"HANGAR_JOBS="+EncodeJobs(Jobs("rc", dir, cfg)),
		"HANGAR_COMMON_ARGS="+strings.Join(common, "\n"),
	)
	return cmd, dir
}

func TestBuildScript(t *testing.T) {
	calls := filepath.Join(t.TempDir(), "calls.log")
	cmd, dir := buildScriptCmd(t, `#!/usr/bin/env bash
echo "args: $*" >> "`+calls+`"
echo "to stdout"
echo "to stderr" >&2
case "$*" in
  *--type=rpm*) exit 1;;
  *--type=deb*--arch=amd64*) exit 0;; # claims success, writes nothing
  # Mimics fleetctl ignoring --outfile for arm64 .msi.
  *--type=msi*--arch=arm64*) echo new > fleet-osquery-arm64.msi; exit 0;;
esac
for a in "$@"; do case "$a" in --outfile=*) touch "${a#--outfile=}";; esac; done
`, "macos", "windows-arm64", "deb-amd64", "deb-arm64", "rpm-amd64", "arch-amd64")

	for _, n := range []string{"fleetd-rc-windows-arm64.msi", "fleetd-rc-linux-amd64.deb"} {
		if err := os.WriteFile(filepath.Join(dir, n), []byte("old\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	// A work dir left by a build that was killed.
	if err := os.MkdirAll(filepath.Join(dir, ".hangar-build.stale", "macos.x"), 0o755); err != nil {
		t.Fatal(err)
	}

	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected non-zero exit with a failed job, output:\n%s", out)
	}
	for _, want := range []string{
		"==> [macos] building fleetd-rc-macos.pkg",
		"==> [macos] ok",
		"==> [windows-arm64] ok",
		"==> [deb-amd64] failed",
		"==> [deb-arm64] ok",
		"==> [rpm-amd64] failed",
		"==> [arch-amd64] ok",
		"==> done: 4 of 6 built",
		// A job's own output, from either stream, is tagged with its key.
		"[arch-amd64] to stdout",
		"[arch-amd64] to stderr",
		"[deb-amd64] fleetctl reported success but wrote no deb file",
	} {
		if !strings.Contains(string(out), want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
	logged, _ := os.ReadFile(calls)
	lines := strings.Split(strings.TrimSpace(string(logged)), "\n")
	if len(lines) != 6 {
		t.Fatalf("calls = %q", logged)
	}
	// Jobs run at once, so calls are logged in no particular order.
	call := func(substr string) string {
		for _, l := range lines {
			if strings.Contains(l, substr) {
				return l
			}
		}
		t.Fatalf("no call with %q in %q", substr, logged)
		return ""
	}
	// A run that wrote nothing leaves the previous file alone.
	if b, _ := os.ReadFile(filepath.Join(dir, "fleetd-rc-linux-amd64.deb")); string(b) != "old\n" {
		t.Errorf("previous deb overwritten: %q", b)
	}
	// No work dirs (this build's or the stale one) are left next to the outputs.
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if e.IsDir() {
			t.Errorf("leftover dir %s", e.Name())
		}
	}
	// Jobs write inside the output folder, so the final move is a rename.
	if pkg := call("--type=pkg "); !strings.Contains(pkg, "--outfile="+filepath.Join(dir, ".hangar-build.")) {
		t.Errorf("pkg built outside the output folder: %q", pkg)
	}
	// The stray arm64 .msi was moved to its intended name (replacing an old
	// one) and made readable.
	msi := filepath.Join(dir, "fleetd-rc-windows-arm64.msi")
	if b, err := os.ReadFile(msi); err != nil || string(b) != "new\n" {
		t.Errorf("arm64 msi = %q, %v", b, err)
	}
	if info, err := os.Stat(msi); err == nil && info.Mode().Perm() != 0o644 {
		t.Errorf("arm64 msi mode = %v", info.Mode().Perm())
	}
	if pkg := call("--type=pkg "); !strings.HasPrefix(pkg, "args: package --type=pkg --outfile=") || strings.Contains(pkg, "--arch") {
		t.Errorf("pkg call = %q", pkg)
	}
	if _, err := os.Stat(filepath.Join(dir, "fleetd-rc-linux-amd64.pkg.tar.zst")); err != nil {
		t.Errorf("Arch package not moved into place: %v", err)
	}
	if deb := call("--type=deb --arch=arm64 "); !strings.Contains(deb, "--outfile=") || !strings.Contains(deb, "--enroll-secret=has space --disable-open-folder --fleet-desktop --enable-scripts") {
		t.Errorf("deb call = %q", deb)
	}
}

func TestBuildScriptSignaledJobHasNoResult(t *testing.T) {
	// A job whose fleetctl dies by a signal (as under Stop) reports neither ok
	// nor failed: the tab shows it as stopped or interrupted.
	cmd, _ := buildScriptCmd(t, `#!/usr/bin/env bash
case "$*" in
  *--type=rpm*) kill -TERM $$;;
esac
for a in "$@"; do case "$a" in --outfile=*) touch "${a#--outfile=}";; esac; done
`, "deb-amd64", "rpm-amd64")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected non-zero exit with a killed job, output:\n%s", out)
	}
	for _, want := range []string{"==> [deb-amd64] ok", "==> [rpm-amd64] building", "==> done: 1 of 2 built"} {
		if !strings.Contains(string(out), want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
	for _, unwanted := range []string{"==> [rpm-amd64] failed", "==> [rpm-amd64] ok"} {
		if strings.Contains(string(out), unwanted) {
			t.Errorf("output has %q:\n%s", unwanted, out)
		}
	}
}

func TestBuildScriptRunsJobsAtOnce(t *testing.T) {
	// Each stub run waits for all three to have started, which can only
	// happen if they run at the same time.
	started := t.TempDir()
	cmd, _ := buildScriptCmd(t, `#!/usr/bin/env bash
touch "`+started+`/$$"
for _ in $(seq 1 100); do
  [ "$(ls "`+started+`" | wc -l)" -ge 3 ] && break
  sleep 0.05
done
[ "$(ls "`+started+`" | wc -l)" -ge 3 ] || { echo "ran alone"; exit 1; }
for a in "$@"; do case "$a" in --outfile=*) touch "${a#--outfile=}";; esac; done
`, "deb-amd64", "rpm-amd64", "arch-amd64")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build failed (%v):\n%s", err, out)
	}
}

func TestBuildScriptStopEndsEveryJob(t *testing.T) {
	for name, stub := range map[string]string{
		"killed by the signal": "exec sleep 30",
		// fleetctl can catch SIGTERM to clean up and then exit non-zero.
		"exits non-zero on SIGTERM": "trap 'sleep 0.2; exit 1' TERM\nsleep 30 & wait",
	} {
		t.Run(name, func(t *testing.T) {
			pids := filepath.Join(t.TempDir(), "pids")
			cmd, dir := buildScriptCmd(t, "#!/usr/bin/env bash\necho $$ >> \""+pids+"\"\n"+stub+"\n", "deb-amd64", "rpm-amd64", "arch-amd64")
			var out bytes.Buffer
			cmd.Stdout, cmd.Stderr = &out, &out
			// Like the process engine: own group, and Stop signals the group.
			cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
			if err := cmd.Start(); err != nil {
				t.Fatal(err)
			}
			var running []int
			for deadline := time.Now().Add(5 * time.Second); len(running) < 3; {
				if time.Now().After(deadline) {
					_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
					t.Fatalf("jobs never all started: %v", running)
				}
				time.Sleep(20 * time.Millisecond)
				b, _ := os.ReadFile(pids)
				running = running[:0]
				for _, f := range strings.Fields(string(b)) {
					if pid, err := strconv.Atoi(f); err == nil {
						running = append(running, pid)
					}
				}
			}

			if err := syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM); err != nil {
				t.Fatal(err)
			}
			exited := make(chan error, 1)
			go func() { exited <- cmd.Wait() }()
			select {
			case <-exited:
			case <-time.After(5 * time.Second):
				_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
				t.Fatal("build didn't exit after SIGTERM")
			}
			if code := cmd.ProcessState.ExitCode(); code != 143 {
				t.Errorf("exit code = %d, want 143", code)
			}
			// A job cut short by Stop was stopped, not failed: the tab would show it
			// as a failed build.
			if strings.Contains(out.String(), "] failed") {
				t.Errorf("a stopped job reported failure:\n%s", out.String())
			}
			for _, pid := range running {
				for deadline := time.Now().Add(2 * time.Second); syscall.Kill(pid, 0) == nil; {
					if time.Now().After(deadline) {
						_ = syscall.Kill(pid, syscall.SIGKILL)
						t.Errorf("job %d still running after Stop", pid)
						break
					}
					time.Sleep(20 * time.Millisecond)
				}
			}
			entries, _ := os.ReadDir(dir)
			for _, e := range entries {
				if e.IsDir() {
					t.Errorf("work dir left behind: %s", e.Name())
				}
			}
		})
	}
}
