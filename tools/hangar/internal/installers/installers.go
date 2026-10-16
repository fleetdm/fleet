// Package installers builds fleetd installers (fleetctl package) for the
// Fleet servers Hangar runs, named so they can be told apart at a glance and
// shared with test machines. It covers the pure parts — platform list, file
// naming, the fleetctl arguments, matching a server to a fleetctl context,
// reading channels from a TUF repo, the manifest — plus a small Fleet API
// client (fleetapi.go) for scanning a server's fleets and enroll secrets.
//
// The builds themselves run build.sh (embedded) through the process engine,
// one process per server, so output streams to a log channel like the TUF tab.
package installers

import (
	_ "embed"
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

	"github.com/fleetdm/fleet/tools/hangar/internal/settings"
)

// ProcPrefix / LogChannelPrefix namespace a server's build in the process
// engine: "installers:s1" logs to "installers-s1".
const (
	ProcPrefix       = "installers:"
	LogChannelPrefix = "installers-"
	ManifestName     = "manifest.json"
	// DefaultDirName is the default output folder, under ~/Documents.
	DefaultDirName = "fleetd-installers"
)

// ProcID is the process-engine id of serverID's build.
func ProcID(serverID string) string { return ProcPrefix + serverID }

// LogChannel is the log channel of serverID's build.
func LogChannel(serverID string) string { return LogChannelPrefix + serverID }

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

// ServerSlug is the server's name as used in folder and file names, falling
// back to its id ("s1") when the name has nothing usable.
func ServerSlug(srv settings.ServerProfile) string {
	if s := Slug(srv.Name); s != "" {
		return s
	}
	return Slug(srv.ID)
}

// channelSuffix describes non-stable channels for the file name: "" when all
// three are stable, "-edge" when every non-stable one is edge, and otherwise
// one "-<component>-<channel>" per non-stable component (e.g. a version pin
// from a local TUF repo: "-orbit-1.61.0").
func channelSuffix(cfg settings.InstallerServerConfig) string {
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
//	fleetd-<server>-<os>[-<arch>][-tuf][-<channels>][-desktop].<ext>
//
// e.g. fleetd-rc-linux-arm64-desktop.deb. Only settings that differ from the
// defaults add a part, so a default rebuild overwrites the same file (and its
// share link keeps working).
func FileName(serverSlug string, p Platform, cfg settings.InstallerServerConfig) string {
	var b strings.Builder
	b.WriteString("fleetd-" + serverSlug + "-" + p.OS)
	if p.Arch != "" {
		b.WriteString("-" + p.Arch)
	}
	if cfg.UpdateSource == settings.UpdateSourceLocalTUF {
		b.WriteString("-tuf")
	}
	b.WriteString(channelSuffix(cfg))
	if cfg.FleetDesktop {
		b.WriteString("-desktop")
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

// Jobs lists the runs for cfg's selected platforms, in Platforms order,
// writing into outDir. Unknown keys are ignored.
func Jobs(serverSlug, outDir string, cfg settings.InstallerServerConfig) []Job {
	selected := map[string]bool{}
	for _, k := range cfg.Platforms {
		selected[k] = true
	}
	var jobs []Job
	for _, p := range Platforms {
		if !selected[p.Key] {
			continue
		}
		jobs = append(jobs, Job{Key: p.Key, Type: p.Type, Arch: p.Arch, OutFile: filepath.Join(outDir, FileName(serverSlug, p, cfg))})
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
// per-server config.
type BuildInputs struct {
	FleetURL     string
	EnrollSecret string
	// UpdateURL / UpdateRoots are set only for a local TUF build.
	UpdateURL   string
	UpdateRoots string
}

// CommonArgs are the fleetctl package flags shared by every job of a build
// (build.sh adds --type, --arch and --outfile per job).
func CommonArgs(cfg settings.InstallerServerConfig, in BuildInputs) []string {
	args := []string{
		"--fleet-url=" + in.FleetURL,
		"--enroll-secret=" + in.EnrollSecret,
		"--disable-open-folder",
	}
	if cfg.FleetDesktop {
		args = append(args, "--fleet-desktop")
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
	return args
}

// MatchContext picks the fleetctl context for a server listening on port: the
// first context whose address is a loopback host on that port. "" if none.
func MatchContext(contexts map[string]string, order []string, port uint16) string {
	for _, name := range order {
		u, err := url.Parse(contexts[name])
		if err != nil || u.Host == "" {
			continue
		}
		switch u.Hostname() {
		case "localhost", "127.0.0.1", "::1":
		default:
			continue
		}
		if u.Port() == strconv.Itoa(int(port)) {
			return name
		}
	}
	return ""
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
	FleetURL        string `json:"fleet_url"`
	Team            string `json:"team"`
	FleetDesktop    bool   `json:"fleet_desktop"`
	Debug           bool   `json:"debug"`
	UpdateURL       string `json:"update_url"`
	OrbitChannel    string `json:"orbit_channel"`
	DesktopChannel  string `json:"desktop_channel"`
	OsquerydChannel string `json:"osqueryd_channel"`
	FleetctlVersion string `json:"fleetctl_version"`
}

// Manifest is a server folder's manifest.json: one entry per file name.
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

// ValidateConfig rejects values that can't be passed through build.sh safely:
// args travel newline-separated, so a newline would inject an extra flag.
func ValidateConfig(cfg settings.InstallerServerConfig) error {
	for name, v := range map[string]string{
		"Fleet URL":       cfg.FleetURL,
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

// OutputFile is one installer found in a server's output folder.
type OutputFile struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Size    int64  `json:"size"`
	ModTime int64  `json:"mod_time_ms"`
}

// ListOutputs lists the installers (pkg/msi/deb/rpm) in dir, newest first.
// A missing dir is an empty list.
func ListOutputs(dir string) ([]OutputFile, error) {
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
		switch filepath.Ext(e.Name()) {
		case ".pkg", ".msi", ".deb", ".rpm":
		default:
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		out = append(out, OutputFile{Name: e.Name(), Path: filepath.Join(dir, e.Name()), Size: info.Size(), ModTime: info.ModTime().UnixMilli()})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].ModTime != out[j].ModTime {
			return out[i].ModTime > out[j].ModTime
		}
		return out[i].Name < out[j].Name
	})
	return out, nil
}
