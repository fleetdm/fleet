// Package installers builds fleetd installers (fleetctl package) for a
// fleetctl context's Fleet server, or for a typed-in URL and enroll secret,
// named so they can be told apart at a glance and shared with test machines.
// It covers the pure parts — platform list, output folder and file naming, the
// fleetctl arguments, matching a context to a Hangar server, reading channels
// from a TUF repo, the manifest — plus a small Fleet API client (fleetapi.go)
// for scanning a server's fleets and enroll secrets.
//
// A build runs build.sh (embedded) through the process engine, so output
// streams to a log channel like the TUF tab. One build runs at a time; its
// platforms build in parallel.
package installers

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/fleetdm/fleet/tools/hangar/internal/paths"
	"github.com/fleetdm/fleet/tools/hangar/internal/settings"
)

const (
	// ProcID / LogChannel are the build's process-engine id and log channel
	// (mirrored in lib/installers.ts).
	ProcID       = "installers:build"
	LogChannel   = "installers-build"
	ManifestName = "manifest.json"
	// PackagesDirName is the folder packages land in, under the file server's
	// directory or the primary repo.
	PackagesDirName = "packages"
)

//go:embed build.sh
var BuildScript string

// Platform is one installer the tab can build.
type Platform struct {
	Key  string `json:"key"`
	OS   string `json:"os"`   // macos | windows | linux — the name's OS part
	Type string `json:"type"` // fleetctl package --type
	Arch string `json:"arch"` // fleetctl package --arch; "" for the universal pkg
}

// Platforms is every installer the tab offers (matches lib/installers.ts).
var Platforms = []Platform{
	{Key: "macos", OS: "macos", Type: "pkg"},
	{Key: "windows-amd64", OS: "windows", Type: "msi", Arch: "amd64"},
	{Key: "windows-arm64", OS: "windows", Type: "msi", Arch: "arm64"},
	{Key: "deb-amd64", OS: "linux", Type: "deb", Arch: "amd64"},
	{Key: "deb-arm64", OS: "linux", Type: "deb", Arch: "arm64"},
	{Key: "rpm-amd64", OS: "linux", Type: "rpm", Arch: "amd64"},
	{Key: "rpm-arm64", OS: "linux", Type: "rpm", Arch: "arm64"},
	{Key: "arch-amd64", OS: "linux", Type: "pkg.tar.zst", Arch: "amd64"},
	{Key: "arch-arm64", OS: "linux", Type: "pkg.tar.zst", Arch: "arm64"},
}

// installerExts are the file extensions of every package type, longest first
// so ".pkg.tar.zst" isn't read as ".pkg".
var installerExts = []string{".pkg.tar.zst", ".pkg", ".msi", ".deb", ".rpm"}

// InstallerExt is name's package extension (".pkg.tar.zst" whole, unlike
// filepath.Ext), or "" when it isn't a package.
func InstallerExt(name string) string {
	for _, ext := range installerExts {
		if strings.HasSuffix(name, ext) {
			return ext
		}
	}
	return ""
}

// PlatformByKey looks up a platform by its key.
func PlatformByKey(key string) (Platform, bool) {
	for _, p := range Platforms {
		if p.Key == key {
			return p, true
		}
	}
	return Platform{}, false
}

var nonSlug = regexp.MustCompile(`[^a-z0-9]+`)

// Slug turns a display name into a file-name-safe token: lowercase ASCII
// letters/digits joined by single hyphens ("Server 2 (n-1)" → "server-2-n-1").
// Emoji and other symbols are dropped.
func Slug(name string) string {
	return strings.Trim(nonSlug.ReplaceAllString(strings.ToLower(name), "-"), "-")
}

// ManualFleet names the files of a build whose enroll secret was typed in.
const ManualFleet = "manual"

// FleetSlug is the fleet part of the file names: the picked fleet's name
// ("unassigned" for the global secret), or ManualFleet for a typed-in secret.
// A fleet whose name has nothing usable (all emoji) goes by its id.
func FleetSlug(t settings.InstallerTarget) string {
	if s := Slug(t.TeamName); s != "" {
		return s
	}
	if t.TeamName != "" && t.TeamID != nil {
		return fmt.Sprintf("fleet-%d", *t.TeamID)
	}
	return ManualFleet
}

