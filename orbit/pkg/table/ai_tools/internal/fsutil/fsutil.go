// Package fsutil holds small filesystem helpers shared across collectors:
// content hashing (a diffable integrity fingerprint) and permission inspection
// (used to flag world-readable secret-bearing files and world-writable
// instruction files) from POSIX mode bits or a Windows DACL.
//
// These never execute a discovered file — they only stat and read it — so they
// preserve the extension's no-exec security posture.
package fsutil

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

// maxHashBytes bounds how large a file we are willing to hash. Hashing streams
// in constant memory, so the limit caps I/O time per query, not memory. Files
// larger than this return an empty hash rather than a misleading prefix hash.
const maxHashBytes = 256 << 20 // 256 MiB

// maxReadFileBytes bounds how much of a file ReadFileBounded will read into
// memory. Every legitimate config/manifest we read is well under this; the cap
// stops a planted multi-gigabyte file from exhausting memory in the root daemon.
const maxReadFileBytes = 64 << 20 // 64 MiB

// OpenRegular opens path read-only for scanning, refusing anything that is not
// a regular file. Because this scanner runs as root over paths writable by
// unprivileged users, it must never follow a symlink (which could point at a
// root-only file) nor block on a FIFO/device. Callers that stream (rather than
// read the whole file) use this directly.
//
// Four guards, defending against a hostile local user racing the scanner:
//   - os.Lstat up front rejects symlinks and non-regular files (fast path);
//   - O_NOFOLLOW on the open (unix), so opening fails outright if the final
//     component is a symlink — the open never follows one;
//   - O_NONBLOCK on the open, so a file swapped for a FIFO in the Lstat→open
//     window still returns immediately instead of blocking the root daemon;
//   - a post-open fstat that re-checks IsRegular AND confirms (via os.SameFile)
//     that the opened file is the same inode Lstat saw. This closes the
//     stat→open TOCTOU race on platforms without O_NOFOLLOW: if the path was
//     swapped for another file after the Lstat, the fstat identity no longer
//     matches and the open is refused.
func OpenRegular(path string) (*os.File, error) {
	lfi, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !lfi.Mode().IsRegular() {
		return nil, os.ErrInvalid
	}
	f, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NONBLOCK|openNoFollow, 0) // #nosec G304 -- path discovered by a curated collector; opened non-following, non-blocking, regular-only, with a post-open identity re-check
	if err != nil {
		return nil, err
	}
	ffi, err := f.Stat()
	if err != nil || !ffi.Mode().IsRegular() || !os.SameFile(lfi, ffi) {
		_ = f.Close()
		return nil, os.ErrInvalid
	}
	return f, nil
}

// minCachedHashBytes is the size from which SHA256 results are cached. Agent
// binaries run to hundreds of MiB and are re-hashed on every hourly query;
// small config and instruction files are cheaper to re-read than to track.
const minCachedHashBytes = 1 << 20

type hashCacheEntry struct {
	fi    os.FileInfo
	ctime time.Time
	hash  string
}

// hashCache lives for the orbit process, so an unchanged binary is hashed once.
// Entries are keyed by path and are valid only while size, modification time,
// change time and file identity all match. The change time is what catches an
// in-place rewrite whose modification time was set back, since only the kernel
// sets it.
var hashCache sync.Map // path -> hashCacheEntry

// hashCacheHits counts hashes served from hashCache.
var hashCacheHits atomic.Int64

