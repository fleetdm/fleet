package jarvis

import (
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// CloneStatus describes one local clone of the target repo: where it is, what
// branch it's on, and whether its working tree is clean (free to branch in).
type CloneStatus struct {
	Path   string
	Branch string
	Clean  bool
}

// Free reports whether the clone is safe to start new work in: a clean tree on
// the default branch.
func (c CloneStatus) Free() bool {
	return c.Clean && (c.Branch == "main" || c.Branch == "master")
}

// DiscoverClones scans the configured base directories (one level deep, plus the
// base dir itself) for git clones whose origin remote matches repo ("owner/name"),
// returning them free-clones-first. Best-effort: unreadable dirs are skipped.
func DiscoverClones(baseDirs []string, repo string) []CloneStatus {
	target := strings.ToLower(repo)
	seen := map[string]bool{}
	var out []CloneStatus

	consider := func(dir string) {
		if seen[dir] {
			return
		}
		if !isGitDir(dir) {
			return
		}
		if !cloneMatchesRepo(dir, target) {
			return
		}
		seen[dir] = true
		out = append(out, CloneStatus{
			Path:   dir,
			Branch: gitCurrentBranch(dir),
			Clean:  gitClean(dir),
		})
	}

	for _, base := range baseDirs {
		base = expandHome(base)
		consider(base)
		entries, err := os.ReadDir(base)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() {
				consider(filepath.Join(base, e.Name()))
			}
		}
	}

	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Free() != out[j].Free() {
			return out[i].Free() // free clones first
		}
		return out[i].Path < out[j].Path
	})
	return out
}

func isGitDir(dir string) bool {
	info, err := os.Stat(filepath.Join(dir, ".git"))
	return err == nil && (info.IsDir() || info.Mode().IsRegular()) // dir, or worktree .git file
}

// cloneMatchesRepo reports whether the clone's origin or upstream remote points at
// target ("owner/name", lowercase). Checking upstream too supports fork workflows.
func cloneMatchesRepo(dir, target string) bool {
	for _, remote := range []string{"origin", "upstream"} {
		if r, ok := gitRemoteRepo(dir, remote); ok && strings.ToLower(r) == target {
			return true
		}
	}
	return false
}

// gitRemoteRepo returns the "owner/name" of a clone's named remote.
func gitRemoteRepo(dir, remote string) (string, bool) {
	out, err := exec.Command("git", "-C", dir, "remote", "get-url", remote).Output()
	if err != nil {
		return "", false
	}
	return normalizeRemote(strings.TrimSpace(string(out)))
}

// normalizeRemote reduces a git remote URL to "owner/name".
func normalizeRemote(url string) (string, bool) {
	url = strings.TrimSuffix(url, ".git")
	// SSH form: git@github.com:owner/name
	if i := strings.Index(url, ":"); strings.HasPrefix(url, "git@") && i >= 0 {
		return url[i+1:], url[i+1:] != ""
	}
	// HTTPS form: https://github.com/owner/name
	if i := strings.Index(url, "github.com/"); i >= 0 {
		p := url[i+len("github.com/"):]
		return p, p != ""
	}
	return "", false
}

func gitCurrentBranch(dir string) string {
	out, err := exec.Command("git", "-C", dir, "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func gitClean(dir string) bool {
	out, err := exec.Command("git", "-C", dir, "status", "--porcelain").Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) == ""
}
