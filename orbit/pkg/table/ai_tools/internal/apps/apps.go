// Package apps detects installed AI desktop applications (and AI IDEs as apps)
// and whether they are running. Discovery is per-OS: macOS .app bundles +
// Info.plist, Windows uninstall registry keys, Linux .desktop files. Liveness
// comes from the shared process snapshot.
package apps

import (
	"strings"

	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/internal/fsutil"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/internal/homes"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/internal/proc"
)

// App is a detected AI desktop application.
type App struct {
	Name           string
	Vendor         string
	Path           string
	BundleID       string
	Version        string
	PlatformSource string // applications | registry | desktop-file
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
		{"jan", []string{"jan.app", "jan ", "/jan"}, []string{"jan"}, 1337},
		{"gpt4all", []string{"gpt4all"}, []string{"gpt4all"}, 0},
		{"msty", []string{"msty"}, []string{"msty"}, 0},
		{"anythingllm", []string{"anythingllm", "anything llm"}, []string{"anythingllm"}, 0},
		{"comet", []string{"comet.app", "comet "}, []string{"comet"}, 0}, // Perplexity Comet (AI browser)
		{"dia", []string{"dia.app", "dia "}, []string{"dia"}, 0},         // Browser Company Dia (AI browser)
		{"perplexity", []string{"perplexity"}, []string{"perplexity"}, 0},
		{"cursor", []string{"cursor"}, []string{"cursor"}, 0},
		{"windsurf", []string{"windsurf"}, []string{"windsurf"}, 0},
		{"antigravity", []string{"antigravity"}, []string{"antigravity"}, 0},
		{"trae", []string{"trae.app", "trae "}, []string{"trae"}, 0},
		{"lm-studio-cli", []string{"lms"}, []string{"lms"}, 0},
	}
}

// Scan returns all detected AI apps with running state filled in.
func Scan(homesList []homes.Home, snap *proc.Snapshot) []App {
	out := scanApps(homesList) // platform-specific (build-tagged)
	for i := range out {
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
// non-alphanumeric characters (or the string edges). Several processNames tokens
// are short, common substrings (e.g. "dia", "jan"); a plain strings.Contains
// would falsely match unrelated processes like "mediaanalysisd", marking an app
// Running with an unrelated PID. It scans every occurrence so a bounded match
// later in the string is still found.
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

// matchKnown finds the AI app a set of identifying strings belongs to.
func matchKnown(tokens ...string) (knownApp, bool) {
	hay := strings.ToLower(strings.Join(tokens, "\x00"))
	for _, k := range knownApps() {
		for _, m := range k.match {
			if strings.Contains(hay, m) {
				return k, true
			}
		}
	}
	return knownApp{}, false
}

// firstNonEmpty is used by scanApps on darwin and linux; the windows build has
// no caller, so it is exempt from the unused check there.
//
//nolint:unused
func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
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