// SHA256 returns the lowercase hex SHA-256 of the file at path, or "" if the
// file can't be read, is not a regular file (directory, symlink, FIFO, device,
// socket), or exceeds maxHashBytes.
func SHA256(path string) string {
	if path == "" {
		return ""
	}
	fi, err := os.Lstat(path)
	if err != nil || !fi.Mode().IsRegular() || fi.Size() > maxHashBytes {
		return ""
	}
	f, err := OpenRegular(path)
	if err != nil {
		return ""
	}
	defer func() { _ = f.Close() }()

	// The opened file's own stat carries its identity (inode, or volume and file
	// index on Windows), so a cached entry can't be confused with a file renamed
	// into place later.
	ffi, err := f.Stat()
	if err != nil {
		return ""
	}
	cacheable := ffi.Size() >= minCachedHashBytes
	if cacheable {
		if v, ok := hashCache.Load(path); ok {
			e := v.(hashCacheEntry)
			if e.fi.Size() == ffi.Size() && e.fi.ModTime().Equal(ffi.ModTime()) &&
				e.ctime.Equal(changeTime(f, ffi)) && os.SameFile(e.fi, ffi) {
				hashCacheHits.Add(1)
				return e.hash
			}
		}
	}

	h := sha256.New()
	// LimitReader guards against a file that grows past the cap between stat and
	// read (e.g. an actively-written log) so we never stream unbounded.
	if _, err := io.Copy(h, io.LimitReader(f, maxHashBytes)); err != nil {
		return ""
	}
	sum := hex.EncodeToString(h.Sum(nil))
	if cacheable {
		hashCache.Store(path, hashCacheEntry{fi: ffi, ctime: changeTime(f, ffi), hash: sum})
	}
	return sum
}

// LinkTargetWithin returns the target of the symlink at link when both the
// link and its target are inside dir and the target is a regular file. It never
// opens anything through a link a user controls: it reads the link's text one
// level and Lstats the target for its type, both through an os.Root on dir, so
// a folder link on the way (~/.local/share -> /etc) can't lead outside dir. ok
// is false for anything else: not a link, a broken link, a link to a directory,
// or a target outside dir.
func LinkTargetWithin(dir, link string) (target string, ok bool) {
	linkRel, ok := relWithin(dir, link)
	if !ok {
		return "", false
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return "", false
	}
	defer func() { _ = root.Close() }()

	fi, err := root.Lstat(linkRel)
	if err != nil || fi.Mode()&os.ModeSymlink == 0 {
		return "", false
	}
	target, err = root.Readlink(linkRel)
	if err != nil {
		return "", false
	}
	if !filepath.IsAbs(target) {
		target = filepath.Join(filepath.Dir(link), target)
	}
	target = filepath.Clean(target)
	targetRel, ok := relWithin(dir, target)
	if !ok {
		return "", false
	}
	if tfi, err := root.Lstat(targetRel); err != nil || !tfi.Mode().IsRegular() {
		return "", false
	}
	return target, true
}

// relWithin returns p relative to dir when p is lexically inside dir.
func relWithin(dir, p string) (string, bool) {
	rel, err := filepath.Rel(dir, p)
	if err != nil || !filepath.IsLocal(rel) {
		return "", false
	}
	return rel, true
}

// PathWithin reports whether p is dir or lexically inside it, without
// resolving links in either.
func PathWithin(dir, p string) bool {
	dir, p = filepath.Clean(dir), filepath.Clean(p)
	return p == dir || strings.HasPrefix(p, dir+string(filepath.Separator))
}

