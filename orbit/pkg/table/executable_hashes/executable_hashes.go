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
	"time"

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

	budget := currentHashBudget()

	var output []fileInfo
	deferred := 0
	for _, p := range paths {
		info, status := processPath(ctx, p, budget)
		switch status {
		case hashOK:
			output = append(output, info)
		case hashDeferred:
			deferred++
		}
	}

	if deferred > 0 {
		log.Debug().Int("count", deferred).Msg("byte budget spent, files deferred to a later run")
	}

	return output, nil
}

// processPath returns the row for a path. Unreadable paths and non-Mach-O files yield no row.
func processPath(ctx context.Context, path string, budget *hashBudget) (fileInfo, hashStatus) {
	stat, err := os.Stat(path)
	if err != nil {
		log.Debug().Err(err).Str("path", path).Msg("skipping path that could not be read")
		return fileInfo{}, hashUnavailable
	}

	if stat.IsDir() {
		return processBundle(ctx, path), hashOK
	}
	if !stat.Mode().IsRegular() {
		return fileInfo{}, hashUnavailable
	}
	return processMachOFile(path, stat, budget)
}

// processBundle always returns a row. A bundle with no executable on disk (e.g. Apple's
// XProtect.bundle) gets an empty hash. Bundles never draw from the byte budget: the server has
// no deferral tolerance for the apps source, so a deferred bundle would drop the app's hash for
// a run.
func processBundle(ctx context.Context, path string) fileInfo {
	row := fileInfo{Path: path, PathType: pathTypeBundle}

	row.ExecPath = getExecutablePath(ctx, path)
	if row.ExecPath == "" {
		return row
	}

	stat, err := os.Stat(row.ExecPath)
	if err != nil {
		log.Debug().Err(err).Str("path", row.ExecPath).Msg("executable could not be read, returning empty hash")
		return row
	}

	_, row.ExecSha256, _ = hashCached(row.ExecPath, stat, pathTypeBundle, nil)
	return row
}

func processMachOFile(path string, stat os.FileInfo, budget *hashBudget) (fileInfo, hashStatus) {
	execPath, hash, status := hashCached(path, stat, pathTypeFile, budget)
	if status != hashOK {
		return fileInfo{}, status
	}
	return fileInfo{Path: path, ExecPath: execPath, ExecSha256: hash, PathType: pathTypeFile}, hashOK
}

type hashStatus int

const (
	hashOK          hashStatus = iota
	hashUnavailable            // unreadable, or not Mach-O when one was required
	hashDeferred               // this run's byte budget is spent; a later run hashes the file
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

	// Aliases of one file (clang, clang++ -> clang-18) share its identity; read it once.
	if hash, ok := fileHashCache.hashByIdentity(identityKey{pathType: pathType, fileIdentity: newFileIdentity(stat)}); ok {
		fileHashCache.add(newHashCacheKey(path, pathType, stat), hashCacheEntry{execPath: execPath, hash: hash})
		return execPath, hash, hashOK
	}

	var hashed fileIdentity
	hash, hashed, status = hashFile(execPath, requireMachO, budget)
	if status == hashOK {
		// Key on the descriptor's own stat, which is exactly what was hashed.
		fileHashCache.add(hashCacheKey{path: path, pathType: pathType, fileIdentity: hashed}, hashCacheEntry{execPath: execPath, hash: hash})
	}
	return execPath, hash, status
}

// hashFile returns the SHA-256 of the file at path and the identity of the descriptor it hashed.
// A nil budget admits everything. Read failures are skips, not errors, so one unreadable file
// does not cost the host every other hash in the batch.
func hashFile(path string, requireMachO bool, budget *hashBudget) (string, fileIdentity, hashStatus) {
	f, err := os.Open(path)
	if err != nil {
		log.Debug().Err(err).Str("path", path).Msg("skipping file that could not be opened")
		return "", fileIdentity{}, hashUnavailable
	}
	defer f.Close()

	before, err := f.Stat()
	if err != nil {
		log.Debug().Err(err).Str("path", path).Msg("skipping file that could not be stat'd")
		return "", fileIdentity{}, hashUnavailable
	}
	id := newFileIdentity(before)

	if requireMachO && !isMachO(f) {
		return "", fileIdentity{}, hashUnavailable
	}

	if budget != nil && !budget.charge(before.Size()) {
		return "", fileIdentity{}, hashDeferred
	}

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		log.Debug().Err(err).Str("path", path).Msg("skipping file that could not be read")
		return "", fileIdentity{}, hashUnavailable
	}

	// A write that lands mid-hash yields a digest of neither version.
	after, err := f.Stat()
	if err != nil || newFileIdentity(after) != id {
		log.Debug().Err(err).Str("path", path).Msg("skipping file that changed while being hashed")
		return "", fileIdentity{}, hashUnavailable
	}

	return hex.EncodeToString(h.Sum(nil)), id, hashOK
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

// hashByteBudget caps bytes hashed per run so a first pass over a large Cellar does not trip
// osquery's watchdog, which kills the extension. One budget is shared by every Generate call in
// a run, since a correlated join calls the table once per keg; deferred files are hashed by
// later runs.
var hashByteBudget int64 = 2 << 30 // 2 GiB

// hashBudgetWindow: Generate calls this close together belong to one run and share a budget.
// Well below the hourly detail interval, well above the seconds one run's calls span.
var hashBudgetWindow = 10 * time.Minute

var hashBudgetNow = time.Now

var sharedHashBudget struct {
	mu      sync.Mutex
	current *hashBudget
}

type hashBudget struct {
	mu        sync.Mutex
	createdAt time.Time
	remaining int64
	spent     bool
}

// currentHashBudget returns the run's shared budget, starting a fresh one when the window has
// passed since the current one was created.
func currentHashBudget() *hashBudget {
	sharedHashBudget.mu.Lock()
	defer sharedHashBudget.mu.Unlock()

	now := hashBudgetNow()
	if b := sharedHashBudget.current; b == nil || now.Sub(b.createdAt) >= hashBudgetWindow {
		sharedHashBudget.current = &hashBudget{createdAt: now, remaining: hashByteBudget}
	}
	return sharedHashBudget.current
}

// charge reports whether a file of size bytes may be hashed. A file larger than what remains
// is admitted only as the first file of the run: it would never fit otherwise, and one such
// file per run is the bound on the burst.
func (b *hashBudget) charge(size int64) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	if size > b.remaining && b.spent {
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

type identityKey struct {
	pathType string
	fileIdentity
}

type hashCache struct {
	mu         sync.Mutex
	max        int
	entries    map[hashCacheKey]hashCacheEntry
	byIdentity map[identityKey]string
}

func newHashCache(maxEntries int) *hashCache {
	return &hashCache{
		max:        maxEntries,
		entries:    make(map[hashCacheKey]hashCacheEntry),
		byIdentity: make(map[identityKey]string),
	}
}

func (c *hashCache) hashByIdentity(key identityKey) (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	hash, ok := c.byIdentity[key]
	return hash, ok
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
		clear(c.byIdentity)
	}
	c.entries[key] = entry
	c.byIdentity[identityKey{pathType: key.pathType, fileIdentity: key.fileIdentity}] = entry.hash
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
