package db

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestEnsureBackupsDirWritesGitignore(t *testing.T) {
	repo := t.TempDir()
	dir, err := EnsureBackupsDir(repo)
	if err != nil {
		t.Fatal(err)
	}
	if dir != filepath.Join(repo, "db-backups") {
		t.Errorf("dir = %q", dir)
	}
	gi, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if err != nil {
		t.Fatalf("gitignore not written: %v", err)
	}
	body := string(gi)
	if !contains(body, "*") || !contains(body, "!.gitignore") {
		t.Errorf("gitignore body unexpected:\n%s", body)
	}
}

func TestListBackups(t *testing.T) {
	repo := t.TempDir()
	dir, _ := EnsureBackupsDir(repo)

	// Two dumps with distinct mtimes; a non-backup file that must be ignored.
	older := filepath.Join(dir, "old.sql.gz")
	newer := filepath.Join(dir, "new.sql.gz")
	write(t, older, "x")
	write(t, newer, "y")
	write(t, filepath.Join(dir, "notes.txt"), "ignore me")
	os.Chtimes(older, time.Unix(1000, 0), time.Unix(1000, 0))
	os.Chtimes(newer, time.Unix(2000, 0), time.Unix(2000, 0))

	// Sidecar metadata for the newer one.
	branch := "main"
	if err := SaveBackupMeta(newer, &branch, nil, 12345); err != nil {
		t.Fatal(err)
	}

	list, err := ListBackups(repo)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("got %d entries, want 2 (non-.sql.gz ignored): %+v", len(list), list)
	}
	// Newest first.
	if list[0].Name != "new.sql.gz" {
		t.Errorf("first entry = %q, want new.sql.gz (newest first)", list[0].Name)
	}
	if list[0].Branch == nil || *list[0].Branch != "main" {
		t.Errorf("branch metadata not attached: %v", list[0].Branch)
	}
	if list[0].CreatedAtMS == nil || *list[0].CreatedAtMS != 12345 {
		t.Errorf("created_at not attached: %v", list[0].CreatedAtMS)
	}
	if list[1].Branch != nil {
		t.Errorf("entry without sidecar should have nil branch, got %v", list[1].Branch)
	}
}

func TestListBackupsMissingDir(t *testing.T) {
	list, err := ListBackups(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 0 {
		t.Errorf("missing dir should list 0, got %d", len(list))
	}
}

func TestSaveBackupMetaTrimsEmptyToNil(t *testing.T) {
	repo := t.TempDir()
	dir, _ := EnsureBackupsDir(repo)
	dump := filepath.Join(dir, "b.sql.gz")
	write(t, dump, "x")

	blank := "   "
	note := "  a real note "
	if err := SaveBackupMeta(dump, &blank, &note, 1); err != nil {
		t.Fatal(err)
	}
	list, _ := ListBackups(repo)
	e := list[0]
	if e.Branch != nil {
		t.Errorf("blank branch should be nil, got %v", *e.Branch)
	}
	if e.Note == nil || *e.Note != "a real note" {
		t.Errorf("note should be trimmed, got %v", e.Note)
	}
}

func TestDeleteBackup(t *testing.T) {
	repo := t.TempDir()
	dir, _ := EnsureBackupsDir(repo)
	dump := filepath.Join(dir, "b.sql.gz")
	write(t, dump, "x")
	if err := SaveBackupMeta(dump, nil, nil, 1); err != nil {
		t.Fatal(err)
	}

	if err := DeleteBackup(repo, dump); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dump); !os.IsNotExist(err) {
		t.Error("dump not deleted")
	}
	if _, err := os.Stat(metaPathFor(dump)); !os.IsNotExist(err) {
		t.Error("sidecar not deleted")
	}
}

func TestDeleteBackupRefusals(t *testing.T) {
	repo := t.TempDir()
	EnsureBackupsDir(repo)

	// Outside the backups dir.
	outside := filepath.Join(repo, "elsewhere.sql.gz")
	write(t, outside, "x")
	if err := DeleteBackup(repo, outside); err == nil {
		t.Error("should refuse to delete outside db-backups")
	}
	// Sibling dir with a confusing prefix must not be treated as inside.
	sib := filepath.Join(repo, "db-backups-evil", "x.sql.gz")
	write(t, sib, "x")
	if err := DeleteBackup(repo, sib); err == nil {
		t.Error("component-wise prefix should reject db-backups-evil")
	}
	// Non-backup extension inside the dir.
	bad := filepath.Join(repo, "db-backups", "notes.txt")
	write(t, bad, "x")
	if err := DeleteBackup(repo, bad); err == nil {
		t.Error("should refuse to delete non-.sql.gz file")
	}
	// ".." traversal that resolves outside the dir but whose leading
	// components match it must be rejected (path is cleaned first).
	evil := filepath.Join(repo, "db-backups-evil", "x.sql.gz")
	write(t, evil, "x")
	traversal := filepath.Join(repo, "db-backups", "..", "db-backups-evil", "x.sql.gz")
	if err := DeleteBackup(repo, traversal); err == nil {
		t.Error("should refuse to delete via .. traversal out of db-backups")
	}
	if _, err := os.Stat(evil); err != nil {
		t.Error("traversal target should not have been deleted")
	}
}

