// Package apps detects installed AI desktop applications (and AI IDEs as apps)
// and whether they are running. Discovery is per-OS: macOS .app bundles +
// Info.plist, Windows uninstall registry keys, Linux .desktop files. Liveness
// comes from the shared process snapshot.
package apps

import (
	"strings"

	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/internal/proc"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/internal/fsutil"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/internal/homes"
)

// App is a detected AI desktop application.
type App struct {
	Name           string
	DisplayName    string
	Vendor         string
	Path           string
	BundleID       string
	Version        string
	PlatformSource string // applications | registry | appx | desktop-file
	Scope          string // system | user
	ServesLocalAPI int
	APIPort        int
	Running        int
	PID            int
	SHA256         string // hash of the app's primary executable (best-effort, diffable identity)

	execPath string // resolved executable file, set per-platform; hashed in Scan
}

type knownApp struct {
	name         string
	match        []string // tokens matched against display name / bundle id (lowercased)
	processNames []string // tokens matched against running process names (lowercased)
	apiPort      int      // local inference API port, if any
}

func knownApps() []knownApp {
	return []knownApp{
		{"claude-desktop", []string{"claude"}, []string{"claude"}, 0},
		{"chatgpt", []string{"chatgpt"}, []string{"chatgpt"}, 0},
		{"ollama", []string{"ollama"}, []string{"ollama"}, 11434},
		{"lm-studio", []string{"lm studio", "lmstudio", "lm-studio"}, []string{"lm studio", "lm-studio", "lmstudio"}, 1234},
		{"jan", []string{"jan"}, []string{"jan"}, 1337},
		{"gpt4all", []string{"gpt4all"}, []string{"gpt4all"}, 0},
		{"msty", []string{"msty"}, []string{"msty"}, 0},
		{"anythingllm", []string{"anythingllm", "anything llm"}, []string{"anythingllm"}, 0},
		{"comet", []string{"comet"}, []string{"comet"}, 0}, // Perplexity Comet (AI browser)
		{"dia", []string{"dia"}, []string{"dia"}, 0},       // Browser Company Dia (AI browser)
		{"perplexity", []string{"perplexity"}, []string{"perplexity"}, 0},
		{"cursor", []string{"cursor"}, []string{"cursor"}, 0},
		{"windsurf", []string{"windsurf"}, []string{"windsurf"}, 0},
		{"antigravity", []string{"antigravity"}, []string{"antigravity"}, 0},
		{"trae", []string{"trae"}, []string{"trae"}, 0},
		{"lm-studio-cli", []string{"lms"}, []string{"lms"}, 0},
		// Exact-id desktop apps. match stays empty so a display name that merely
		// contains the product word is not enough; see exactBundleIDs and
		// windowsPrograms. grok-bot is the Grok Bot desktop app, not the Grok CLI
		// (known agent "grok", discovered from ~/.grok).
		{"granola", nil, []string{"granola"}, 0},
		{"grok-bot", nil, []string{"grok bot"}, 0},
		{"meta-muse", nil, []string{"muse"}, 0},
	}
}

// exactBundleIDs are matched by equality against a whole identifying token
// (a macOS CFBundleIdentifier). They are not word-boundary tokens: a different
// bundle id, or an unrelated app named Museeks or MuseScore, must not match.
//
// Granola and Grok Bot ids are the ones Fleet's maintained-apps catalog
// queries. Meta Muse is macOS-only; com.meta.endo is CFBundleIdentifier from
// the official Muse DMG.
var exactBundleIDs = map[string][]string{
	"granola":   {"com.granola.app"},
	"grok-bot":  {"com.anysphere.sand"},
	"meta-muse": {"com.meta.endo"},
}

// windowsProgram is an uninstall entry matched the way the maintained-apps
// catalog matches it: DisplayName LIKE '<prefix> %' and Publisher equal to
// publisher. Meta Muse has no Windows release and is not listed here.
type windowsProgram struct {
	app       string
	prefix    string
	publisher string
}

var windowsPrograms = []windowsProgram{
	{"granola", "Granola", "Granola"},
	{"grok-bot", "Grok Bot", "SpaceXAI"},
}

