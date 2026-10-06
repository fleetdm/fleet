package fsutil

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
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
	if err := os.WriteFile(ww, []byte("x"), 0o666); err != nil { //nolint:gosec // test fixture: intentionally world-writable to exercise WorldWritable detection
		t.Fatal(err)
	}
	if err := os.Chmod(ww, 0o666); err != nil { //nolint:gosec // test fixture: intentionally world-writable to exercise WorldWritable detection
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

func TestWalkHomeVisitsEachDirOnce(t *testing.T) {
	home := t.TempDir()
	leaf := filepath.Join(home, "src", "a", "b", "c")
	require.NoError(t, os.MkdirAll(leaf, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(leaf, "AGENTS.md"), []byte("#"), 0o600))
	for _, d := range []string{
		filepath.Join("src", "a", "b", "c", "d"),
		filepath.Join("Projects", "x"),
		filepath.Join("projects", "y"), // the same directory as Projects on a case-insensitive filesystem
		filepath.Join("Developer", "z"),
		filepath.Join("other", "y", "z", "w"),
		filepath.Join(".hidden", "q"),
		filepath.Join("node_modules", "p"),
	} {
		require.NoError(t, os.MkdirAll(filepath.Join(home, d), 0o755))
	}
	// WalkHome keeps only directories holding a probed name, so give every
	// directory one to see the whole walk.
	require.NoError(t, filepath.WalkDir(home, func(p string, e os.DirEntry, err error) error {
		if err == nil && e.IsDir() {
			err = os.WriteFile(filepath.Join(p, "probe.md"), nil, 0o600)
		}
		return err
	}))

	got := map[string]bool{}
	var leafDir *WalkedDir
	for _, d := range WalkHome(home, []string{"probe.md", "AGENTS.md", "d"}) {
		rel, err := filepath.Rel(home, d.Path)
		require.NoError(t, err)
		_, dup := got[rel]
		require.False(t, dup, "directory %q visited twice", rel)
		got[rel] = d.Project
		if d.Path == leaf {
			leafDir = &d
		}
	}

	for rel, project := range map[string]bool{
		".":                                 false,
		"other":                             false,
		filepath.Join("other", "y", "z"):    false,
		"src":                               true,
		filepath.Join("src", "a", "b", "c"): true,
		filepath.Join("Projects", "x"):      true,
		filepath.Join("Developer", "z"):     true,
		filepath.Join("projects", "y"):      true,
	} {
		gotProject, ok := got[rel]
		if !ok && rel == filepath.Join("projects", "y") {
			// Case-insensitive filesystem: reached as Projects/y instead.
			gotProject, ok = got[filepath.Join("Projects", "y")]
		}
		require.True(t, ok, "%q not visited; got %v", rel, got)
		require.Equal(t, project, gotProject, "Project flag for %q", rel)
	}
	for _, rel := range []string{
		filepath.Join("other", "y", "z", "w"),
		filepath.Join("src", "a", "b", "c", "d"),
		".hidden",
		"node_modules",
	} {
		require.NotContains(t, got, rel)
	}

	// Directories at the depth limit are listed too, so collectors can probe
	// them without a stat per file.
	require.NotNil(t, leafDir)
	require.True(t, leafDir.Exists("AGENTS.md"))
	require.True(t, leafDir.IsDir("d"))
}

// Only directories holding a probed name are kept, each with only the probed
// entries, so a large home costs memory in proportion to its matches.
func TestWalkHomeKeepsOnlyProbedDirectories(t *testing.T) {
	home := t.TempDir()
	for a := range 5 {
		for b := range 5 {
			d := filepath.Join(home, "src", fmt.Sprint(a), fmt.Sprint(b))
			require.NoError(t, os.MkdirAll(d, 0o755))
			for f := range 20 {
				require.NoError(t, os.WriteFile(filepath.Join(d, fmt.Sprintf("file%d.go", f)), nil, 0o600))
			}
		}
	}
	hit := filepath.Join(home, "src", "1", "2")
	require.NoError(t, os.WriteFile(filepath.Join(hit, "AGENTS.md"), nil, 0o600))
	cursorHit := filepath.Join(home, "src", "3", "4")
	require.NoError(t, os.MkdirAll(filepath.Join(cursorHit, ".cursor"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(cursorHit, ".cursor", "mcp.json"), nil, 0o600))

	got := WalkHome(home, []string{"AGENTS.md", filepath.Join(".cursor", "mcp.json")})
	var paths []string
	for _, d := range got {
		paths = append(paths, d.Path)
		require.LessOrEqual(t, len(d.entries), 2, "only probed entries are kept for %s", d.Path)
	}
	require.ElementsMatch(t, []string{hit, cursorHit}, paths)

	for _, d := range got {
		if d.Path == hit {
			require.True(t, d.Exists("AGENTS.md"))
			// A name that wasn't probed is checked on disk instead.
			require.True(t, d.Exists("file0.go"))
			require.False(t, d.Exists("missing.go"))
		} else {
			require.True(t, d.Exists(filepath.Join(".cursor", "mcp.json")))
			require.False(t, d.Exists("AGENTS.md"))
		}
	}
}

func TestSHA256CachesUnchangedFiles(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "agent")
	original := bytes.Repeat([]byte("a"), minCachedHashBytes)
	require.NoError(t, os.WriteFile(p, original, 0o600))
	mtime := time.Now().Add(-time.Hour).Truncate(time.Second)
	require.NoError(t, os.Chtimes(p, mtime, mtime))

	first := SHA256(p)
	require.NotEmpty(t, first)
	hits := hashCacheHits.Load()
	require.Equal(t, first, SHA256(p))
	require.Equal(t, hits+1, hashCacheHits.Load(), "unchanged file is served from the cache")

	// A new modification time is a new key.
	later := mtime.Add(time.Minute)
	require.NoError(t, os.Chtimes(p, later, later))
	require.Equal(t, first, SHA256(p))
	require.Equal(t, hits+1, hashCacheHits.Load())

	// A different file renamed into place with the same size and time is a new key too.
	other := filepath.Join(dir, "other")
	require.NoError(t, os.WriteFile(other, bytes.Repeat([]byte("b"), minCachedHashBytes), 0o600))
	require.NoError(t, os.Chtimes(other, later, later))
	require.NoError(t, os.Rename(other, p))
	require.NotEqual(t, first, SHA256(p))
}

// Rewriting a cached file in place and restoring its modification time must
// not keep the old hash. Only the kernel sets the change time, so it catches
// the rewrite; Windows doesn't expose it through os.FileInfo.
func TestSHA256CacheSeesInPlaceRewrite(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("change time isn't available on Windows")
	}
	p := filepath.Join(t.TempDir(), "agent")
	require.NoError(t, os.WriteFile(p, bytes.Repeat([]byte("a"), minCachedHashBytes), 0o600))
	mtime := time.Now().Add(-time.Hour).Truncate(time.Second)
	require.NoError(t, os.Chtimes(p, mtime, mtime))
	first := SHA256(p)
	require.NotEmpty(t, first)

	// Change times have coarse resolution on some filesystems.
	time.Sleep(20 * time.Millisecond)
	require.NoError(t, os.WriteFile(p, bytes.Repeat([]byte("b"), minCachedHashBytes), 0o600))
	require.NoError(t, os.Chtimes(p, mtime, mtime))
	require.NotEqual(t, first, SHA256(p))
}

