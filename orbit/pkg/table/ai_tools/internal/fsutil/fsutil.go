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
)

// maxHashBytes bounds how large a file we are willing to hash. Hashing streams
// in constant memory, so the limit caps I/O time per query, not memory. Files
// larger than this return an empty hash rather than a misleading prefix hash.
const maxHashBytes = 256 << 20 // 256 MiB

// SHA256 returns the lowercase hex SHA-256 of the file at path, or "" if the
// file can't be read, is a directory, or exceeds maxHashBytes.
func SHA256(path string) string {
	if path == "" {
		return ""
	}
	fi, err := os.Stat(path)
	if err != nil || fi.IsDir() || fi.Size() > maxHashBytes {
		return ""
	}
	f, err := os.Open(path) // #nosec G304 -- caller passes a path already discovered by a curated collector
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

// Exists reports whether path is an existing regular (non-directory) file.
func Exists(path string) bool {
	fi, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !fi.IsDir()
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
