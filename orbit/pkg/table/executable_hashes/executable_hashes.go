//go:build darwin

// Package executable_hashes implements an extension osquery table to get information about a macOS bundle
package executable_hashes

import (
	"context"
	"crypto/sha256"
	"debug/macho"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/osquery/osquery-go/plugin/table"
	"github.com/rs/zerolog/log"
)

const (
	colPath     = "path"
	colExecPath = "executable_path"
	colExecHash = "executable_sha256"
	colPathType = "path_type"

	// pathTypeBundle: `path` is an app bundle, `executable_path` the executable its Info.plist names.
	pathTypeBundle = "bundle"
	// pathTypeFile: `path` is a Mach-O file, possibly a symlink; `executable_path` is the resolved file.
	pathTypeFile = "file"
)

// Columns is the schema of the table.
func Columns() []table.ColumnDefinition {
	return []table.ColumnDefinition{
		table.TextColumn(colPath),
		table.TextColumn(colExecPath),
		table.TextColumn(colExecHash),
		table.TextColumn(colPathType),
	}
}

// Generate is called to return the results for the table at query time.
func Generate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	path := ""
	wildcard := false

	var results []map[string]string

	if constraintList, present := queryContext.Constraints[colPath]; present {
		// 'path' is in the where clause
		for _, constraint := range constraintList.Constraints {
			path = constraint.Expression

			switch constraint.Operator {
			case table.OperatorLike:
				path = constraint.Expression
				wildcard = true
			case table.OperatorEquals:
				path = constraint.Expression
				wildcard = false
			}
		}
	} else {
		return results, errors.New("missing `path` constraint: provide a `path` in the query's `WHERE` clause")
	}

	processed, err := processFile(ctx, path, wildcard)
	if err != nil {
		return nil, err
	}

	for _, res := range processed {
		results = append(results, map[string]string{
			colPath:     res.Path,
			colExecPath: res.ExecPath,
			colExecHash: res.ExecSha256,
			colPathType: res.PathType,
		})
	}

	return results, nil
}

type fileInfo struct {
	Path       string
	ExecPath   string
	ExecSha256 string
	PathType   string
}

func processFile(ctx context.Context, path string, wildcard bool) ([]fileInfo, error) {
	paths := []string{path}

	if wildcard {
		replacedPath := strings.ReplaceAll(path, "%", "*")

		resolvedPaths, err := filepath.Glob(replacedPath)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve filepaths for incoming path: %w", err)
		}
		paths = resolvedPaths
	}

	budget := newHashBudget()

	var output []fileInfo
	for _, p := range paths {
		if info, ok := processPath(ctx, p, budget); ok {
			output = append(output, info)
		}
	}

	if budget.deferred > 0 {
		log.Debug().Int("count", budget.deferred).Msg("byte budget spent, files deferred to a later call")
	}

	return output, nil
}

// processPath returns the row for a path. Unreadable paths and non-Mach-O files yield no row.
func processPath(ctx context.Context, path string, budget *hashBudget) (fileInfo, bool) {
	stat, err := os.Stat(path)
	if err != nil {
		log.Debug().Err(err).Str("path", path).Msg("skipping path that could not be read")
		return fileInfo{}, false
	}

	if stat.IsDir() {
		return processBundle(ctx, path, budget)
	}
	if !stat.Mode().IsRegular() {
		return fileInfo{}, false
	}
	return processMachOFile(path, stat, budget)
}

// processBundle always returns a row. A bundle with no executable on disk (e.g. Apple's
// XProtect.bundle) gets an empty hash.
func processBundle(ctx context.Context, path string, budget *hashBudget) (fileInfo, bool) {
	row := fileInfo{Path: path, PathType: pathTypeBundle}

	row.ExecPath = getExecutablePath(ctx, path)
	if row.ExecPath == "" {
		return row, true
	}

	stat, err := os.Stat(row.ExecPath)
	if err != nil {
		log.Debug().Err(err).Str("path", row.ExecPath).Msg("executable could not be read, returning empty hash")
		return row, true
	}

	_, hash, status := hashCached(row.ExecPath, stat, pathTypeBundle, budget)
	if status == hashDeferred {
		return fileInfo{}, false
	}
	row.ExecSha256 = hash
	return row, true
}