func TestWalkedDirProbesFromListing(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a"), []byte("x"), 0o600))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "d"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "d", "f"), []byte("x"), 0o600))

	d, ok := ListDir(dir)
	require.True(t, ok)
	require.True(t, d.Exists("a"))
	require.False(t, d.Exists("d"), "a directory is not a file")
	require.False(t, d.Exists("missing"))
	require.True(t, d.IsDir("d"))
	require.False(t, d.IsDir("a"))
	require.True(t, d.Exists(filepath.Join("d", "f")))
	require.False(t, d.Exists(filepath.Join("d", "g")))
	require.False(t, d.Exists(filepath.Join("missing", "f")))
	require.False(t, d.Exists(filepath.Join("a", "f")), "a file is not a parent")

	_, ok = ListDir(filepath.Join(dir, "a"))
	require.False(t, ok, "a file is not a directory")

	if runtime.GOOS == "windows" {
		return
	}
	require.NoError(t, os.Symlink(filepath.Join(dir, "a"), filepath.Join(dir, "l")))
	require.NoError(t, os.Symlink(filepath.Join(dir, "d"), filepath.Join(dir, "ld")))
	d, ok = ListDir(dir)
	require.True(t, ok)
	require.False(t, d.Exists("l"), "a link is refused like fsutil.Exists")
	require.True(t, d.IsDir("ld"), "a link to a directory is followed like os.Stat")
	require.True(t, d.Exists(filepath.Join("ld", "f")))
}

func TestLinkTargetWithin(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("creating symlinks needs elevated rights on Windows")
	}
	home := t.TempDir()
	outside := t.TempDir()
	bin := filepath.Join(home, "bin")
	require.NoError(t, os.MkdirAll(bin, 0o755))
	installed := filepath.Join(home, "share", "tool", "1.2.3")
	require.NoError(t, os.MkdirAll(filepath.Dir(installed), 0o755))
	require.NoError(t, os.WriteFile(installed, []byte("x"), 0o600))
	outsideFile := filepath.Join(outside, "tool")
	require.NoError(t, os.WriteFile(outsideFile, []byte("x"), 0o600))
	plain := filepath.Join(bin, "plain")
	require.NoError(t, os.WriteFile(plain, []byte("x"), 0o600))
	// A folder inside the home that is a link to one outside it: paths through
	// it look contained but aren't.
	escape := filepath.Join(home, "escape")
	require.NoError(t, os.Symlink(outside, escape))
	require.NoError(t, os.Symlink(installed, filepath.Join(outside, "inner-link")))

	link := func(name, target string) string {
		p := filepath.Join(bin, name)
		require.NoError(t, os.Symlink(target, p))
		return p
	}
	rel, err := filepath.Rel(bin, installed)
	require.NoError(t, err)

	for _, tc := range []struct {
		name, link, want string
	}{
		{"absolute target", link("abs", installed), installed},
		{"relative target", link("rel", rel), installed},
		{"not a link", plain, ""},
		{"target outside", link("out", outsideFile), ""},
		{"climbs out with ..", link("climb", filepath.Join("..", "..", filepath.Base(outside), "tool")), ""},
		{"broken link", link("broken", filepath.Join(home, "share", "tool", "9.9.9")), ""},
		{"link to a directory", link("dir", filepath.Dir(installed)), ""},
		{"missing", filepath.Join(bin, "missing"), ""},
		{"target through a linked folder", link("via", filepath.Join(escape, "tool")), ""},
		{"link inside a linked folder", filepath.Join(escape, "inner-link"), ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := LinkTargetWithin(home, tc.link)
			require.Equal(t, tc.want != "", ok)
			require.Equal(t, tc.want, got)
		})
	}

	// A link that isn't itself inside dir is not read.
	outsideLink := filepath.Join(outside, "link")
	require.NoError(t, os.Symlink(installed, outsideLink))
	_, ok := LinkTargetWithin(home, outsideLink)
	require.False(t, ok)
}
