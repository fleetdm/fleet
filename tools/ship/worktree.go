package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// -----------------------------------------------------------------------------
// Registry helpers — operate on a *Config so callers can SaveConfig after.
// -----------------------------------------------------------------------------

// FindWorktreeByPath returns a pointer to the entry matching path, or nil.
// Path comparison is via filepath.Clean.
func FindWorktreeByPath(cfg *Config, path string) *WorktreeEntry {
	want := filepath.Clean(path)
	for i := range cfg.Worktrees {
		if filepath.Clean(cfg.Worktrees[i].Path) == want {
			return &cfg.Worktrees[i]
		}
	}
	return nil
}

// FindWorktreeByName returns a pointer to the entry matching name, or nil.
func FindWorktreeByName(cfg *Config, name string) *WorktreeEntry {
	for i := range cfg.Worktrees {
		if cfg.Worktrees[i].Name == name {
			return &cfg.Worktrees[i]
		}
	}
	return nil
}

// UpsertWorktree appends or updates an entry by path. Returns the entry that
// ended up in the registry.
func UpsertWorktree(cfg *Config, entry WorktreeEntry) WorktreeEntry {
	if existing := FindWorktreeByPath(cfg, entry.Path); existing != nil {
		// Refresh branch if known; preserve user-set name.
		if entry.Branch != "" {
			existing.Branch = entry.Branch
		}
		return *existing
	}
	cfg.Worktrees = append(cfg.Worktrees, entry)
	return entry
}

// RemoveWorktree drops an entry by name. Returns true if a matching entry
// was found and removed.
func RemoveWorktree(cfg *Config, name string) bool {
	for i := range cfg.Worktrees {
		if cfg.Worktrees[i].Name == name {
			cfg.Worktrees = append(cfg.Worktrees[:i], cfg.Worktrees[i+1:]...)
			if cfg.ActiveWorktree == name {
				cfg.ActiveWorktree = ""
			}
			return true
		}
	}
	return false
}

// PruneDeadWorktrees removes registry entries whose Path no longer exists
// on disk. Useful at startup so a manually-deleted worktree directory
// doesn't show up in the switcher.
func PruneDeadWorktrees(cfg *Config) (removed []string) {
	live := cfg.Worktrees[:0]
	for _, w := range cfg.Worktrees {
		info, err := os.Stat(w.Path)
		if err != nil || !info.IsDir() {
			removed = append(removed, w.Name)
			continue
		}
		live = append(live, w)
	}
	cfg.Worktrees = live
	if cfg.ActiveWorktree != "" && FindWorktreeByName(cfg, cfg.ActiveWorktree) == nil {
		cfg.ActiveWorktree = ""
	}
	return removed
}

// -----------------------------------------------------------------------------
// Path defaulting
// -----------------------------------------------------------------------------

// DefaultWorktreePath turns a branch name into a sensible directory path,
// sibling of the launching worktree. Slashes in branch names become dashes
// (`feature/x` → `fleet-feature-x`), so the resulting path is filesystem-safe.
func DefaultWorktreePath(launchRoot, branch string) string {
	parent := filepath.Dir(launchRoot)
	suffix := strings.ReplaceAll(branch, "/", "-")
	suffix = strings.TrimPrefix(suffix, "-") // tolerate "/foo"
	return filepath.Join(parent, "fleet-"+suffix)
}

// DefaultWorktreeName is the friendly label for a registry entry: the
// directory's basename (e.g. "fleet", "fleet-vpp-fix").
func DefaultWorktreeName(path string) string {
	return filepath.Base(filepath.Clean(path))
}

// -----------------------------------------------------------------------------
// Git wrappers
// -----------------------------------------------------------------------------

// gitWorktreeAdd creates a new branch off `main` AND a new worktree directory
// holding it. Equivalent to:
//
//	git worktree add -b <branch> <path> main
//
// Run from launchRoot — git resolves the .git directory by walking up.
func gitWorktreeAdd(ctx context.Context, launchRoot, branch, path string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "-C", launchRoot,
		"worktree", "add", "-b", branch, path, "main")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("git worktree add: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return string(out), nil
}

// gitWorktreeRemove tears down a worktree. Refuses if the worktree has
// uncommitted changes unless force is true.
func gitWorktreeRemove(ctx context.Context, launchRoot, path string, force bool) (string, error) {
	args := []string{"-C", launchRoot, "worktree", "remove"}
	if force {
		args = append(args, "--force")
	}
	args = append(args, path)
	out, err := exec.CommandContext(ctx, "git", args...).CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("git worktree remove: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return string(out), nil
}

// gitWorktreeListEntry is one parsed row from `git worktree list --porcelain`.
type gitWorktreeListEntry struct {
	Path     string
	Branch   string // refs/heads/foo → "foo"; empty if detached
	Detached bool
}

// gitWorktreeListAll runs `git worktree list --porcelain` and parses the
// blank-line-separated blocks into a slice. Used for prune sweeps and for
// surfacing branches the user created outside of ship.
func gitWorktreeListAll(ctx context.Context, launchRoot string) ([]gitWorktreeListEntry, error) {
	cmd := exec.CommandContext(ctx, "git", "-C", launchRoot, "worktree", "list", "--porcelain")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git worktree list: %w", err)
	}

	var entries []gitWorktreeListEntry
	cur := gitWorktreeListEntry{}
	flush := func() {
		if cur.Path != "" {
			entries = append(entries, cur)
		}
		cur = gitWorktreeListEntry{}
	}

	sc := bufio.NewScanner(strings.NewReader(string(out)))
	for sc.Scan() {
		line := sc.Text()
		if line == "" {
			flush()
			continue
		}
		switch {
		case strings.HasPrefix(line, "worktree "):
			cur.Path = strings.TrimPrefix(line, "worktree ")
		case strings.HasPrefix(line, "branch "):
			cur.Branch = strings.TrimPrefix(line, "branch refs/heads/")
		case line == "detached":
			cur.Detached = true
		}
	}
	flush()
	return entries, nil
}

// branchExistsLocally checks `git show-ref` to see if a branch already
// exists. Used to validate the new-worktree form before letting `git
// worktree add -b` fail with a less obvious error.
func branchExistsLocally(ctx context.Context, launchRoot, branch string) bool {
	if strings.TrimSpace(branch) == "" {
		return false
	}
	err := exec.CommandContext(ctx, "git", "-C", launchRoot,
		"show-ref", "--verify", "--quiet", "refs/heads/"+branch).Run()
	return err == nil
}

// pathExists is a convenience for the form's path-collision check.
func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// validateNewWorktreeForm checks the form's two fields and returns a
// human-readable error explaining what's wrong, or nil if everything's
// fine.
func validateNewWorktreeForm(ctx context.Context, launchRoot, branch, path string) error {
	branch = strings.TrimSpace(branch)
	path = strings.TrimSpace(path)
	if branch == "" {
		return errors.New("branch name is required")
	}
	if path == "" {
		return errors.New("path is required")
	}
	if pathExists(path) {
		return fmt.Errorf("%s already exists — pick a different path", path)
	}
	if branchExistsLocally(ctx, launchRoot, branch) {
		return fmt.Errorf("branch %q already exists locally", branch)
	}
	return nil
}