var nonContextSafe = regexp.MustCompile(`[^a-z0-9._-]+`)

// ContextDirName is the context's folder under packages/: its name lowercased
// with anything but letters, digits, '.', '_' and '-' replaced, so it can't
// add path segments. When that changes the name, a hash of the original is
// appended: "my ctx" and "my-ctx" (or "Prod" and "prod" on a case-insensitive
// disk) must not share a folder, or one's packages would replace the other's.
func ContextDirName(context string) string {
	name := strings.Trim(nonContextSafe.ReplaceAllString(strings.ToLower(context), "-"), "-.")
	if name == context && name != "" {
		return name
	}
	sum := sha256.Sum256([]byte(context))
	if name == "" {
		name = "context"
	}
	return name + "-" + hex.EncodeToString(sum[:3])
}

// OutputDir is where a build's packages go: <base>/packages/<context>/, or
// <base>/packages/ for a manual build (context ""). base is the file server's
// directory when one is set, so the packages can be downloaded from it right
// away, else the primary repo.
func OutputDir(fileServerDir, repo, context string) (string, error) {
	base := strings.TrimSpace(fileServerDir)
	if base == "" {
		base = strings.TrimSpace(repo)
	}
	if base == "" {
		return "", errors.New("set the Fleet repo (or the file server's directory) in Settings first")
	}
	base = paths.Expand(base)
	if !filepath.IsAbs(base) {
		return "", fmt.Errorf("packages folder must be an absolute path (got %s)", base)
	}
	dir := filepath.Join(base, PackagesDirName)
	if context != "" {
		dir = filepath.Join(dir, ContextDirName(context))
	}
	return dir, nil
}

// channelSuffix describes non-stable channels for the file name: "" when all
// three are stable, "-edge" when every non-stable one is edge, and otherwise
// one "-<component>-<channel>" per non-stable component (e.g. a version pin
// from a local TUF repo: "-orbit-1.61.0").
func channelSuffix(cfg settings.InstallerOptions) string {
	type comp struct{ name, ch string }
	var custom []comp
	for _, c := range []comp{{"orbit", cfg.OrbitChannel}, {"desktop", cfg.DesktopChannel}, {"osquery", cfg.OsquerydChannel}} {
		if c.ch != "" && c.ch != "stable" {
			custom = append(custom, c)
		}
	}
	if len(custom) == 0 {
		return ""
	}
	allEdge := true
	for _, c := range custom {
		if c.ch != "edge" {
			allEdge = false
		}
	}
	if allEdge {
		return "-edge"
	}
	var b strings.Builder
	for _, c := range custom {
		b.WriteString("-" + c.name + "-" + strings.Trim(nonSlugKeepDots.ReplaceAllString(strings.ToLower(c.ch), "-"), "-"))
	}
	return b.String()
}

var nonSlugKeepDots = regexp.MustCompile(`[^a-z0-9.]+`)

// FileName is the installer's file name:
//
//	fleetd-<fleet>-<os>[-<arch>][-tuf][-<channels>][-nodesktop][-noscripts].<ext>
//
// e.g. fleetd-workstations-linux-arm64.deb. Only settings that differ from the
// defaults add a part, so a default rebuild overwrites the same file (and its
// share link keeps working).
func FileName(fleetSlug string, p Platform, opts settings.InstallerOptions) string {
	var b strings.Builder
	b.WriteString("fleetd-" + fleetSlug + "-" + p.OS)
	if p.Arch != "" {
		b.WriteString("-" + p.Arch)
	}
	if opts.UpdateSource == settings.UpdateSourceLocalTUF {
		b.WriteString("-tuf")
	}
	b.WriteString(channelSuffix(opts))
	if !opts.FleetDesktop {
		b.WriteString("-nodesktop")
	}
	if !opts.EnableScripts {
		b.WriteString("-noscripts")
	}
	b.WriteString("." + p.Type)
	return b.String()
}

// Job is one fleetctl package run of a build.
type Job struct {
	Key     string
	Type    string
	Arch    string
	OutFile string
}

