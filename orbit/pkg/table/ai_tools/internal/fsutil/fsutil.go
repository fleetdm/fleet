// Package fsutil holds small, dependency-free filesystem helpers shared across
// collectors: content hashing (a diffable integrity fingerprint) and POSIX
// permission inspection (used to flag world-readable secret-bearing files).
//
// These never execute a discovered file — they only stat and read it — so they
// preserve the extension's no-exec security posture.
package fsutil

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
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

	h := sha256.New()
	// LimitReader guards against a file that grows past the cap between stat and
	// read (e.g. an actively-written log) so we never stream unbounded.
	if _, err := io.Copy(h, io.LimitReader(f, maxHashBytes)); err != nil {
		return ""
	}
	return hex.EncodeToString(h.Sum(nil))
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

// Perm describes the POSIX permission posture of a file. Known is false on
// platforms where Unix mode bits are not meaningful (Windows), so callers don't
// emit false "world-readable" signals there.
type Perm struct {
	WorldReadable bool // group OR other has read
	WorldWritable bool // group OR other has write
	Known         bool
}

// Stat returns the permission posture of path. On Windows, Known is false.
func Stat(path string) Perm {
	if runtime.GOOS == "windows" {
		return Perm{}
	}
	fi, err := os.Lstat(path)
	if err != nil {
		return Perm{}
	}
	m := fi.Mode().Perm()
	return Perm{
		WorldReadable: m&0o044 != 0,
		WorldWritable: m&0o022 != 0,
		Known:         true,
	}
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
// (.cursor, .vscode, .github, ...) are passed to visit but not descended, so
// callers probe known dotted paths via visit() without paying to recurse them.
func WalkBounded(root string, maxDepth int, visit func(dir string)) {
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
		visit(it.dir)
		if it.depth >= maxDepth {
			continue
		}
		entries, err := os.ReadDir(it.dir)
		if err != nil {
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
