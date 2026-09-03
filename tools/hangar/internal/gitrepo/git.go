// Package gitrepo wraps the git CLI for branch listing, status, and
// checkout. Ported from src-tauri/src/git.rs. We shell out to git (rather
// than a Go git library) for exact behavioral parity — same config, hooks,
// and output the user sees on the command line.
package gitrepo

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/fleetdm/fleet/tools/hangar/internal/shellpath"
)

// unitSep separates fields in our git --format strings (an ASCII Unit
// Separator can't appear in branch names, subjects, or author names).
const unitSep = "\x1f"

// FileChange is one entry from `git status --porcelain`.
type FileChange struct {
	Status string `json:"status"`
	Path   string `json:"path"`
}

// CommitInfo summarizes a single commit.
type CommitInfo struct {
	SHA     string `json:"sha"`
	Subject string `json:"subject"`
	Author  string `json:"author"`
	TimeAgo string `json:"time_ago"`
}

// BranchStatus is the working-tree status of the current branch.
type BranchStatus struct {
	Branch     string       `json:"branch"`
	Clean      bool         `json:"clean"`
	Ahead      uint32       `json:"ahead"`
	Behind     uint32       `json:"behind"`
	Modified   []FileChange `json:"modified"`
	LastCommit *CommitInfo  `json:"last_commit"`
}

// Branch is one branch in the list view.
type Branch struct {
	Name       string      `json:"name"`
	IsCurrent  bool        `json:"is_current"`
	IsLocal    bool        `json:"is_local"`
	IsRemote   bool        `json:"is_remote"`
	LastCommit *CommitInfo `json:"last_commit"`
}

// Worktree is one entry from `git worktree list --porcelain`. Multi-server
// Hangar runs each server from its own worktree so they can build/run
// different branches simultaneously while sharing one .git.
type Worktree struct {
	Path     string  `json:"path"`
	Head     string  `json:"head"`   // commit SHA the worktree is at
	Branch   *string `json:"branch"` // short branch name; nil if detached/bare
	Detached bool    `json:"detached"`
	Bare     bool    `json:"bare"`
	Locked   bool    `json:"locked"`
	IsMain   bool    `json:"is_main"` // the primary (non-linked) worktree
}

// runGit runs `git -C repo <args>` with the login-shell PATH so git
// resolves in a Finder-launched app. On failure it returns git's stderr.
func runGit(repo string, args ...string) (string, error) {
	cmd := shellpath.Command("git", append([]string{"-C", repo}, args...)...)
	out, err := cmd.Output()
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			return "", errors.New(string(ee.Stderr))
		}
		return "", fmt.Errorf("failed to spawn git: %w", err)
	}
	return string(out), nil
}

// ---- pure parsers (the actual logic; tested directly) ----

// parsePorcelain parses `git status --porcelain` output into file changes
// and a "clean" flag. Clean = no tracked changes; untracked files ("??")
// don't count.
func parsePorcelain(raw string) ([]FileChange, bool) {
	var modified []FileChange
	for _, line := range strings.Split(raw, "\n") {
		if len(line) < 4 {
			continue
		}
		modified = append(modified, FileChange{
			Status: strings.TrimSpace(line[:2]),
			Path:   line[3:],
		})
	}
	clean := true
	for _, f := range modified {
		if f.Status != "??" {
			clean = false
			break
		}
	}
	return modified, clean
}

// parseAheadBehind parses `git rev-list --left-right --count HEAD...@{u}`.
func parseAheadBehind(raw string) (uint32, uint32) {
	parts := strings.Fields(raw)
	var ahead, behind uint32
	if len(parts) > 0 {
		if n, err := strconv.ParseUint(parts[0], 10, 32); err == nil {
			ahead = uint32(n)
		}
	}
	if len(parts) > 1 {
		if n, err := strconv.ParseUint(parts[1], 10, 32); err == nil {
			behind = uint32(n)
		}
	}
	return ahead, behind
}

// parseLastCommit parses a "%h\x1f%s\x1f%an\x1f%cr" line.
func parseLastCommit(raw string) *CommitInfo {
	parts := strings.SplitN(strings.TrimSpace(raw), unitSep, 4)
	if len(parts) != 4 {
		return nil
	}
	return &CommitInfo{SHA: parts[0], Subject: parts[1], Author: parts[2], TimeAgo: parts[3]}
}

// parseWorktrees parses `git worktree list --porcelain`. Entries are separated
// by blank lines; each is a set of `key value` lines (`worktree <path>`,
// `HEAD <sha>`, `branch refs/heads/<name>` | `detached`, plus optional `bare`
// / `locked`). The first entry is always the main worktree.
func parseWorktrees(raw string) []Worktree {
	var out []Worktree
	var cur *Worktree
	flush := func() {
		if cur != nil && cur.Path != "" {
			cur.IsMain = len(out) == 0
			out = append(out, *cur)
		}
		cur = nil
	}
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimRight(line, "\r")
		if line == "" {
			flush()
			continue
		}
		key, val, _ := strings.Cut(line, " ")
		switch key {
		case "worktree":
			cur = &Worktree{Path: val}
		case "HEAD":
			if cur != nil {
				cur.Head = val
			}
		case "branch":
			if cur != nil {
				name := strings.TrimPrefix(val, "refs/heads/")
				cur.Branch = &name
			}
		case "detached":
			if cur != nil {
				cur.Detached = true
			}
		case "bare":
			if cur != nil {
				cur.Bare = true
			}
		case "locked":
			if cur != nil {
				cur.Locked = true
			}
		}
	}
	flush()
	return out
}

