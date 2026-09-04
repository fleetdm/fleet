package jarvis

import (
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// BranchState classifies a local branch by how recoverable it is if deleted.
type BranchState int

const (
	// BranchLocalOnly has no origin/<branch> ref and no gone-upstream marker: it
	// was never pushed, so deleting it loses the work.
	BranchLocalOnly BranchState = iota
	// BranchPushed is fully contained on origin/<branch> — deleting it is safe and
	// recoverable with a fetch.
	BranchPushed
	// BranchAhead has an origin/<branch> ref but local commits not yet pushed.
	BranchAhead
	// BranchGone had an upstream that is now gone (the PR merged and the remote
	// branch was deleted). Common after a web merge; the safe "pushed" sweep won't
	// catch it, so it needs an explicit delete.
	BranchGone
)

// Label is the short tag shown next to a branch in the cleanup view.
func (s BranchState) Label() string {
	switch s {
	case BranchPushed:
		return "pushed"
	case BranchAhead:
		return "ahead"
	case BranchGone:
		return "gone"
	default:
		return "local"
	}
}

// BranchRow is one deletable local branch in one clone.
type BranchRow struct {
	Repo      string      // clone folder base name (e.g. "fleet", "fleet-plan")
	Path      string      // absolute clone path
	Branch    string      // local branch name
	State     BranchState // push/merge state
	Ahead     int         // commits ahead of origin/<branch> (BranchAhead only)
	Current   bool        // the checked-out branch (git can never delete it)
	Protected bool        // current or main/master — never deleted, even in bulk
}

// ScanFleetBranches finds git repos matching glob under each base dir and lists
// their local branches with push state. When prune is set it runs
// `git fetch --prune origin` per repo first, so branches whose merged remote was
// deleted surface as gone rather than looking fully pushed. Best-effort:
// unreadable or non-git dirs are skipped.
func ScanFleetBranches(baseDirs []string, glob string, prune bool) []BranchRow {
	if glob == "" {
		glob = "fleet*"
	}
	seen := map[string]bool{}
	var rows []BranchRow
	for _, base := range baseDirs {
		base = expandHome(base)
		matches, _ := filepath.Glob(filepath.Join(base, glob))
		sort.Strings(matches)
		for _, dir := range matches {
			if seen[dir] || !isGitDir(dir) {
				continue
			}
			seen[dir] = true
			if prune {
				_ = exec.Command("git", "-C", dir, "fetch", "--prune", "--quiet", "origin").Run()
			}
			rows = append(rows, branchRowsForRepo(dir)...)
		}
	}
	return rows
}

// branchRowsForRepo lists one clone's local branches with their state, in git's
// listing order (alphabetical), current/main first-class but not reordered.
func branchRowsForRepo(dir string) []BranchRow {
	repo := filepath.Base(dir)
	current := gitCurrentBranch(dir)
	// One call per repo: branch name, its upstream, and the upstream track state
	// ([gone] once the remote branch is pruned), NUL-separated so names with
	// spaces can't confuse the split.
	out, err := exec.Command("git", "-C", dir, "for-each-ref",
		"--format=%(refname:short)%00%(upstream:track)",
		"refs/heads/").Output()
	if err != nil {
		return nil
	}
	var rows []BranchRow
	for _, line := range strings.Split(strings.TrimRight(string(out), "\n"), "\n") {
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\x00")
		branch := parts[0]
		track := ""
		if len(parts) > 1 {
			track = parts[1]
		}
		row := BranchRow{Repo: repo, Path: dir, Branch: branch}
		row.Current = branch == current
		row.Protected = row.Current || branch == "main" || branch == "master"
		switch {
		case strings.Contains(track, "gone"):
			row.State = BranchGone
		case gitRefExists(dir, "refs/remotes/origin/"+branch):
			ahead, err := gitAheadCount(dir, branch)
			switch {
			case err != nil:
				// Fail closed: an unverifiable count must not look fully pushed, or
				// the bulk "delete pushed" sweep could destroy unpushed commits.
				row.State = BranchAhead
			case ahead == 0:
				row.State = BranchPushed
			default:
				row.Ahead = ahead
				row.State = BranchAhead
			}
		default:
			row.State = BranchLocalOnly
		}
		rows = append(rows, row)
	}
	return rows
}

// DeleteBranch force-deletes a local branch (-D). Force is required because
// pushed and merged-then-deleted branches aren't "merged into HEAD" from git's
// point of view, so -d would refuse them. Never call this on a protected branch.
func DeleteBranch(dir, branch string) error {
	return exec.Command("git", "-C", dir, "branch", "-D", branch).Run()
}

func gitRefExists(dir, ref string) bool {
	return exec.Command("git", "-C", dir, "show-ref", "--verify", "--quiet", ref).Run() == nil
}

func gitAheadCount(dir, branch string) (int, error) {
	out, err := exec.Command("git", "-C", dir, "rev-list", "--count", "origin/"+branch+".."+branch).Output()
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(strings.TrimSpace(string(out)))
}