func TestCheckBackupName(t *testing.T) {
	repo := t.TempDir()
	dir, _ := EnsureBackupsDir(repo)

	// Existing dump so the "exists" flag can be exercised.
	write(t, filepath.Join(dir, "taken.sql.gz"), "x")

	ok, err := CheckBackupName(repo, "my-backup_01")
	if err != nil {
		t.Fatal(err)
	}
	if ok.FinalName != "my-backup_01.sql.gz" {
		t.Errorf("final name = %q", ok.FinalName)
	}
	if ok.RelativePath != "db-backups/my-backup_01.sql.gz" {
		t.Errorf("relative path = %q", ok.RelativePath)
	}
	if ok.Exists {
		t.Error("fresh name should not exist")
	}

	// User typed the extension — must not double it.
	got, _ := CheckBackupName(repo, "taken.sql.gz")
	if got.FinalName != "taken.sql.gz" {
		t.Errorf("final name = %q, want taken.sql.gz (no double ext)", got.FinalName)
	}
	if !got.Exists {
		t.Error("taken.sql.gz should report exists")
	}

	for _, bad := range []string{"", "   ", ".hidden", "has/slash", "..", "a b", "naughty;rm"} {
		if _, err := CheckBackupName(repo, bad); err == nil {
			t.Errorf("CheckBackupName(%q) should error", bad)
		}
	}
}

func TestSanitizeSegment(t *testing.T) {
	cases := map[string]string{
		"s2":            "s2",
		"my-server_01":  "my-server_01",
		"..":            "",       // dots dropped -> can't produce . or ..
		"../evil":       "evil",   // no separator survives -> single safe segment
		"a/b":           "ab",     // slash dropped -> can't escape the segment
		"na.me":         "name",   // dots dropped
		"  spaced  ":    "spaced", // spaces dropped
		"!@#$":          "",
		"UUID-abc-123":  "UUID-abc-123",
	}
	for in, want := range cases {
		if got := sanitizeSegment(in); got != want {
			t.Errorf("sanitizeSegment(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestServerBackupsDir(t *testing.T) {
	// Empty-after-sanitize ids are rejected before any filesystem access.
	for _, bad := range []string{"", "..", "!!!", "///"} {
		if _, err := ServerBackupsDir(bad); err == nil {
			t.Errorf("ServerBackupsDir(%q) should error", bad)
		}
	}
	// Happy path: lands under <data>/db-backups/<id>, with no traversal.
	dir, err := ServerBackupsDir("s2")
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(dir) != "s2" || filepath.Base(filepath.Dir(dir)) != "db-backups" {
		t.Errorf("dir = %q, want .../db-backups/s2", dir)
	}
	if contains(dir, "..") {
		t.Errorf("dir must not contain ..: %q", dir)
	}
}

// The central location addresses backups by directory (not repo). Verify the
// dir-based ops work on an arbitrary directory and return dir-relative names.
func TestBackupOpsInArbitraryDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "central", "s2")
	if _, err := EnsureDir(dir); err != nil {
		t.Fatal(err)
	}
	// EnsureDir must NOT write a .gitignore (app-data isn't a git repo).
	if _, err := os.Stat(filepath.Join(dir, ".gitignore")); !os.IsNotExist(err) {
		t.Error("EnsureDir should not write .gitignore")
	}

	chk, err := CheckBackupNameInDir(dir, "snap")
	if err != nil {
		t.Fatal(err)
	}
	if chk.FinalName != "snap.sql.gz" || chk.RelativePath != "snap.sql.gz" {
		t.Errorf("check = %+v, want bare filename as RelativePath", chk)
	}

	dump := filepath.Join(dir, chk.FinalName)
	write(t, dump, "x")
	list, err := ListBackupsInDir(dir)
	if err != nil || len(list) != 1 || list[0].Name != "snap.sql.gz" {
		t.Fatalf("ListBackupsInDir = %+v, err %v", list, err)
	}
	if err := DeleteBackupInDir(dir, dump); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dump); !os.IsNotExist(err) {
		t.Error("dump not deleted")
	}
	// Confinement still holds for the dir-based delete.
	outside := filepath.Join(t.TempDir(), "x.sql.gz")
	write(t, outside, "x")
	if err := DeleteBackupInDir(dir, outside); err == nil {
		t.Error("should refuse to delete outside the given dir")
	}
}

func write(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