func processMachOFile(path string, stat os.FileInfo, budget *hashBudget) (fileInfo, bool) {
	execPath, hash, status := hashCached(path, stat, pathTypeFile, budget)
	if status != hashOK {
		return fileInfo{}, false
	}
	return fileInfo{Path: path, ExecPath: execPath, ExecSha256: hash, PathType: pathTypeFile}, true
}

type hashStatus int

const (
	hashOK          hashStatus = iota
	hashUnavailable            // unreadable, or not Mach-O when one was required
	hashDeferred               // this call's byte budget is spent; a later call hashes the file
)

// hashCached returns the hashed path and SHA-256 of the file at path. File rows resolve
// symlinks (a keg's bin often links into libexec) and must be Mach-O; bundle rows hash as given.
//
// Results are keyed on the queried path, type and the target's identity, so a cached file
// costs only the caller's stat. The stat follows symlinks, so a retargeted link changes the
// inode and misses.
func hashCached(path string, stat os.FileInfo, pathType string, budget *hashBudget) (execPath, hash string, status hashStatus) {
	if entry, ok := fileHashCache.get(newHashCacheKey(path, pathType, stat)); ok {
		return entry.execPath, entry.hash, hashOK
	}

	execPath = path
	requireMachO := pathType == pathTypeFile
	if requireMachO {
		var err error
		if execPath, err = filepath.EvalSymlinks(path); err != nil {
			log.Debug().Err(err).Str("path", path).Msg("skipping path whose symlinks could not be resolved")
			return "", "", hashUnavailable
		}
	}

	var hashed os.FileInfo
	hash, hashed, status = hashFile(execPath, requireMachO, budget)
	if status == hashOK {
		// Key on the descriptor's own stat, which is exactly what was hashed.
		fileHashCache.add(newHashCacheKey(path, pathType, hashed), hashCacheEntry{execPath: execPath, hash: hash})
	}
	return execPath, hash, status
}

// hashFile returns the SHA-256 of the file at path and the stat of the descriptor it hashed.
// Read failures are skips, not errors, so one unreadable file does not cost the host every
// other hash in the batch.
func hashFile(path string, requireMachO bool, budget *hashBudget) (string, os.FileInfo, hashStatus) {
	f, err := os.Open(path)
	if err != nil {
		log.Debug().Err(err).Str("path", path).Msg("skipping file that could not be opened")
		return "", nil, hashUnavailable
	}
	defer f.Close()

	before, err := f.Stat()
	if err != nil {
		log.Debug().Err(err).Str("path", path).Msg("skipping file that could not be stat'd")
		return "", nil, hashUnavailable
	}

	if requireMachO && !isMachO(f) {
		return "", nil, hashUnavailable
	}

	if !budget.charge(before.Size()) {
		return "", nil, hashDeferred
	}

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		log.Debug().Err(err).Str("path", path).Msg("skipping file that could not be read")
		return "", nil, hashUnavailable
	}

	// A write that lands mid-hash yields a digest of neither version.
	after, err := f.Stat()
	if err != nil || newFileIdentity(after) != newFileIdentity(before) {
		log.Debug().Err(err).Str("path", path).Msg("skipping file that changed while being hashed")
		return "", nil, hashUnavailable
	}

	return hex.EncodeToString(h.Sum(nil)), before, hashOK
}

const (
	// FAT_MAGIC_64 from mach-o/fat.h; debug/macho does not define it.
	magicFat64 uint32 = 0xcafebabf

	// Java class files share the fat magic. Bytes 4-8 are nfat_arch in a fat header but
	// minor (0) then major (>= 45) version in a class file. file(1) also uses 20.
	maxFatArch = 20
)

// isMachO reports whether f starts with a Mach-O or fat magic, without moving its offset.
func isMachO(f *os.File) bool {
	var hdr [8]byte
	if _, err := f.ReadAt(hdr[:], 0); err != nil {
		return false
	}

	le := binary.LittleEndian.Uint32(hdr[:4])
	be := binary.BigEndian.Uint32(hdr[:4])
	switch {
	case le == macho.Magic32, le == macho.Magic64, be == macho.Magic32, be == macho.Magic64:
		return true
	case be == macho.MagicFat, be == magicFat64:
		return binary.BigEndian.Uint32(hdr[4:8]) < maxFatArch
	}
	return false
}