// ReadFileBounded reads up to maxReadFileBytes of the regular file at path. It
// is the safe replacement for os.ReadFile in this package: it refuses symlinks
// and non-regular files and never blocks on a FIFO/device (see
// OpenRegular), so a hostile file planted in a scanned home directory
// cannot leak a root-only target or hang the root daemon.
//
// A file larger than maxReadFileBytes is silently truncated (partial content,
// nil error). Every caller parses the result as JSON/plist/TOML/XML, so a
// truncated blob simply fails to parse and the entry is dropped; a future
// caller that needs the whole file must not treat a bounded read as complete.
func ReadFileBounded(path string) ([]byte, error) {
	f, err := OpenRegular(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	return io.ReadAll(io.LimitReader(f, maxReadFileBytes))
}

// SHA256Bytes returns the lowercase hex SHA-256 of b (used for hashing
// synthesized strings such as a launch spec, not files).
func SHA256Bytes(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// Perm describes how widely a file is readable or writable. "World" means the
// POSIX group/other bits on macOS and Linux, and a DACL grant to a well-known
// everyone-style SID on Windows. Known is false when the posture could not be
// determined — an unreadable path, or a Windows security descriptor we could not
// read — so callers don't emit a risk signal they haven't actually established.
// A DACL that is read but contains an ACE type we don't decode still reports
// Known: true; skipping such an ACE can only lose a signal, never invent one.
type Perm struct {
	WorldReadable bool
	WorldWritable bool
	Known         bool
}

// Stat returns the permission posture of path. The per-platform reader lives in
// perm_unix.go and perm_windows.go.
func Stat(path string) Perm {
	return statPerm(path)
}

// Exists reports whether path is an existing regular file. It uses Lstat and
// refuses symlinks and other non-regular files, matching the read path: the
// root scanner must not treat a symlink to a root-only file as a scannable
// config, nor probe FIFOs/devices.
func Exists(path string) bool {
	fi, err := os.Lstat(path)
	if err != nil {
		return false
	}
	return fi.Mode().IsRegular()
}

// walkSkip are directory names never descended into during a bounded walk —
// large, machine-generated, or irrelevant to agentic-config discovery.
var walkSkip = map[string]struct{}{
	"node_modules": {}, ".git": {}, "vendor": {}, "library": {},
	".trash": {}, ".cache": {}, "dist": {}, "build": {},
	".venv": {}, "venv": {}, "target": {}, ".next": {},
}

// WalkBounded invokes visit(dir) for root and each descendant directory, capped
// at maxDepth levels and maxDirs total directories. Dotted directories
// (.cursor, .vscode, .github, ...) are neither visited nor descended, so
// callers probe known dotted paths from the visit callback.
func WalkBounded(root string, maxDepth int, visit func(dir string)) {
	walkBounded(root, maxDepth, func(dir string, _ []fs.DirEntry, _ int) bool {
		visit(dir)
		return true
	})
}

// walkBounded is WalkBounded with a visit callback that sees each directory's
// listing and depth and returns false to keep the walk out of its subtree.
// Every visited directory is read, including those at maxDepth, so callers
// test for files by name instead of probing each one with a stat.
func walkBounded(root string, maxDepth int, visit func(dir string, entries []fs.DirEntry, depth int) bool) {
	const maxDirs = 4000
	type item struct {
		dir   string
		depth int
	}
	stack := []item{{root, 0}}
	count := 0
	for len(stack) > 0 {
		it := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		count++
		if count > maxDirs {
			return
		}
		entries, _ := os.ReadDir(it.dir)
		if !visit(it.dir, entries, it.depth) || it.depth >= maxDepth {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			name := e.Name()
			if _, skip := walkSkip[strings.ToLower(name)]; strings.HasPrefix(name, ".") || skip {
				continue
			}
			stack = append(stack, item{filepath.Join(it.dir, name), it.depth + 1})
		}
	}
}

// projectRootNames are the conventional dev-project directories directly under
// a home, matched case-insensitively.
var projectRootNames = []string{
	"projects", "developer", "documents", "src", "code", "git", "dev", "workspace", "repos",
}

// WalkedDir is one directory visited by WalkHome, with the part of its listing
// collectors probe, so they test for well-known files by name instead of a stat
// per probe.
type WalkedDir struct {
	Path string
	// Project is true at or under a dev-project root (~/src, ~/Projects, ...)
	// and false for the home itself and its other subtrees.
	Project bool
	entries map[string]fs.FileMode // name -> type bits; 0 is a regular file
	// kept is the set of names entries was filtered to, or nil when entries is
	// the full listing. A name outside it is checked on disk.
	kept map[string]struct{}
}

// newWalkedDir builds a WalkedDir from a listing, keeping only the names in
// kept (all of them when kept is nil). ok is false when kept is set and none of
// its names is present.
func newWalkedDir(path string, entries []fs.DirEntry, kept map[string]struct{}) (WalkedDir, bool) {
	d := WalkedDir{Path: path, kept: kept}
	for _, e := range entries {
		if kept != nil {
			if _, ok := kept[e.Name()]; !ok {
				continue
			}
		}
		if d.entries == nil {
			d.entries = map[string]fs.FileMode{}
		}
		d.entries[e.Name()] = e.Type()
	}
	return d, kept == nil || len(d.entries) > 0
}

// ListDir reads one directory outside the walk (a tool home) into a WalkedDir.
// ok is false when it can't be read or isn't a directory.
func ListDir(path string) (WalkedDir, bool) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return WalkedDir{}, false
	}
	d, _ := newWalkedDir(path, entries, nil)
	return d, true
}

