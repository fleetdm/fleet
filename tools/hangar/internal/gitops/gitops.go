// Package gitops discovers GitOps repos under a configured directory and
// validates generate targets. Ported from src-tauri/src/gitops.rs. A
// "repo" is any directory containing default.yml.
package gitops

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/fleetdm/fleet/tools/hangar/internal/paths"
)

// File is a team/fleet YAML found inside a repo.
type File struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Size    uint64 `json:"size"`
	MtimeMS uint64 `json:"mtime_ms"`
	Subdir  string `json:"subdir"` // "teams" or "fleets"
}

// Repo is a discovered GitOps repo (one that has default.yml).
type Repo struct {
	Name           string `json:"name"`
	Path           string `json:"path"`
	HasDefault     bool   `json:"has_default"`
	DefaultPath    string `json:"default_path"`
	DefaultSize    uint64 `json:"default_size"`
	DefaultMtimeMS uint64 `json:"default_mtime_ms"`
	TeamFiles      []File `json:"team_files"`
}

// DirScan is the result of scanning the configured GitOps directory.
type DirScan struct {
	Root          string   `json:"root"`
	SingleRepoMode bool    `json:"single_repo_mode"`
	Repos         []Repo   `json:"repos"`
	Ignored       []string `json:"ignored"`
}

// TargetCheck validates a generate target subdirectory.
type TargetCheck struct {
	Path      string  `json:"path"`
	Exists    bool    `json:"exists"`
	FileCount uint32  `json:"file_count"`
	Writable  bool    `json:"writable"`
	Reason    *string `json:"reason"`
}

func mtimeMS(p string) uint64 {
	fi, err := os.Stat(p)
	if err != nil {
		return 0
	}
	return uint64(fi.ModTime().UnixMilli())
}

func fileSize(p string) uint64 {
	fi, err := os.Stat(p)
	if err != nil {
		return 0
	}
	return uint64(fi.Size())
}

// collectTeamYAMLs returns all non-hidden *.yml/*.yaml in <repo>/<subdir>,
// sorted by name. Empty when the subdir is absent.
func collectTeamYAMLs(repo, subdir string) []File {
	dir := filepath.Join(repo, subdir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []File
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		lower := strings.ToLower(name)
		if !strings.HasSuffix(lower, ".yml") && !strings.HasSuffix(lower, ".yaml") {
			continue
		}
		p := filepath.Join(dir, name)
		out = append(out, File{
			Name:    name,
			Path:    p,
			Size:    fileSize(p),
			MtimeMS: mtimeMS(p),
			Subdir:  subdir,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// buildRepo builds a Repo if repoPath contains default.yml, else nil.
// team_files lists teams first, then fleets (older convention reads first).
func buildRepo(repoPath string) *Repo {
	defaultYML := filepath.Join(repoPath, "default.yml")
	if fi, err := os.Stat(defaultYML); err != nil || !fi.Mode().IsRegular() {
		return nil
	}
	teamFiles := collectTeamYAMLs(repoPath, "teams")
	teamFiles = append(teamFiles, collectTeamYAMLs(repoPath, "fleets")...)
	return &Repo{
		Name:           filepath.Base(repoPath),
		Path:           repoPath,
		HasDefault:     true,
		DefaultPath:    defaultYML,
		DefaultSize:    fileSize(defaultYML),
		DefaultMtimeMS: mtimeMS(defaultYML),
		TeamFiles:      teamFiles,
	}
}

// ListRepos scans dir. If dir itself contains default.yml it's treated as a
// single repo; otherwise its direct child dirs are scanned, with those
// lacking default.yml surfaced in Ignored.
func ListRepos(dir string) (DirScan, error) {
	root := paths.Expand(dir)
	fi, err := os.Stat(root)
	if err != nil || !fi.IsDir() {
		return DirScan{}, fmt.Errorf("not a directory: %s", root)
	}

	// Single-repo mode: the configured dir IS the repo.
	if r := buildRepo(root); r != nil {
		return DirScan{
			Root:           root,
			SingleRepoMode: true,
			Repos:          []Repo{*r},
			Ignored:        []string{},
		}, nil
	}

	entries, err := os.ReadDir(root)
	if err != nil {
		return DirScan{}, err
	}
	repos := []Repo{}
	ignored := []string{}
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		p := filepath.Join(root, e.Name())
		if r := buildRepo(p); r != nil {
			repos = append(repos, *r)
		} else {
			ignored = append(ignored, e.Name())
		}
	}
	sort.Slice(repos, func(i, j int) bool { return repos[i].Name < repos[j].Name })
	sort.Strings(ignored)
	return DirScan{Root: root, SingleRepoMode: false, Repos: repos, Ignored: ignored}, nil
}

// countFiles counts regular files under path, capped at cap (so a huge tree
// doesn't stall — the UI only needs "N" vs "cap+").
func countFiles(path string, cap uint32) uint32 {
	var total uint32
	stack := []string{path}
	for len(stack) > 0 {
		if total >= cap {
			return cap
		}
		p := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		entries, err := os.ReadDir(p)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if total >= cap {
				return cap
			}
			ep := filepath.Join(p, e.Name())
			if e.IsDir() {
				stack = append(stack, ep)
			} else if e.Type().IsRegular() {
				total++
			}
		}
	}
	return total
}

// writable mirrors the Rust heuristic (!permissions.readonly()): any write
// bit set. Not an ownership-aware check — just a fast hint for the UI.
func writable(p string) bool {
	fi, err := os.Stat(p)
	if err != nil {
		return false
	}
	return fi.Mode().Perm()&0o222 != 0
}

// CheckTarget validates the subdirectory name the user wants to generate
// into. It creates nothing — just answers "is this a safe spot".
func CheckTarget(dir, name string) (TargetCheck, error) {
	root := paths.Expand(dir)
	reason := func(s string) *string { return &s }

	if name == "" || strings.ContainsAny(name, `/\`) || strings.Contains(name, "..") || name == "." || name == "~" {
		return TargetCheck{
			Path:   filepath.Join(root, name),
			Reason: reason("invalid subdirectory name"),
		}, nil
	}
	if fi, err := os.Stat(root); err != nil || !fi.IsDir() {
		return TargetCheck{
			Path:   filepath.Join(root, name),
			Reason: reason(fmt.Sprintf("parent directory does not exist: %s", root)),
		}, nil
	}

	target := filepath.Join(root, name)
	fi, err := os.Stat(target)
	if err != nil {
		// Parent exists, target doesn't — the "available" state. Check the
		// parent for writability.
		return TargetCheck{Path: target, Exists: false, Writable: writable(root)}, nil
	}
	if fi.Mode().IsRegular() {
		return TargetCheck{
			Path:   target,
			Exists: true,
			Reason: reason("target is a file, not a directory"),
		}, nil
	}
	return TargetCheck{
		Path:      target,
		Exists:    true,
		FileCount: countFiles(target, 200),
		Writable:  writable(target),
	}, nil
}