// hashByteBudget caps bytes hashed per Generate call so a first pass over a large Cellar
// does not trip osquery's watchdog, which kills the extension. Deferred files are hashed
// by later calls.
var hashByteBudget int64 = 2 << 30 // 2 GiB

type hashBudget struct {
	remaining int64
	spent     bool
	deferred  int
}

func newHashBudget() *hashBudget {
	return &hashBudget{remaining: hashByteBudget}
}

// charge reports whether a file of size bytes may be hashed. A file larger than what remains
// is admitted only as the first file of a call: it would never fit otherwise, and one such file
// per call is the bound on the burst.
func (b *hashBudget) charge(size int64) bool {
	if size > b.remaining && b.spent {
		b.deferred++
		return false
	}
	b.remaining -= size
	b.spent = true
	return true
}

// hashCacheMaxEntries covers any realistic Cellar. A full cache is cleared outright: a
// working set past the bound would churn FIFO or LRU on every call anyway.
const hashCacheMaxEntries = 10000

var fileHashCache = newHashCache(hashCacheMaxEntries)

type hashCacheKey struct {
	path     string
	pathType string
	fileIdentity
}

// fileIdentity is what the cache trusts to say a file is unchanged. Size and mtime alone can
// be preserved by `touch -r`; ctime cannot be set from userland and a new file has a new inode.
type fileIdentity struct {
	dev        int32
	ino        uint64
	size       int64
	modTime    int64
	changeTime int64
}

func newFileIdentity(stat os.FileInfo) fileIdentity {
	id := fileIdentity{size: stat.Size(), modTime: stat.ModTime().UnixNano()}
	if st, ok := stat.Sys().(*syscall.Stat_t); ok {
		id.dev, id.ino, id.changeTime = st.Dev, st.Ino, st.Ctimespec.Nano()
	}
	return id
}

func newHashCacheKey(path, pathType string, stat os.FileInfo) hashCacheKey {
	return hashCacheKey{path: path, pathType: pathType, fileIdentity: newFileIdentity(stat)}
}

type hashCacheEntry struct {
	execPath string
	hash     string
}

type hashCache struct {
	mu      sync.Mutex
	max     int
	entries map[hashCacheKey]hashCacheEntry
}

func newHashCache(maxEntries int) *hashCache {
	return &hashCache{max: maxEntries, entries: make(map[hashCacheKey]hashCacheEntry)}
}

func (c *hashCache) get(key hashCacheKey) (hashCacheEntry, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, ok := c.entries[key]
	return entry, ok
}

func (c *hashCache) add(key hashCacheKey, entry hashCacheEntry) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.entries[key]; !exists && len(c.entries) >= c.max {
		log.Debug().Int("entries", len(c.entries)).Msg("hash cache full, clearing")
		clear(c.entries)
	}
	c.entries[key] = entry
}

func getExecutablePath(ctx context.Context, path string) string {
	infoPlistPath := filepath.Join(path, "/Contents/Info.plist")
	if _, err := os.Stat(infoPlistPath); err != nil {
		// A glob matches plain directories too; don't spawn `defaults` for each of them.
		log.Debug().Err(err).Str("path", path).Msg("no Info.plist, returning empty binary path")
		return ""
	}
	output, err := exec.CommandContext(ctx, "/usr/bin/defaults", "read", infoPlistPath, "CFBundleExecutable").Output()
	if err != nil {
		// lots of helper app bundles nested within parent bundles seem to have invalid Info.plists - warn and continue
		log.Warn().Err(err).Str("path", path).Msg("failed to read CFBundleExecutable from Info.plist, returning empty binary path")
		return ""
	}

	executableName := strings.TrimSpace(string(output))
	if executableName == "" {
		return ""
	}

	// The macOS `defaults read` command encodes supplementary Unicode characters (such as emoji) as
	// \uXXXX escape sequences using UTF-16 surrogate pairs.
	executableName = fleet.DecodeUnicodeEscapes(executableName)

	return filepath.Join(path, "/Contents/MacOS/", executableName)
}