// Scan returns all detected AI apps with running state filled in.
func Scan(homesList []homes.Home, snap *proc.Snapshot) []App {
	out := scanApps(homesList) // platform-specific (build-tagged)
	for i := range out {
		// Some discoveries carry no display name (e.g. a bare service binary);
		// fall back to the known-app key rather than reporting a nameless row.
		if out[i].DisplayName == "" {
			out[i].DisplayName = out[i].Name
		}
		k, ok := knownByName(out[i].Name)
		if !ok {
			continue
		}
		if k.apiPort > 0 {
			out[i].ServesLocalAPI = 1
			out[i].APIPort = k.apiPort
		}
		markRunning(&out[i], k, snap)
		// Hash the primary executable (best-effort): per-platform execPath first,
		// falling back to Path when it points directly at a file.
		if h := fsutil.SHA256(out[i].execPath); h != "" {
			out[i].SHA256 = h
		} else {
			out[i].SHA256 = fsutil.SHA256(out[i].Path)
		}
	}
	return out
}

// containsWordBoundary reports whether want appears in pn delimited by
// non-alphanumeric characters (or the string edges). Several match and
// processNames tokens are short, common substrings (e.g. "dia", "jan", "lms");
// a plain strings.Contains would falsely match unrelated programs like "NVIDIA
// Control Panel" or processes like "mediaanalysisd", reporting an AI app that
// was never installed or marking one Running with an unrelated PID. It scans
// every occurrence so a bounded match later in the string is still found.
func containsWordBoundary(pn, want string) bool {
	if want == "" {
		return false
	}
	for from := 0; from <= len(pn)-len(want); {
		i := strings.Index(pn[from:], want)
		if i < 0 {
			return false
		}
		i += from
		before := i == 0 || !isAlnum(pn[i-1])
		after := i+len(want) == len(pn) || !isAlnum(pn[i+len(want)])
		if before && after {
			return true
		}
		from = i + 1
	}
	return false
}

func isAlnum(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9')
}

// matchKnown finds the AI app a set of identifying strings belongs to. Tokens
// are matched at word boundaries, never mid-word: the joining NUL and the
// non-alphanumeric characters of display names, bundle ids, and file names all
// delimit words. Bundle ids in exactBundleIDs are the exception and match only
// when a whole token is equal to the id.
func matchKnown(tokens ...string) (knownApp, bool) {
	hay := strings.ToLower(strings.Join(tokens, "\x00"))
	for _, k := range knownApps() {
		if matchExactBundle(k.name, tokens) {
			return k, true
		}
		for _, m := range k.match {
			if containsWordBoundary(hay, m) {
				return k, true
			}
		}
	}
	return knownApp{}, false
}

func matchExactBundle(name string, tokens []string) bool {
	ids := exactBundleIDs[name]
	if len(ids) == 0 {
		return false
	}
	for _, tok := range tokens {
		for _, id := range ids {
			if tok == id {
				return true
			}
		}
	}
	return false
}

// matchWindowsProgram reports the desktop app whose catalog query is
// DisplayName LIKE '<prefix> %' AND Publisher = publisher. LIKE is
// ASCII-case-insensitive; publisher equality is case-sensitive, matching
// SQLite. A name that only contains the product word does not match.
func matchWindowsProgram(displayName, publisher string) (knownApp, bool) {
	if displayName == "" || publisher == "" {
		return knownApp{}, false
	}
	for _, w := range windowsPrograms {
		if publisher != w.publisher || !likeSpacePrefix(displayName, w.prefix) {
			continue
		}
		return knownByName(w.app)
	}
	return knownApp{}, false
}

// likeSpacePrefix reports whether name matches SQL LIKE 'prefix %'.
func likeSpacePrefix(name, prefix string) bool {
	if len(name) <= len(prefix) || name[len(prefix)] != ' ' {
		return false
	}
	return strings.EqualFold(name[:len(prefix)], prefix)
}

func knownByName(name string) (knownApp, bool) {
	for _, k := range knownApps() {
		if k.name == name {
			return k, true
		}
	}
	return knownApp{}, false
}

func markRunning(a *App, k knownApp, snap *proc.Snapshot) {
	if snap == nil {
		return
	}
	for pid, p := range snap.Procs {
		pn := strings.ToLower(p.Name)
		if pn == "" {
			continue
		}
		for _, want := range k.processNames {
			if pn == want || containsWordBoundary(pn, want) {
				a.Running, a.PID = 1, pid
				return
			}
		}
	}
}