// Jobs lists the runs for opts' selected platforms, in Platforms order,
// writing into outDir. Unknown keys are ignored.
func Jobs(fleetSlug, outDir string, opts settings.InstallerOptions) []Job {
	selected := map[string]bool{}
	for _, k := range opts.Platforms {
		selected[k] = true
	}
	var jobs []Job
	for _, p := range Platforms {
		if !selected[p.Key] {
			continue
		}
		jobs = append(jobs, Job{Key: p.Key, Type: p.Type, Arch: p.Arch, OutFile: filepath.Join(outDir, FileName(fleetSlug, p, opts))})
	}
	return jobs
}

// EncodeJobs renders jobs as the tab-separated lines build.sh reads. An empty
// arch is written as "-": bash's read collapses consecutive tabs, so an empty
// field would shift the rest.
func EncodeJobs(jobs []Job) string {
	var b strings.Builder
	for _, j := range jobs {
		arch := j.Arch
		if arch == "" {
			arch = "-"
		}
		fmt.Fprintf(&b, "%s\t%s\t%s\t%s\n", j.Key, j.Type, arch, j.OutFile)
	}
	return b.String()
}

// BuildInputs is everything resolved at build time that isn't in the saved
// options.
type BuildInputs struct {
	FleetURL     string
	EnrollSecret string
	// UpdateURL / UpdateRoots are set only for a local TUF build.
	UpdateURL   string
	UpdateRoots string
	// ExtraFlags are the user's additional flags (see ParseExtraFlags).
	ExtraFlags []string
}

// CommonArgs are the fleetctl package flags shared by every job of a build
// (build.sh adds --type, --arch and --outfile per job).
func CommonArgs(cfg settings.InstallerOptions, in BuildInputs) []string {
	args := []string{
		"--fleet-url=" + in.FleetURL,
		"--enroll-secret=" + in.EnrollSecret,
		"--disable-open-folder",
	}
	if cfg.FleetDesktop {
		args = append(args, "--fleet-desktop")
	}
	if cfg.EnableScripts {
		args = append(args, "--enable-scripts")
	}
	if cfg.Debug {
		args = append(args, "--debug")
	}
	for _, c := range []struct{ flag, ch string }{
		{"--orbit-channel", cfg.OrbitChannel},
		{"--desktop-channel", cfg.DesktopChannel},
		{"--osqueryd-channel", cfg.OsquerydChannel},
	} {
		if c.ch != "" && c.ch != "stable" {
			args = append(args, c.flag+"="+c.ch)
		}
	}
	if cfg.UpdateSource == settings.UpdateSourceLocalTUF {
		args = append(args, "--update-url="+in.UpdateURL, "--update-roots="+in.UpdateRoots)
	}
	return append(args, in.ExtraFlags...)
}

// managedFlags are the fleetctl package flags Hangar sets itself; extra flags
// may not repeat them.
var managedFlags = map[string]bool{
	"--type": true, "--arch": true, "--outfile": true, "--fleet-url": true, "--enroll-secret": true,
	"--fleet-desktop": true, "--enable-scripts": true, "--debug": true, "--disable-open-folder": true,
	"--orbit-channel": true, "--desktop-channel": true, "--osqueryd-channel": true,
	"--update-url": true, "--update-roots": true,
}

// ParseExtraFlags splits extra `fleetctl package` flags like a shell would
// (whitespace-separated, with '…' and "…" quoting) and rejects flags Hangar
// already sets. Values may be attached (--flag=value) or follow the flag.
func ParseExtraFlags(s string) ([]string, error) {
	var out []string
	var cur strings.Builder
	inTok := false
	var quote rune
	for _, r := range s {
		switch {
		case quote != 0:
			if r == quote {
				quote = 0
			} else {
				cur.WriteRune(r)
			}
		case r == '\'' || r == '"':
			quote, inTok = r, true
		case r == ' ' || r == '\t' || r == '\n' || r == '\r':
			if inTok {
				out = append(out, cur.String())
				cur.Reset()
				inTok = false
			}
		default:
			cur.WriteRune(r)
			inTok = true
		}
	}
	if quote != 0 {
		return nil, errors.New("extra flags: unclosed quote")
	}
	if inTok {
		out = append(out, cur.String())
	}
	for _, a := range out {
		name, _, _ := strings.Cut(a, "=")
		// fleetctl reads -flag like --flag, and the last one given wins.
		if strings.HasPrefix(name, "-") {
			name = "--" + strings.TrimLeft(name, "-")
		}
		if managedFlags[name] {
			return nil, fmt.Errorf("extra flags: %s is set by Hangar — use the tab's own setting", name)
		}
	}
	return out, nil
}

