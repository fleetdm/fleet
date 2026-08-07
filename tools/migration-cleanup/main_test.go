package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// setupTestRepo creates a minimal git repo with migration-like files and returns its path.
func setupTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	runGit(t, dir, "init", "-b", "main")
	runGit(t, dir, "config", "user.email", "test@test.com")
	runGit(t, dir, "config", "user.name", "Test")

	// Create migration directories
	tablesDir := filepath.Join(dir, "server/datastore/mysql/migrations/tables")
	dataDir := filepath.Join(dir, "server/datastore/mysql/migrations/data")
	os.MkdirAll(tablesDir, 0o755)
	os.MkdirAll(dataDir, 0o755)

	// Initial commit with a migration file
	writeFile(t, filepath.Join(tablesDir, "20250101000000_init.sql"), "CREATE TABLE t1;")
	writeFile(t, filepath.Join(dataDir, "20250101000001_seed.sql"), "INSERT INTO t1;")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "initial migrations")

	// Rename commit: move a table migration to a new version ID
	os.Remove(filepath.Join(tablesDir, "20250101000000_init.sql"))
	writeFile(t, filepath.Join(tablesDir, "20250201000000_init.sql"), "CREATE TABLE t1;")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "rename migration")

	// Tag the rename commit with an annotated tag
	renameSHA := lastCommitSHA(t, dir)
	runGit(t, dir, "tag", "-a", "rename-tag", "-m", "annotated tag", renameSHA)

	// Non-rename commit
	writeFile(t, filepath.Join(tablesDir, "20250301000000_new.sql"), "CREATE TABLE t2;")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "add new migration")

	return dir
}

func lastCommitSHA(t *testing.T, dir string) string {
	t.Helper()
	out, err := exec.Command("git", "-C", dir, "rev-parse", "HEAD").Output()
	if err != nil {
		t.Fatalf("git rev-parse HEAD: %v", err)
	}
	return strings.TrimSpace(string(out))
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writeFile: %v", err)
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
}

func TestCountModeFlags(t *testing.T) {
	tests := []struct {
		name string
		opts options
		want int
	}{
		{"no flags", options{}, 0},
		{"branch only", options{branch: "main"}, 1},
		{"sinceCommit only", options{sinceCommit: "abc"}, 1},
		{"commit only", options{commit: "abc"}, 1},
		{"branch + sinceCommit", options{branch: "main", sinceCommit: "abc"}, 2},
		{"branch + commit", options{branch: "main", commit: "abc"}, 2},
		{"sinceCommit + commit", options{sinceCommit: "abc", commit: "def"}, 2},
		{"all three", options{branch: "main", sinceCommit: "abc", commit: "def"}, 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := countModeFlags(tt.opts); got != tt.want {
				t.Errorf("countModeFlags() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestResolveRef(t *testing.T) {
	dir := setupTestRepo(t)

	// Resolve a commit SHA
	sha := lastCommitSHA(t, dir)
	resolved, err := resolveRef(dir, sha)
	if err != nil {
		t.Fatalf("resolveRef(full SHA): %v", err)
	}
	if resolved != sha {
		t.Errorf("resolveRef(full SHA) = %q, want %q", resolved, sha)
	}

	// Resolve HEAD
	resolved, err = resolveRef(dir, "HEAD")
	if err != nil {
		t.Fatalf("resolveRef(HEAD): %v", err)
	}
	if resolved != sha {
		t.Errorf("resolveRef(HEAD) = %q, want %q", resolved, sha)
	}

	// Resolve an annotated tag — must peel to the commit SHA (HEAD~1 is the rename commit)
	out, _ := exec.Command("git", "-C", dir, "rev-parse", "HEAD~1").Output()
	renameSHA := strings.TrimSpace(string(out))
	resolved, err = resolveRef(dir, "rename-tag")
	if err != nil {
		t.Fatalf("resolveRef(annotated tag): %v", err)
	}
	if resolved != renameSHA {
		t.Errorf("resolveRef(annotated tag) = %q, want %q", resolved, renameSHA)
	}

	// Invalid ref
	_, err = resolveRef(dir, "nonexistent-ref-xyz")
	if err == nil {
		t.Fatal("resolveRef(invalid ref) expected error, got nil")
	}
	if !strings.Contains(err.Error(), "cannot resolve ref") {
		t.Errorf("resolveRef(invalid ref) error = %v, want 'cannot resolve ref'", err)
	}
}

func TestFindRenameCommitsRange(t *testing.T) {
	dir := setupTestRepo(t)

	// Get the SHA of the first commit (initial migrations)
	out, err := exec.Command("git", "-C", dir, "rev-parse", "HEAD~2").Output()
	if err != nil {
		t.Fatalf("git rev-parse HEAD~2: %v", err)
	}
	initialSHA := strings.TrimSpace(string(out))

	// findRenameCommits with range initialSHA..main should find the rename commit
	commits, err := findRenameCommits(dir, "main", initialSHA)
	if err != nil {
		t.Fatalf("findRenameCommits: %v", err)
	}
	if len(commits) != 1 {
		t.Errorf("findRenameCommits(range) = %d commits, want 1", len(commits))
	}

	// The rename commit should be the second commit (HEAD~1)
	out, _ = exec.Command("git", "-C", dir, "rev-parse", "HEAD~1").Output()
	expectedRenameSHA := strings.TrimSpace(string(out))
	if commits[0] != expectedRenameSHA {
		t.Errorf("findRenameCommits() = %q, want %q", commits[0], expectedRenameSHA)
	}
}

func TestExtractRenames(t *testing.T) {
	dir := setupTestRepo(t)

	// Get the rename commit SHA
	out, _ := exec.Command("git", "-C", dir, "rev-parse", "HEAD~1").Output()
	renameSHA := strings.TrimSpace(string(out))

	renames, err := extractRenames(dir, renameSHA)
	if err != nil {
		t.Fatalf("extractRenames(rename commit): %v", err)
	}
	if len(renames) != 1 {
		t.Errorf("extractRenames(rename commit) = %d renames, want 1", len(renames))
	}
	if len(renames) > 0 {
		if renames[0].oldVersionID != 20250101000000 {
			t.Errorf("rename oldVersionID = %d, want %d", renames[0].oldVersionID, 20250101000000)
		}
		if renames[0].newVersionID != 20250201000000 {
			t.Errorf("rename newVersionID = %d, want %d", renames[0].newVersionID, 20250201000000)
		}
	}

	// Non-rename commit should produce no renames
	out, _ = exec.Command("git", "-C", dir, "rev-parse", "HEAD").Output()
	nonRenameSHA := strings.TrimSpace(string(out))

	renames, err = extractRenames(dir, nonRenameSHA)
	if err != nil {
		t.Fatalf("extractRenames(non-rename commit): %v", err)
	}
	if len(renames) != 0 {
		t.Errorf("extractRenames(non-rename commit) = %d renames, want 0", len(renames))
	}
}

func TestExtractRenamesAnnotatedTag(t *testing.T) {
	dir := setupTestRepo(t)

	// resolveRef should peel the annotated tag to a commit SHA
	resolved, err := resolveRef(dir, "rename-tag")
	if err != nil {
		t.Fatalf("resolveRef(rename-tag): %v", err)
	}

	renames, err := extractRenames(dir, resolved)
	if err != nil {
		t.Fatalf("extractRenames(resolved tag): %v", err)
	}
	if len(renames) != 1 {
		t.Errorf("extractRenames(resolved tag) = %d renames, want 1", len(renames))
	}
}
