package evidence

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/internal/fsutil"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/internal/homes"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/internal/paths"
)

// harnessFrameworks are libraries/products that imply agent-harness category.
var harnessFrameworks = map[string]struct{}{
	"crewai": {}, "langgraph": {}, "autogen": {}, "hermes": {},
	"openclaw": {}, "openai-agents": {}, "semantic-kernel": {},
	"@langchain/langgraph": {}, "langchain": {},
}

// allFrameworkMarkers maps package name → short label.
var allFrameworkMarkers = map[string]string{
	"crewai":                    "crewai",
	"langgraph":                 "langgraph",
	"@langchain/langgraph":      "langgraph",
	"langchain":                 "langchain",
	"langchain-core":            "langchain",
	"pyautogen":                 "autogen",
	"autogen":                   "autogen",
	"autogen-agentchat":         "autogen",
	"openai-agents":             "openai-agents",
	"semantic-kernel":           "semantic-kernel",
	"hermes":                    "hermes",
	"openclaw":                  "openclaw",
	"@anthropic-ai/claude-code": "claude-code",
	"@google/gemini-cli":        "gemini-cli",
	"@openai/codex":             "codex",
	"aider-chat":                "aider",
	"opencode-ai":               "opencode",
}

func scanFrameworks(h homes.Home, dirs []fsutil.WalkedDir) []Framework {
	var out []Framework
	seen := map[string]struct{}{}

	add := func(name, path, source string) {
		key := name + "|" + path
		if _, dup := seen[key]; dup {
			return
		}
		seen[key] = struct{}{}
		_, harness := harnessFrameworks[name]
		out = append(out, Framework{
			UID:      h.UID,
			Username: h.Username,
			Name:     name,
			Path:     path,
			Source:   source,
			Harness:  harness,
		})
	}

	// Global node_modules package dirs (names only).
	for _, nm := range paths.NodeModulesDirs(h.Dir) {
		ents, err := os.ReadDir(nm)
		if err != nil {
			continue
		}
		for _, e := range ents {
			if !e.IsDir() {
				continue
			}
			// scoped packages
			if strings.HasPrefix(e.Name(), "@") {
				sub, err := os.ReadDir(filepath.Join(nm, e.Name()))
				if err != nil {
					continue
				}
				for _, s := range sub {
					pkg := e.Name() + "/" + s.Name()
					if label, ok := allFrameworkMarkers[pkg]; ok {
						add(label, filepath.Join(nm, e.Name(), s.Name()), "global-node")
					}
				}
				continue
			}
			if label, ok := allFrameworkMarkers[e.Name()]; ok {
				add(label, filepath.Join(nm, e.Name()), "global-node")
			}
		}
	}

	// pipx venvs
	pipx := filepath.Join(h.Dir, ".local", "pipx", "venvs")
	if ents, err := os.ReadDir(pipx); err == nil {
		for _, e := range ents {
			if !e.IsDir() {
				continue
			}
			if label, ok := allFrameworkMarkers[e.Name()]; ok {
				add(label, filepath.Join(pipx, e.Name()), "pipx")
			}
		}
	}

	// Project manifests in project directories.
	for _, d := range dirs {
		if !d.Project {
			continue
		}
		pj := filepath.Join(d.Path, "package.json")
		if d.Exists("package.json") {
			for _, label := range parsePackageJSONDeps(pj) {
				add(label, pj, "package.json")
			}
		}
		for _, f := range []struct{ name, source string }{
			{"pyproject.toml", "pyproject"},
			{"requirements.txt", "requirements"},
		} {
			p := filepath.Join(d.Path, f.name)
			if d.Exists(f.name) {
				for _, label := range parsePythonDependencyNames(p) {
					add(label, p, f.source)
				}
			}
		}
	}
	return out
}

func parsePackageJSONDeps(path string) []string {
	b, err := fsutil.ReadFileBounded(path)
	if err != nil {
		return nil
	}
	b = b[:min(len(b), 256<<10)]
	var m struct {
		Deps    map[string]string `json:"dependencies"`
		DevDeps map[string]string `json:"devDependencies"`
		Name    string            `json:"name"`
	}
	if err := json.Unmarshal(b, &m); err != nil {
		return nil
	}
	var labels []string
	consider := func(pkg string) {
		if label, ok := allFrameworkMarkers[pkg]; ok {
			labels = append(labels, label)
		}
	}
	for pkg := range m.Deps {
		consider(pkg)
	}
	for pkg := range m.DevDeps {
		consider(pkg)
	}
	consider(m.Name)
	return uniqueSorted(labels)
}

// parsePythonDependencyNames returns the framework labels for the Python
// packages a pyproject.toml or requirements.txt names. Every quoted string and
// every line is read as a PEP 508 requirement, whose name is the leading run of
// letters, digits, ".", "_" and "-" before any extras, version specifier or
// environment marker; a TOML key (Poetry's `crewai = "^0.80"`) is read the
// same way. This is a scan, not a TOML parser, so a quoted string that happens
// to be exactly a framework's name counts too.
func parsePythonDependencyNames(path string) []string {
	b, err := fsutil.ReadFileBounded(path)
	if err != nil {
		return nil
	}
	b = b[:min(len(b), 128<<10)]
	var labels []string
	consider := func(s string) {
		if label, ok := allFrameworkMarkers[normalizePythonName(requirementName(s))]; ok {
			labels = append(labels, label)
		}
	}
	for line := range strings.SplitSeq(string(b), "\n") {
		line, _, _ = strings.Cut(line, "#")
		consider(line)
		for _, q := range quotedStrings(line) {
			consider(q)
		}
	}
	return uniqueSorted(labels)
}

// uniqueSorted returns labels sorted and without repeats, so the result
// doesn't depend on map iteration order.
func uniqueSorted(labels []string) []string {
	slices.Sort(labels)
	return slices.Compact(labels)
}

// requirementName returns the distribution name at the start of a PEP 508
// requirement, or "" when s doesn't start with one.
func requirementName(s string) string {
	s = strings.TrimSpace(s)
	end := strings.IndexFunc(s, func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '.' || r == '_' || r == '-')
	})
	if end < 0 {
		return s
	}
	switch rest := strings.TrimSpace(s[end:]); {
	case rest == "", strings.ContainsRune("[;<>=~!,@", rune(rest[0])):
		return s[:end]
	default:
		// Followed by more words: prose, not a requirement.
		return ""
	}
}

// pythonNameSeparators is PEP 503's run of separators that normalizes to "-".
var pythonNameSeparators = regexp.MustCompile(`[-_.]+`)

// normalizePythonName applies the PEP 503 normalization, so "Semantic__Kernel"
// and "semantic-kernel" are the same package.
func normalizePythonName(s string) string {
	return pythonNameSeparators.ReplaceAllString(strings.ToLower(s), "-")
}

// quotedStrings returns the contents of each complete '...' or "..." string in
// line.
func quotedStrings(line string) []string {
	var out []string
	for {
		i := strings.IndexAny(line, `"'`)
		if i < 0 {
			return out
		}
		s, rest, ok := strings.Cut(line[i+1:], line[i:i+1])
		if !ok {
			return out
		}
		out = append(out, s)
		line = rest
	}
}
