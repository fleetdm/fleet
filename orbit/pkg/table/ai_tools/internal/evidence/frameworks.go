package evidence

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/internal/fsutil"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/internal/homes"
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

// inSet reports whether a key is present in a string set.
func inSet(s map[string]struct{}, k string) bool {
	_, ok := s[k]
	return ok
}

func scanFrameworks(h homes.Home) []Framework {
	var out []Framework
	seen := map[string]struct{}{}

	add := func(name, path, source string) {
		key := name + "|" + path
		if _, dup := seen[key]; dup {
			return
		}
		seen[key] = struct{}{}
		out = append(out, Framework{
			UID:      h.UID,
			Username: h.Username,
			Name:     name,
			Path:     path,
			Source:   source,
			Harness:  inSet(harnessFrameworks, name),
		})
	}

	// Global node_modules package dirs (names only).
	for _, nm := range nodeModuleRoots(h.Dir) {
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

	// Project package.json / pyproject under known project roots (depth-capped).
	for _, root := range projectRoots(h.Dir) {
		fsutil.WalkBounded(root, 3, func(dir string) {
			pj := filepath.Join(dir, "package.json")
			if fsutil.Exists(pj) {
				for _, label := range parsePackageJSONDeps(pj) {
					add(label, pj, "package.json")
				}
			}
			py := filepath.Join(dir, "pyproject.toml")
			if fsutil.Exists(py) {
				for _, label := range parsePyprojectNames(py) {
					add(label, py, "pyproject")
				}
			}
		})
	}
	return out
}

func nodeModuleRoots(home string) []string {
	dirs := []string{
		filepath.Join(home, ".npm-global", "lib", "node_modules"),
		filepath.Join(home, ".bun", "install", "global", "node_modules"),
		"/usr/local/lib/node_modules",
		"/opt/homebrew/lib/node_modules",
	}
	if matches, _ := filepath.Glob(filepath.Join(home, ".nvm", "versions", "node", "*", "lib", "node_modules")); matches != nil {
		dirs = append(dirs, matches...)
	}
	var out []string
	for _, d := range dirs {
		if fi, err := os.Stat(d); err == nil && fi.IsDir() {
			out = append(out, d)
		}
	}
	return out
}

func parsePackageJSONDeps(path string) []string {
	b, err := fsutil.ReadFileBounded(path)
	if err != nil {
		return nil
	}
	if len(b) > 256<<10 {
		b = b[:256<<10]
	}
	var m struct {
		Deps    map[string]string `json:"dependencies"`
		DevDeps map[string]string `json:"devDependencies"`
		Name    string            `json:"name"`
	}
	if err := json.Unmarshal(b, &m); err != nil {
		return nil
	}
	var labels []string
	seen := map[string]struct{}{}
	consider := func(pkg string) {
		if label, ok := allFrameworkMarkers[pkg]; ok && !inSet(seen, label) {
			seen[label] = struct{}{}
			labels = append(labels, label)
		}
	}
	for pkg := range m.Deps {
		consider(pkg)
	}
	for pkg := range m.DevDeps {
		consider(pkg)
	}
	if label, ok := allFrameworkMarkers[m.Name]; ok && !inSet(seen, label) {
		labels = append(labels, label)
	}
	return labels
}

func parsePyprojectNames(path string) []string {
	b, err := fsutil.ReadFileBounded(path)
	if err != nil {
		return nil
	}
	if len(b) > 128<<10 {
		b = b[:128<<10]
	}
	s := strings.ToLower(string(b))
	var labels []string
	seen := map[string]struct{}{}
	for pkg, label := range allFrameworkMarkers {
		if strings.Contains(pkg, "/") {
			continue // npm scoped
		}
		// rough: dependency name appears in file
		if strings.Contains(s, `"`+pkg+`"`) || strings.Contains(s, `'`+pkg+`'`) ||
			strings.Contains(s, pkg+" ") || strings.Contains(s, pkg+"\n") {
			if _, dup := seen[label]; !dup {
				seen[label] = struct{}{}
				labels = append(labels, label)
			}
		}
	}
	return labels
}
