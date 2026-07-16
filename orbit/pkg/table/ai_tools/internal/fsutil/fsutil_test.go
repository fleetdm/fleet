package fsutil

import (
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"sort"
	"testing"
)

func TestSHA256(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "f.txt")
	if err := os.WriteFile(p, []byte("hello"), 0o600); err != nil {
		t.Fatal(err)
	}
	// echo -n hello | shasum -a 256
	const want = "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"
	if got := SHA256(p); got != want {
		t.Errorf("SHA256=%q want %q", got, want)
	}
	if got := SHA256(dir); got != "" {
		t.Errorf("SHA256(dir)=%q want empty", got)
	}
	if got := SHA256(filepath.Join(dir, "missing")); got != "" {
		t.Errorf("SHA256(missing)=%q want empty", got)
	}
}

func TestSHA256Bytes(t *testing.T) {
	const want = "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"
	if got := SHA256Bytes([]byte("hello")); got != want {
		t.Errorf("SHA256Bytes=%q want %q", got, want)
	}
}

func TestStatPerms(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX mode bits not meaningful on Windows")
	}
	dir := t.TempDir()

	priv := filepath.Join(dir, "priv")
	if err := os.WriteFile(priv, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if p := Stat(priv); !p.Known || p.WorldReadable || p.WorldWritable {
		t.Errorf("0600: %+v want known, not world readable/writable", p)
	}

	open := filepath.Join(dir, "open")
	if err := os.WriteFile(open, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if p := Stat(open); !p.WorldReadable || p.WorldWritable {
		t.Errorf("0644: %+v want world readable, not writable", p)
	}

	ww := filepath.Join(dir, "ww")
	if err := os.WriteFile(ww, []byte("x"), 0o666); err != nil { // #nosec G306 -- test fixture: intentionally world-writable to exercise WorldWritable detection
		t.Fatal(err)
	}
	if err := os.Chmod(ww, 0o666); err != nil { // #nosec G302 -- test fixture: intentionally world-writable to exercise WorldWritable detection
		t.Fatal(err)
	}
	if p := Stat(ww); !p.WorldWritable {
		t.Errorf("0666: %+v want world writable", p)
	}
}

func TestWalkBoundedAndExists(t *testing.T) {
	root := t.TempDir()
	// root/a/b/c (depth 3) and a skipped node_modules dir.
	deep := filepath.Join(root, "a", "b", "c")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "node_modules", "pkg"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".hidden", "x"), 0o755); err != nil {
		t.Fatal(err)
	}

	var visited []string
	WalkBounded(root, 2, func(dir string) {
		rel, _ := filepath.Rel(root, dir)
		visited = append(visited, rel)
	})
	sort.Strings(visited)

	has := func(p string) bool {
		return slices.Contains(visited, p)
	}
	if !has(".") || !has("a") || !has(filepath.Join("a", "b")) {
		t.Errorf("expected root/a/a-b visited, got %v", visited)
	}
	if has(filepath.Join("a", "b", "c")) {
		t.Errorf("depth cap breached: %v", visited)
	}
	// Dotted dirs are skipped by the walk; callers probe known dotted paths
	// (.cursor/.vscode/...) inside the visit callback instead.
	if has(".hidden") || has(filepath.Join(".hidden", "x")) {
		t.Errorf("dotted dir should be skipped by the walk: %v", visited)
	}
	if has("node_modules") {
		t.Errorf("node_modules should be skipped: %v", visited)
	}

	f := filepath.Join(root, "file")
	if err := os.WriteFile(f, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if !Exists(f) {
		t.Error("Exists(file) = false")
	}
	if Exists(root) {
		t.Error("Exists(dir) = true, want false")
	}
}
