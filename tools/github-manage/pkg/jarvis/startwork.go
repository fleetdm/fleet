package jarvis

import (
	"fmt"
	"os/exec"
	"strings"
)

// StartWork creates a fresh branch off the clone's default branch, then sets the
// issue's project Status to "In progress". Branch creation is the hard step; if
// the status write fails (e.g. the issue isn't on a known board) it's returned as
// a non-fatal warning so the branch/session still proceed.
func StartWork(issue, project int, clonePath, branch string) (statusSet, warn string, err error) {
	if err := createBranch(clonePath, branch); err != nil {
		return "", "", err
	}
	statusSet, serr := resolveAndSetStatus(issue, project, statusInProgress)
	if serr != nil {
		warn = "branch created; status not set: " + serr.Error()
	}
	return statusSet, warn, nil
}

// createBranch fetches origin and creates `branch` off the clone's default branch.
func createBranch(clonePath, branch string) error {
	if out, err := runGit(clonePath, "fetch", "origin"); err != nil {
		return fmt.Errorf("git fetch: %s", firstLine(out))
	}
	base := defaultBase(clonePath)
	if out, err := runGit(clonePath, "checkout", "-b", branch, base); err != nil {
		return fmt.Errorf("git checkout -b %s %s: %s", branch, base, firstLine(out))
	}
	return nil
}

// defaultBase returns the clone's default remote branch ref (origin/main), falling
// back to origin/main when it can't be determined.
func defaultBase(clonePath string) string {
	out, err := runGit(clonePath, "symbolic-ref", "refs/remotes/origin/HEAD")
	if err == nil {
		ref := strings.TrimSpace(out)
		if r := strings.TrimPrefix(ref, "refs/remotes/"); r != ref {
			return r
		}
	}
	return "origin/main"
}

// runGit runs a git command in clonePath and returns combined output.
func runGit(clonePath string, args ...string) (string, error) {
	full := append([]string{"-C", clonePath}, args...)
	out, err := exec.Command("git", full...).CombinedOutput()
	return string(out), err
}

// suggestBranch proposes a branch name from the issue: "<login>-<number>-<slug>".
func suggestBranch(login string, issue int, title string) string {
	slug := slugify(title, 40)
	prefix := ""
	if login != "" {
		prefix = strings.ToLower(login) + "-"
	}
	return strings.Trim(fmt.Sprintf("%s%d-%s", prefix, issue, slug), "-")
}

// slugify lowercases s, replaces runs of non-alphanumerics with a single hyphen,
// and truncates to maxLen (on a hyphen boundary where possible).
func slugify(s string, maxLen int) string {
	var b strings.Builder
	prevDash := false
	for _, r := range strings.ToLower(s) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			prevDash = false
		default:
			if !prevDash {
				b.WriteByte('-')
				prevDash = true
			}
		}
	}
	slug := strings.Trim(b.String(), "-")
	if len(slug) > maxLen {
		slug = slug[:maxLen]
		if i := strings.LastIndexByte(slug, '-'); i > 0 {
			slug = slug[:i]
		}
	}
	return slug
}