// ServerForAddress is the Hangar server a fleetctl context's address points
// at (a loopback host on that server's port), or nil. Its fleetctl is the one
// to build with, so packages match the server's version.
func ServerForAddress(servers []settings.ServerProfile, address string) *settings.ServerProfile {
	u, err := url.Parse(strings.TrimSpace(address))
	if err != nil || u.Host == "" {
		return nil
	}
	switch u.Hostname() {
	case "localhost", "127.0.0.1", "::1":
	default:
		return nil
	}
	port := u.Port()
	if port == "" {
		port = map[string]string{"https": "443", "http": "80"}[u.Scheme]
	}
	for i := range servers {
		if strconv.Itoa(int(servers[i].Ports.Server)) == port {
			return &servers[i]
		}
	}
	return nil
}

// Channels lists the channels a package can point each component at.
type Channels struct {
	Orbit    []string `json:"orbit"`
	Desktop  []string `json:"desktop"`
	Osqueryd []string `json:"osqueryd"`
}

// OfficialChannels are the channels published on updates.fleetdm.com that
// Hangar offers (see orbit/TUF.md).
func OfficialChannels() Channels {
	return Channels{
		Orbit:    []string{"stable", "edge"},
		Desktop:  []string{"stable", "edge"},
		Osqueryd: []string{"stable", "edge"},
	}
}

// TUFChannels reads the channels a TUF repo actually publishes from its
// targets.json (target paths are <component>/<platform>/<channel>/<file>).
// stable sorts first, then the rest in version order.
func TUFChannels(targetsJSON []byte) (Channels, error) {
	var doc struct {
		Signed struct {
			Targets map[string]json.RawMessage `json:"targets"`
		} `json:"signed"`
	}
	if err := json.Unmarshal(targetsJSON, &doc); err != nil {
		return Channels{}, fmt.Errorf("parse targets.json: %w", err)
	}
	sets := map[string]map[string]bool{"orbit": {}, "desktop": {}, "osqueryd": {}}
	for path := range doc.Signed.Targets {
		parts := strings.Split(path, "/")
		if len(parts) < 4 {
			continue
		}
		if set, ok := sets[parts[0]]; ok {
			set[parts[2]] = true
		}
	}
	return Channels{Orbit: sortChannels(sets["orbit"]), Desktop: sortChannels(sets["desktop"]), Osqueryd: sortChannels(sets["osqueryd"])}, nil
}

func sortChannels(set map[string]bool) []string {
	out := make([]string, 0, len(set))
	for ch := range set {
		out = append(out, ch)
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if (a == "stable") != (b == "stable") {
			return a == "stable"
		}
		if (a == "edge") != (b == "edge") {
			return a == "edge"
		}
		return versionLess(a, b)
	})
	return out
}

// versionLess orders dotted numeric versions numerically ("5" < "5.23" <
// "5.24.0" < "26.10"), falling back to string order for anything else.
func versionLess(a, b string) bool {
	as, bs := strings.Split(a, "."), strings.Split(b, ".")
	for i := 0; i < len(as) && i < len(bs); i++ {
		ai, errA := strconv.Atoi(as[i])
		bi, errB := strconv.Atoi(bs[i])
		if errA != nil || errB != nil {
			return a < b
		}
		if ai != bi {
			return ai < bi
		}
	}
	return len(as) < len(bs)
}

// ManifestEntry records how one installer was built.
type ManifestEntry struct {
	Platform        string `json:"platform"`
	BuiltAt         string `json:"built_at"`
	Context         string `json:"context,omitempty"`
	FleetURL        string `json:"fleet_url"`
	Team            string `json:"team"`
	FleetDesktop    bool   `json:"fleet_desktop"`
	EnableScripts   bool   `json:"enable_scripts"`
	Debug           bool   `json:"debug"`
	UpdateURL       string `json:"update_url"`
	OrbitChannel    string `json:"orbit_channel"`
	DesktopChannel  string `json:"desktop_channel"`
	OsquerydChannel string `json:"osqueryd_channel"`
	FleetctlVersion string `json:"fleetctl_version"`
	ExtraFlags      string `json:"extra_flags,omitempty"`
}