// parseRCMinorKey extracts the minor-line key from an RC branch name:
// "rc-minor-fleet-v4.86.0" / "rc-patch-fleet-v4.86.3" -> "4.86".
func parseRCMinorKey(name string) (string, bool) {
	s, ok := strings.CutPrefix(name, "rc-minor-fleet-v")
	if !ok {
		s, ok = strings.CutPrefix(name, "rc-patch-fleet-v")
		if !ok {
			return "", false
		}
	}
	parts := strings.Split(s, ".")
	if len(parts) < 2 {
		return "", false
	}
	if _, err := strconv.ParseUint(parts[0], 10, 32); err != nil {
		return "", false
	}
	if _, err := strconv.ParseUint(parts[1], 10, 32); err != nil {
		return "", false
	}
	return parts[0] + "." + parts[1], true
}

// parseBranches parses for-each-ref output (6 fields per line) into a deduped
// branch list. When query is non-empty it's a name search: branches are
// filtered to substring (case-insensitive) matches and capped to limit, and
// the RC minor-line grouping is bypassed (you're looking for a specific
// branch, not browsing recent release lines). With an empty query the
// recency view is unchanged: RC groups by minor line and keeps the N
// most-recent lines; other filters truncate to limit.
func parseBranches(raw, current, query string, isRC bool, limit *uint32) []Branch {
	seen := map[string]bool{}
	var branches []Branch

	for _, line := range strings.Split(raw, "\n") {
		parts := strings.SplitN(line, unitSep, 6)
		if len(parts) != 6 {
			continue
		}
		name := parts[0]
		fullRef := parts[5]
		isLocal := strings.HasPrefix(fullRef, "refs/heads/")
		isRemote := strings.HasPrefix(fullRef, "refs/remotes/")

		if isRemote {
			rest, ok := strings.CutPrefix(name, "origin/")
			if !ok || rest == "HEAD" {
				continue
			}
			name = rest
		}
		if seen[name] {
			continue // first occurrence wins (local precedence)
		}
		seen[name] = true

		branches = append(branches, Branch{
			Name:       name,
			IsCurrent:  isLocal && name == current,
			IsLocal:    isLocal,
			IsRemote:   isRemote && !isLocal,
			LastCommit: &CommitInfo{SHA: parts[1], Subject: parts[2], Author: parts[3], TimeAgo: parts[4]},
		})
	}

	// Name search: filter across the full ref set (server-side, so matches
	// surface regardless of recency), then cap. Skips RC grouping.
	if q := strings.ToLower(strings.TrimSpace(query)); q != "" {
		var matched []Branch
		for _, b := range branches {
			if strings.Contains(strings.ToLower(b.Name), q) {
				matched = append(matched, b)
			}
		}
		if limit != nil && len(matched) > int(*limit) {
			matched = matched[:*limit]
		}
		return matched
	}

	if isRC {
		n := 10
		if limit != nil {
			n = int(*limit)
		}
		var order []string
		seenKeys := map[string]bool{}
		for _, b := range branches {
			if key, ok := parseRCMinorKey(b.Name); ok && !seenKeys[key] {
				seenKeys[key] = true
				order = append(order, key)
			}
		}
		if len(order) > n {
			order = order[:n]
		}
		kept := map[string]bool{}
		for _, k := range order {
			kept[k] = true
		}
		var filtered []Branch
		for _, b := range branches {
			if b.IsCurrent {
				filtered = append(filtered, b)
				continue
			}
			if key, ok := parseRCMinorKey(b.Name); ok && kept[key] {
				filtered = append(filtered, b)
			}
		}
		return filtered
	}

	if limit != nil && len(branches) > int(*limit) {
		branches = branches[:*limit]
	}
	return branches
}

// ---- commands ----

// BranchStatusFor returns the working-tree status of repo's current branch.
func BranchStatusFor(repo string) (BranchStatus, error) {
	if _, err := os.Stat(filepath.Join(repo, ".git")); err != nil {
		return BranchStatus{}, fmt.Errorf("not a git repo: %s", repo)
	}
	branchOut, err := runGit(repo, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return BranchStatus{}, err
	}
	porcelain, err := runGit(repo, "status", "--porcelain")
	if err != nil {
		return BranchStatus{}, err
	}
	modified, clean := parsePorcelain(porcelain)

	var ahead, behind uint32
	if ab, err := runGit(repo, "rev-list", "--left-right", "--count", "HEAD...@{upstream}"); err == nil {
		ahead, behind = parseAheadBehind(ab)
	}

	var last *CommitInfo
	if lc, err := readLastCommit(repo, "HEAD"); err == nil {
		last = lc
	}

	return BranchStatus{
		Branch:     strings.TrimSpace(branchOut),
		Clean:      clean,
		Ahead:      ahead,
		Behind:     behind,
		Modified:   modified,
		LastCommit: last,
	}, nil
}

