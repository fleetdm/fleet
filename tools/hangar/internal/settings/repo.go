package settings

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/fleetdm/fleet/tools/hangar/internal/paths"
)

// RepoProbe is the result of validating a candidate Fleet clone.
type RepoProbe struct {
	Path   string  `json:"path"`
	Valid  bool    `json:"valid"`
	Reason *string `json:"reason"`
}

// devRoots are the well-known dev-root parent dirs we scan for clones. Each
// is searched at depth 2 as well (catches <parent>/<org>/fleet layouts like
// GitHub Desktop's ~/Documents/GitHub/fleetdm/fleet).
var devRoots = []string{
	"~/repositories", "~/repos", "~/code", "~/Code", "~/src",
	"~/Developer", "~/Documents/GitHub", "~/Projects", "~/projects",
	"~/work", "~/dev", "~/github", "~/git",
}

func invalid(path, reason string) RepoProbe {
	return RepoProbe{Path: path, Valid: false, Reason: &reason}
}

// probe validates a single path (after tilde expansion) as a Fleet clone:
// it must exist, contain a go.mod, and that go.mod's module must be
// github.com/fleetdm/fleet.
func probe(path string) RepoProbe {
	p := paths.Expand(path)
	if _, err := os.Stat(p); err != nil {
		return invalid(p, "path does not exist")
	}
	goMod := filepath.Join(p, "go.mod")
	if _, err := os.Stat(goMod); err != nil {
		return invalid(p, "no go.mod found")
	}
	contents, err := os.ReadFile(goMod)
	if err != nil {
		return invalid(p, "could not read go.mod: "+err.Error())
	}
	if !strings.Contains(string(contents), "github.com/fleetdm/fleet") {
		return invalid(p, "go.mod module is not github.com/fleetdm/fleet")
	}
	return RepoProbe{Path: p, Valid: true}
}

// ProbeOne validates the given path as a Fleet repo.
func ProbeOne(path string) RepoProbe { return probe(path) }

// DiscoverFleetRepos walks the well-known dev-root parents (plus ~/fleet)
// and returns every directory that looks like a Fleet clone, deduped by
// canonical path so a clone reached via a symlink appears once.
func DiscoverFleetRepos() []RepoProbe {
	home, _ := os.UserHomeDir()
	roots := make([]string, len(devRoots))
	for i, r := range devRoots {
		roots[i] = paths.Expand(r)
	}
	return discoverFleetRepos(home, roots)
}

func discoverFleetRepos(home string, roots []string) []RepoProbe {
	var results []RepoProbe
	seen := map[string]bool{}

	canonical := func(p string) string {
		if c, err := filepath.EvalSymlinks(p); err == nil {
			return c
		}
		return p
	}
	maybeAdd := func(path string) {
		pr := probe(path)
		if !pr.Valid {
			return
		}
		key := canonical(path)
		if !seen[key] {
			seen[key] = true
			results = append(results, pr)
		}
	}

	// ~/fleet as a one-off (we don't scan all of ~ — too noisy).
	if home != "" {
		maybeAdd(filepath.Join(home, "fleet"))
	}

	for _, root := range roots {
		entries, err := os.ReadDir(root)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
				continue
			}
			child := filepath.Join(root, entry.Name())
			// Depth-1 hit?
			if pr := probe(child); pr.Valid {
				key := canonical(child)
				if !seen[key] {
					seen[key] = true
					results = append(results, pr)
				}
				continue // don't descend into a known repo
			}
			// Not itself a repo (no go.mod) → treat as a potential org
			// folder and scan one level deeper.
			if _, err := os.Stat(filepath.Join(child, "go.mod")); err != nil {
				grand, err := os.ReadDir(child)
				if err != nil {
					continue
				}
				for _, g := range grand {
					if !g.IsDir() || strings.HasPrefix(g.Name(), ".") {
						continue
					}
					maybeAdd(filepath.Join(child, g.Name()))
				}
			}
		}
	}

	sort.Slice(results, func(i, j int) bool { return results[i].Path < results[j].Path })
	return results
}

// DetectFleetConfig returns the relative name (fleet.yml / fleet.yaml) of a
// serve config in the repo root, or "" if none exists. Relative because
// serve always runs with the repo as cwd.
func DetectFleetConfig(repo string) string {
	root := paths.Expand(repo)
	for _, name := range []string{"fleet.yml", "fleet.yaml"} {
		if fi, err := os.Stat(filepath.Join(root, name)); err == nil && fi.Mode().IsRegular() {
			return name
		}
	}
	return ""
}