// Manifest is a packages folder's manifest.json: one entry per file name.
type Manifest struct {
	Files map[string]ManifestEntry `json:"files"`
}

// UpdateManifest merges entries for the jobs whose output file was written at
// or after since into dir/manifest.json, keeping entries for other files.
// Entries for files that no longer exist are dropped. Returns how many jobs
// produced a file.
func UpdateManifest(dir string, jobs []Job, since time.Time, entry ManifestEntry) (int, error) {
	path := filepath.Join(dir, ManifestName)
	m := Manifest{Files: map[string]ManifestEntry{}}
	if b, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(b, &m) // a corrupt manifest is rebuilt from scratch
		if m.Files == nil {
			m.Files = map[string]ManifestEntry{}
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return 0, err
	}
	built := 0
	for _, j := range jobs {
		info, err := os.Stat(j.OutFile)
		if err != nil || info.ModTime().Before(since) {
			continue
		}
		e := entry
		e.Platform = j.Key
		m.Files[filepath.Base(j.OutFile)] = e
		built++
	}
	for name := range m.Files {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			delete(m.Files, name)
		}
	}
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return built, err
	}
	return built, WriteFileAtomic(path, b, 0o644)
}

// WriteFileAtomic writes via a temp file and rename, so readers (and a bash
// already running an older copy of a script) never see a half-written file.
func WriteFileAtomic(path string, b []byte, perm os.FileMode) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name()) // no-op after a successful rename
	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmp.Name(), perm); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

// ValidateArgs rejects resolved fleetctl arguments that build.sh would split:
// it reads them newline-separated, and values like the enroll secret or the
// scanned server_url come from the server rather than the user.
func ValidateArgs(args []string) error {
	for _, a := range args {
		if strings.ContainsAny(a, "\r\n") {
			flag, _, _ := strings.Cut(a, "=")
			return fmt.Errorf("%s value contains a line break", flag)
		}
	}
	return nil
}

// ValidateConfig rejects values that can't be passed through build.sh safely:
// args travel newline-separated, so a newline would inject an extra flag.
func ValidateConfig(cfg settings.InstallerOptions, target settings.InstallerTarget) error {
	for name, v := range map[string]string{
		"Fleet URL":       target.FleetURL,
		"orbit channel":   cfg.OrbitChannel,
		"desktop channel": cfg.DesktopChannel,
		"osquery channel": cfg.OsquerydChannel,
	} {
		if strings.ContainsAny(v, "\r\n\t") {
			return fmt.Errorf("%s contains a line break or tab", name)
		}
	}
	return nil
}

// OutputFile is one installer found in a packages folder, with how it was
// built when manifest.json has an entry for it (nil for files Hangar didn't
// build).
type OutputFile struct {
	Name    string         `json:"name"`
	Path    string         `json:"path"`
	Size    int64          `json:"size"`
	ModTime int64          `json:"mod_time_ms"`
	Build   *ManifestEntry `json:"build"`
}

// ListOutputs lists the installers (pkg/msi/deb/rpm/pkg.tar.zst) in dir,
// newest first. A missing dir is an empty list.
func ListOutputs(dir string) ([]OutputFile, error) {
	var m Manifest
	if b, err := os.ReadFile(filepath.Join(dir, ManifestName)); err == nil {
		_ = json.Unmarshal(b, &m) // a corrupt manifest just means no build details
	}
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return []OutputFile{}, nil
	}
	if err != nil {
		return nil, err
	}
	out := []OutputFile{}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if InstallerExt(e.Name()) == "" {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		f := OutputFile{Name: e.Name(), Path: filepath.Join(dir, e.Name()), Size: info.Size(), ModTime: info.ModTime().UnixMilli()}
		if entry, ok := m.Files[e.Name()]; ok {
			f.Build = &entry
		}
		out = append(out, f)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].ModTime != out[j].ModTime {
			return out[i].ModTime > out[j].ModTime
		}
		return out[i].Name < out[j].Name
	})
	return out, nil
}
