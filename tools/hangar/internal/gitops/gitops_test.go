package gitops

import (
	"os"
	"path/filepath"
	"testing"
)

func mkdir(t *testing.T, p string) {
	t.Helper()
	if err := os.MkdirAll(p, 0o755); err != nil {
		t.Fatal(err)
	}
}

func touch(t *testing.T, p string) {
	t.Helper()
	mkdir(t, filepath.Dir(p))
	if err := os.WriteFile(p, []byte("x: 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestListReposSingleMode(t *testing.T) {
	root := t.TempDir()
	touch(t, filepath.Join(root, "default.yml"))
	touch(t, filepath.Join(root, "teams", "alpha.yml"))

	scan, err := ListRepos(root)
	if err != nil {
		t.Fatal(err)
	}
	if !scan.SingleRepoMode {
		t.Error("root with default.yml should be single-repo mode")
	}
	if len(scan.Repos) != 1 {
		t.Fatalf("want 1 repo, got %d", len(scan.Repos))
	}
	if len(scan.Repos[0].TeamFiles) != 1 || scan.Repos[0].TeamFiles[0].Name != "alpha.yml" {
		t.Errorf("team files = %+v", scan.Repos[0].TeamFiles)
	}
}

func TestListReposMultiMode(t *testing.T) {
	root := t.TempDir()
	// Two valid repos + one ignored (no default.yml).
	touch(t, filepath.Join(root, "repo-b", "default.yml"))
	touch(t, filepath.Join(root, "repo-a", "default.yml"))
	touch(t, filepath.Join(root, "repo-a", "teams", "t2.yml"))
	touch(t, filepath.Join(root, "repo-a", "teams", "t1.yml"))
	touch(t, filepath.Join(root, "repo-a", "fleets", "f1.yml"))
	touch(t, filepath.Join(root, "repo-a", "teams", ".hidden.yml")) // skipped
	touch(t, filepath.Join(root, "repo-a", "teams", "notes.txt"))   // skipped
	mkdir(t, filepath.Join(root, "not-a-repo"))

	scan, err := ListRepos(root)
	if err != nil {
		t.Fatal(err)
	}
	if scan.SingleRepoMode {
		t.Error("multi-repo dir should not be single-repo mode")
	}
	if len(scan.Repos) != 2 {
		t.Fatalf("want 2 repos, got %d: %+v", len(scan.Repos), scan.Repos)
	}
	// Sorted by name.
	if scan.Repos[0].Name != "repo-a" || scan.Repos[1].Name != "repo-b" {
		t.Errorf("repos not sorted: %s, %s", scan.Repos[0].Name, scan.Repos[1].Name)
	}
	// teams before fleets, each group sorted, hidden + non-yaml skipped.
	tf := scan.Repos[0].TeamFiles
	wantOrder := []string{"t1.yml", "t2.yml", "f1.yml"}
	if len(tf) != 3 {
		t.Fatalf("team files = %+v, want 3", tf)
	}
	for i, name := range wantOrder {
		if tf[i].Name != name {
			t.Errorf("team file[%d] = %q, want %q", i, tf[i].Name, name)
		}
	}
	if tf[0].Subdir != "teams" || tf[2].Subdir != "fleets" {
		t.Errorf("subdir tags wrong: %+v", tf)
	}
	// "not-a-repo" surfaced as ignored.
	if len(scan.Ignored) != 1 || scan.Ignored[0] != "not-a-repo" {
		t.Errorf("ignored = %v, want [not-a-repo]", scan.Ignored)
	}
}

func TestListReposNotADir(t *testing.T) {
	if _, err := ListRepos(filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Error("missing dir should error")
	}
}

func TestCheckTarget(t *testing.T) {
	root := t.TempDir()

	// Invalid names.
	for _, bad := range []string{"", "a/b", "..", ".", "~", `a\b`} {
		got, _ := CheckTarget(root, bad)
		if got.Reason == nil {
			t.Errorf("CheckTarget name %q should have a reason", bad)
		}
	}

	// Parent doesn't exist.
	got, _ := CheckTarget(filepath.Join(root, "nope"), "team")
	if got.Reason == nil {
		t.Error("missing parent should have a reason")
	}

	// Target available (doesn't exist yet).
	got, _ = CheckTarget(root, "fresh")
	if got.Exists || got.Reason != nil {
		t.Errorf("fresh target should be available: %+v", got)
	}
	if !got.Writable {
		t.Error("temp dir parent should be writable")
	}

	// Target is an existing file.
	touch(t, filepath.Join(root, "afile"))
	got, _ = CheckTarget(root, "afile")
	if !got.Exists || got.Reason == nil {
		t.Errorf("file target should report a reason: %+v", got)
	}

	// Target is an existing dir with files.
	mkdir(t, filepath.Join(root, "existing"))
	touch(t, filepath.Join(root, "existing", "a.yml"))
	touch(t, filepath.Join(root, "existing", "b.yml"))
	got, _ = CheckTarget(root, "existing")
	if !got.Exists || got.Reason != nil {
		t.Errorf("existing dir should be ok: %+v", got)
	}
	if got.FileCount != 2 {
		t.Errorf("file count = %d, want 2", got.FileCount)
	}
}
