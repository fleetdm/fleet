// Package browserext discovers AI browser extensions across Chromium- and
// Gecko-family browsers, for every user home on the host. It is disk-only (no
// process snapshot) and read-only: extension manifests/registries are parsed
// and the on-disk artifact is hashed, never executed. Install provenance is
// read from local browser state (Chromium Preferences / Gecko extensions.json),
// never by contacting a web store. Only AI-classified extensions are emitted.
package browserext

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/internal/fsutil"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/internal/homes"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/internal/paths"
)

// signedStateUnknown marks a Gecko addon whose signedState we could not read.
const signedStateUnknown = -99

// Extension is one discovered AI browser extension (a browser_extension row).
type Extension struct {
	UID, Username string
	Browser       string // chrome, edge, brave, arc, opera, vivaldi, chromium, comet, dia, firefox, zen, ...
	Engine        string // chromium | gecko
	Profile       string // Default, Profile 1, <gecko profile name>
	ID            string
	Name          string
	Version       string
	Path          string // manifest.json (chromium) or .xpi (gecko) — the hashed artifact
	Category      string
	Scope         string // user
	ManifestVer   int    // chromium manifest_version (0 = unknown)
	HostPerms     []string
	FromWebstore  int  // -1 unknown, 0 no, 1 yes (chromium)
	SignedState   int  // signedStateUnknown, or Gecko signedState (-2..2)
	Sideloaded    bool // set per-engine; feeds computeRisk
	SHA256        string
	RiskFlags     string
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func isDir(p string) bool {
	fi, err := os.Stat(p)
	if err != nil {
		return false
	}
	return fi.IsDir()
}

// browserRoot is one resolved browser profile-parent directory.
type browserRoot struct {
	label string
	dir   string
}

// browserSub holds the per-OS subpath (relative to the OS app-data base) for
// one browser, slash-separated. An empty subpath means "not present on this OS".
type browserSub struct {
	label           string
	mac, linux, win string
}

var chromiumSubs = []browserSub{
	{"chrome", "Google/Chrome", "google-chrome", "Google/Chrome/User Data"},
	{"chrome-beta", "Google/Chrome Beta", "google-chrome-beta", "Google/Chrome Beta/User Data"},
	{"edge", "Microsoft Edge", "microsoft-edge", "Microsoft/Edge/User Data"},
	{"brave", "BraveSoftware/Brave-Browser", "BraveSoftware/Brave-Browser", "BraveSoftware/Brave-Browser/User Data"},
	{"arc", "Arc/User Data", "", "Arc/User Data"},
	{"opera", "com.operasoftware.Opera", "opera", "Opera Software/Opera Stable"},
	{"vivaldi", "Vivaldi", "vivaldi", "Vivaldi/User Data"},
	{"chromium", "Chromium", "chromium", "Chromium/User Data"},
	{"comet", "Perplexity/Comet", "Perplexity/Comet", "Perplexity/Comet/User Data"},
	{"dia", "Dia/User Data", "", "Dia/User Data"},
}

var geckoSubs = []browserSub{
	{"firefox", "Firefox", "", "Mozilla/Firefox"}, // linux handled specially below
	{"zen", "zen", "", "zen"},
	{"librewolf", "librewolf", "", "librewolf"},
	{"waterfox", "Waterfox", "", "Waterfox"},
}

// chromiumBase returns the OS app-data base that Chromium profile roots hang
// off. Empty when the OS is unsupported.
func chromiumBase(r paths.Roots) string {
	switch runtime.GOOS {
	case "darwin":
		return r.MacAppSupport
	case "windows":
		return r.LocalAppData
	default:
		return r.XDGConfig
	}
}

func subForOS(s browserSub) string {
	switch runtime.GOOS {
	case "darwin":
		return s.mac
	case "windows":
		return s.win
	default:
		return s.linux
	}
}

func chromiumRoots(r paths.Roots) []browserRoot {
	base := chromiumBase(r)
	if base == "" {
		return nil
	}
	var out []browserRoot
	for _, s := range chromiumSubs {
		sub := subForOS(s)
		if sub == "" {
			continue
		}
		out = append(out, browserRoot{s.label, filepath.Join(base, filepath.FromSlash(sub))})
	}
	return out
}

// geckoRoots resolves Firefox-family profile-parent dirs. Linux uses dotfile
// roots under the home dir rather than the XDG base.
func geckoRoots(r paths.Roots) []browserRoot {
	var out []browserRoot
	if runtime.GOOS == "linux" {
		for _, s := range []struct{ label, dir string }{
			{"firefox", ".mozilla/firefox"},
			{"zen", ".zen"},
			{"librewolf", ".librewolf"},
			{"waterfox", ".waterfox"},
		} {
			out = append(out, browserRoot{s.label, filepath.Join(r.Home, filepath.FromSlash(s.dir))})
		}
		return out
	}
	var base string
	switch runtime.GOOS {
	case "darwin":
		base = r.MacAppSupport
	case "windows":
		base = r.AppData // Firefox uses Roaming on Windows
	default:
		return nil
	}
	// linux is handled above; here mac/win columns drive the path. Skip any
	// browser with no subpath on this OS (matches chromiumRoots' behavior).
	for _, s := range geckoSubs {
		sub := macOrWin(s)
		if sub == "" {
			continue
		}
		out = append(out, browserRoot{s.label, filepath.Join(base, filepath.FromSlash(sub))})
	}
	return out
}

// macOrWin returns the platform subpath for a Gecko browser (darwin/windows
// only; linux is handled separately in geckoRoots).
func macOrWin(s browserSub) string {
	if runtime.GOOS == "windows" {
		return s.win
	}
	return s.mac
}

type profDir struct {
	name string
	path string
}

// chromiumProfiles lists profile dirs under a Chromium User-Data root.
func chromiumProfiles(root string) []profDir {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil
	}
	var out []profDir
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		low := strings.ToLower(e.Name())
		if low == "system profile" || low == "guest profile" {
			continue
		}
		p := filepath.Join(root, e.Name())
		if fsutil.Exists(filepath.Join(p, "Preferences")) ||
			fsutil.Exists(filepath.Join(p, "Secure Preferences")) ||
			isDir(filepath.Join(p, "Extensions")) {
			out = append(out, profDir{e.Name(), p})
		}
	}
	return out
}

// Scan returns every AI browser extension under a home directory, across all
// Chromium- and Gecko-family browsers and their profiles.
func Scan(h homes.Home) []Extension {
	r := paths.For(h.Dir)
	var out []Extension
	for _, root := range chromiumRoots(r) {
		for _, prof := range chromiumProfiles(root.dir) {
			out = append(out, collectChromiumProfile(prof.path, root.label, prof.name, h)...)
		}
	}
	for _, root := range geckoRoots(r) {
		for _, prof := range geckoProfiles(root.dir, h.Dir) {
			out = append(out, collectGeckoProfile(prof.path, root.label, prof.name, h)...)
		}
	}
	return out
}