func readLastCommit(repo, ref string) (*CommitInfo, error) {
	raw, err := runGit(repo, "log", "-1", "--format=%h"+unitSep+"%s"+unitSep+"%an"+unitSep+"%cr", ref)
	if err != nil {
		return nil, err
	}
	ci := parseLastCommit(raw)
	if ci == nil {
		return nil, errors.New("unexpected git log output")
	}
	return ci, nil
}

// ListBranches lists branches matching filter ("rc", "main", or all),
// capped by limit. When query is non-empty it's a case-insensitive name
// search across the full ref set for the filter (the recency cap is dropped
// so a stale branch still surfaces); matches are then capped by limit.
func ListBranches(repo, filter, query string, limit *uint32) ([]Branch, error) {
	cur, err := runGit(repo, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return nil, err
	}
	current := strings.TrimSpace(cur)

	var patterns []string
	switch filter {
	case "rc":
		patterns = []string{
			"refs/heads/rc-patch-fleet-v*", "refs/heads/rc-minor-fleet-v*",
			"refs/remotes/origin/rc-patch-fleet-v*", "refs/remotes/origin/rc-minor-fleet-v*",
		}
	case "main":
		patterns = []string{
			"refs/heads/main", "refs/heads/master",
			"refs/remotes/origin/main", "refs/remotes/origin/master",
		}
	default:
		patterns = []string{"refs/heads", "refs/remotes"}
	}

	format := "--format=%(refname:short)" + unitSep + "%(objectname:short)" + unitSep +
		"%(contents:subject)" + unitSep + "%(authorname)" + unitSep +
		"%(committerdate:relative)" + unitSep + "%(refname)"
	args := []string{"for-each-ref", "--sort=-committerdate", format}
	// Non-RC filters cap on the for-each-ref side; RC handles its "N minor
	// lines" semantics in parseBranches, so it fetches the full set. A name
	// search also fetches the full set so parseBranches can match across all
	// refs rather than only the most-recent slice.
	if filter != "rc" && query == "" && limit != nil {
		args = append(args, fmt.Sprintf("--count=%d", *limit*2))
	}
	args = append(args, patterns...)

	raw, err := runGit(repo, args...)
	if err != nil {
		return nil, err
	}
	// parseBranches preserves git's --sort=-committerdate order.
	return parseBranches(raw, current, query, filter == "rc", limit), nil
}

// Fetch runs `git fetch --all --prune`.
func Fetch(repo string) (string, error) { return runGit(repo, "fetch", "--all", "--prune") }

// Pull runs `git pull --ff-only`.
func Pull(repo string) (string, error) { return runGit(repo, "pull", "--ff-only") }

// Checkout runs `git checkout <branch>`.
func Checkout(repo, branch string) (string, error) { return runGit(repo, "checkout", branch) }

// StashAndCheckout stashes (including untracked) then checks out branch.
func StashAndCheckout(repo, branch string) (string, error) {
	if _, err := runGit(repo, "stash", "push", "-u", "-m", "fleet-hangar auto-stash"); err != nil {
		return "", err
	}
	return runGit(repo, "checkout", branch)
}

// DiscardAndCheckout discards all local changes then checks out branch.
func DiscardAndCheckout(repo, branch string) (string, error) {
	if _, err := runGit(repo, "checkout", "--", "."); err != nil {
		return "", err
	}
	if _, err := runGit(repo, "clean", "-fd"); err != nil {
		return "", err
	}
	return runGit(repo, "checkout", branch)
}

// ---- worktrees ----

// ListWorktrees returns every worktree linked to repo (including repo's own,
// which is reported first as IsMain). repo may be any worktree of the repo —
// they all share the same worktree list.
func ListWorktrees(repo string) ([]Worktree, error) {
	out, err := runGit(repo, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, err
	}
	return parseWorktrees(out), nil
}

// AddWorktree creates a new worktree at path checked out to ref (a branch,
// tag, or commit). Git's DWIM applies: a remote-only branch name creates a
// local tracking branch. path must not already exist. On success returns
// git's (often empty) stdout; on failure returns git's stderr.
func AddWorktree(repo, path, ref string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", errors.New("worktree path is required")
	}
	args := []string{"worktree", "add"}
	if strings.TrimSpace(ref) != "" {
		args = append(args, path, ref)
	} else {
		// No ref: git checks out a detached HEAD at the current commit and
		// warns; callers usually pass a ref.
		args = append(args, path)
	}
	return runGit(repo, args...)
}

// RemoveWorktree removes the worktree at path (`git worktree remove`). force
// allows removal even with uncommitted changes. It does NOT delete the branch.
func RemoveWorktree(repo, path string, force bool) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", errors.New("worktree path is required")
	}
	args := []string{"worktree", "remove"}
	if force {
		args = append(args, "--force")
	}
	args = append(args, path)
	return runGit(repo, args...)
}