// Exists reports whether rel names a regular file under the directory, with
// the same symlink refusal as the package-level Exists. A single name is
// answered from the listing; a nested path is checked on disk only when its
// first element is listed as a directory or a link. A name the walk wasn't
// asked to keep is checked on disk.
func (d WalkedDir) Exists(rel string) bool {
	return d.probe(rel, fs.FileMode.IsRegular, Exists)
}

// IsDir reports whether rel names a directory under the directory, following a
// link as os.Stat does.
func (d WalkedDir) IsDir(rel string) bool {
	return d.probe(rel, fs.FileMode.IsDir, IsDir)
}

func (d WalkedDir) probe(rel string, want func(fs.FileMode) bool, onDisk func(string) bool) bool {
	first, _, nested := strings.Cut(rel, string(filepath.Separator))
	if d.kept != nil {
		if _, ok := d.kept[first]; !ok {
			return onDisk(filepath.Join(d.Path, rel))
		}
	}
	typ, ok := d.entries[first]
	if !ok {
		return false
	}
	isLink := typ&fs.ModeSymlink != 0
	if !nested && !isLink {
		return want(typ)
	}
	if nested && !typ.IsDir() && !isLink {
		return false
	}
	return onDisk(filepath.Join(d.Path, rel))
}

// IsDir reports whether p is a directory, following a link as os.Stat does.
func IsDir(p string) bool {
	fi, err := os.Stat(p)
	if err != nil {
		return false
	}
	return fi.IsDir()
}

// WalkHome walks a home directory once for every collector that probes
// per-directory files (MCP project configs, instruction files, workspace
// markers, framework manifests). The home is walked to depth 3 and each
// project root to depth 3 on its own budget, the same reach the collectors had
// with separate walks, but no directory is read twice: project roots are cut
// out of the home walk. Roots are taken from the home's listing, so paths keep
// their on-disk names on a case-insensitive filesystem.
//
// probes are the paths, relative to a walked directory, that collectors check
// (".mcp.json", ".cursor/mcp.json", ...). Only directories containing the first
// element of one of them are returned, each with only those entries, so the
// walk's memory grows with what it finds rather than with the size of the home.
func WalkHome(home string, probes []string) []WalkedDir {
	kept := make(map[string]struct{}, len(probes))
	for _, p := range probes {
		first, _, _ := strings.Cut(p, string(filepath.Separator))
		kept[first] = struct{}{}
	}

	entries, err := os.ReadDir(home)
	if err != nil {
		return nil
	}
	var roots []string
	isRoot := map[string]struct{}{}
	for _, e := range entries {
		if !slices.Contains(projectRootNames, strings.ToLower(e.Name())) {
			continue
		}
		p := filepath.Join(home, e.Name())
		// Stat, not the entry type: a root may be a symlink to a directory
		// elsewhere, as the per-collector walks allowed.
		if fi, err := os.Stat(p); err == nil && fi.IsDir() {
			roots = append(roots, p)
			isRoot[p] = struct{}{}
		}
	}

	var out []WalkedDir
	walkBounded(home, 3, func(dir string, entries []fs.DirEntry, depth int) bool {
		if _, ok := isRoot[dir]; ok && depth == 1 {
			return false
		}
		if d, ok := newWalkedDir(dir, entries, kept); ok {
			out = append(out, d)
		}
		return true
	})
	for _, root := range roots {
		walkBounded(root, 3, func(dir string, entries []fs.DirEntry, _ int) bool {
			if d, ok := newWalkedDir(dir, entries, kept); ok {
				d.Project = true
				out = append(out, d)
			}
			return true
		})
	}
	return out
}
