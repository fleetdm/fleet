//go:build darwin

package executable_hashes

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/osquery/osquery-go/plugin/table"
	"github.com/stretchr/testify/require"
)

func TestGenerateWithExactPath(t *testing.T) {
	tests := []struct {
		name           string
		bundleName     string
		executableName string
		content        []byte
	}{
		{
			name:           "ASCII app name",
			bundleName:     "Test.app",
			executableName: "Test",
			content:        []byte("test file content for hashing"),
		},
		{
			name:           "emoji app name",
			bundleName:     "🖨️ Printer.app",
			executableName: "🖨️ Printer",
			content:        []byte("emoji executable content"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()

			bundlePath := filepath.Join(dir, tt.bundleName)
			contentsDir := filepath.Join(bundlePath, "Contents")
			macosDir := filepath.Join(contentsDir, "MacOS")
			require.NoError(t, os.MkdirAll(macosDir, 0o755))

			infoPlistPath := filepath.Join(contentsDir, "Info.plist")
			infoPlistContent := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>CFBundleExecutable</key>
	<string>%s</string>
</dict>
</plist>`, tt.executableName)
			require.NoError(t, os.WriteFile(infoPlistPath, []byte(infoPlistContent), 0o644))

			execPath := filepath.Join(macosDir, tt.executableName)
			require.NoError(t, os.WriteFile(execPath, tt.content, 0o644))

			h := sha256.New()
			h.Write(tt.content)
			expectedHash := hex.EncodeToString(h.Sum(nil))

			rows, err := Generate(t.Context(), table.QueryContext{
				Constraints: map[string]table.ConstraintList{
					colPath: {
						Constraints: []table.Constraint{{
							Expression: bundlePath,
							Operator:   table.OperatorEquals,
						}},
					},
				},
			})
			require.NoError(t, err)
			require.Len(t, rows, 1)
			require.Equal(t, bundlePath, rows[0][colPath])
			require.Equal(t, execPath, rows[0][colExecPath])
			require.Equal(t, expectedHash, rows[0][colExecHash])
			require.Equal(t, pathTypeBundle, rows[0][colPathType])
		})
	}
}

// TestGenerateWithExactPathMissingExecutable reproduces issue #45327: some app
// bundles (e.g. Apple's XProtect.bundle) declare a CFBundleExecutable in their
// Info.plist but ship no binary at that path. Generating the table must not fail.
func TestGenerateWithExactPathMissingExecutable(t *testing.T) {
	dir := t.TempDir()

	bundlePath := filepath.Join(dir, "XProtect.bundle")
	contentsDir := filepath.Join(bundlePath, "Contents")
	macosDir := filepath.Join(contentsDir, "MacOS")
	require.NoError(t, os.MkdirAll(macosDir, 0o755))

	// Valid Info.plist that names an executable, but we intentionally do NOT
	// create Contents/MacOS/XProtect.
	infoPlistPath := filepath.Join(contentsDir, "Info.plist")
	infoPlistContent := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>CFBundleExecutable</key>
	<string>XProtect</string>
</dict>
</plist>`
	require.NoError(t, os.WriteFile(infoPlistPath, []byte(infoPlistContent), 0o644))

	rows, err := Generate(t.Context(), table.QueryContext{
		Constraints: map[string]table.ConstraintList{
			colPath: {
				Constraints: []table.Constraint{{
					Expression: bundlePath,
					Operator:   table.OperatorEquals,
				}},
			},
		},
	})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, bundlePath, rows[0][colPath])
	require.Equal(t, filepath.Join(contentsDir, "MacOS", "XProtect"), rows[0][colExecPath])
	require.Empty(t, rows[0][colExecHash])
	require.Equal(t, pathTypeBundle, rows[0][colPathType])
}

// TestGenerateWithWildcardPartialMissingExecutables ensures a single bundle whose
// executable is missing does not abort hashing of the rest of the wildcard batch.
func TestGenerateWithWildcardPartialMissingExecutables(t *testing.T) {
	dir := t.TempDir()

	testBundles := map[string]struct {
		executableName string
		content        []byte
		createExec     bool
	}{
		"Good.app":    {"Good", []byte("content of good"), true},
		"Missing.app": {"Missing", nil, false},
	}

	expectedHashByBundlePath := make(map[string]string)
	expectedExecPathByBundlePath := make(map[string]string)

	for bundleName, bundleInfo := range testBundles {
		bundlePath := filepath.Join(dir, bundleName)
		contentsDir := filepath.Join(bundlePath, "Contents")
		macosDir := filepath.Join(contentsDir, "MacOS")
		require.NoError(t, os.MkdirAll(macosDir, 0o755))

		infoPlistPath := filepath.Join(contentsDir, "Info.plist")
		infoPlistContent := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>CFBundleExecutable</key>
	<string>%s</string>
</dict>
</plist>`, bundleInfo.executableName)
		require.NoError(t, os.WriteFile(infoPlistPath, []byte(infoPlistContent), 0o644))

		execPath := filepath.Join(macosDir, bundleInfo.executableName)
		expectedExecPathByBundlePath[bundlePath] = execPath

		if bundleInfo.createExec {
			require.NoError(t, os.WriteFile(execPath, bundleInfo.content, 0o644))
			h := sha256.New()
			h.Write(bundleInfo.content)
			expectedHashByBundlePath[bundlePath] = hex.EncodeToString(h.Sum(nil))
		} else {
			// Missing executable -> empty hash, but still a row.
			expectedHashByBundlePath[bundlePath] = ""
		}
	}

	rows, err := Generate(t.Context(), table.QueryContext{
		Constraints: map[string]table.ConstraintList{
			colPath: {
				Constraints: []table.Constraint{{
					Expression: filepath.Join(dir, "%.app"),
					Operator:   table.OperatorLike,
				}},
			},
		},
	})
	require.NoError(t, err)
	require.Len(t, rows, 2)

	got := make(map[string]fileInfo, 2)
	for _, row := range rows {
		got[row[colPath]] = fileInfo{
			Path:       row[colPath],
			ExecPath:   row[colExecPath],
			ExecSha256: row[colExecHash],
			PathType:   row[colPathType],
		}
	}

	for bundlePath, expectedHash := range expectedHashByBundlePath {
		require.Contains(t, got, bundlePath)
		info := got[bundlePath]
		require.Equal(t, expectedExecPathByBundlePath[bundlePath], info.ExecPath)
		require.Equal(t, expectedHash, info.ExecSha256)
		require.Equal(t, pathTypeBundle, info.PathType)
	}
}

func TestGenerateWithExactPathMissing(t *testing.T) {
	require.Empty(t, generateExact(t, filepath.Join(t.TempDir(), "does-not-exist")))
}

// A directory that is not a bundle keeps its historical row: empty executable path and hash.
func TestGenerateWithPlainDirectory(t *testing.T) {
	dir := t.TempDir()
	rows := generateExact(t, dir)
	require.Len(t, rows, 1)
	require.Equal(t, dir, rows[0][colPath])
	require.Empty(t, rows[0][colExecPath])
	require.Empty(t, rows[0][colExecHash])
	require.Equal(t, pathTypeBundle, rows[0][colPathType])
}

func TestGenerateWithWildcard(t *testing.T) {
	dir := t.TempDir()
	defer os.RemoveAll(dir)

	testBundles := map[string]struct {
		executableName string
		content        []byte
	}{
		"Foo.app":      {"Foo", []byte("content of foo")},
		"Bar.app":      {"Bar", []byte("content of bar")},
		"Baz.service":  {"Baz", []byte("content of baz")},
		"Bonk.service": {"Bonk", []byte("content of bonk")},
	}

	expectedHashByBundlePath := make(map[string]string)
	expectedExecPathByBundlePath := make(map[string]string)

	// Create macOS app bundle structures
	for bundleName, bundleInfo := range testBundles {
		bundlePath := filepath.Join(dir, bundleName)
		contentsDir := filepath.Join(bundlePath, "Contents")
		macosDir := filepath.Join(contentsDir, "MacOS")
		require.NoError(t, os.MkdirAll(macosDir, 0o755))

		// Create Info.plist with CFBundleExecutable key
		infoPlistPath := filepath.Join(contentsDir, "Info.plist")
		infoPlistContent := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>CFBundleExecutable</key>
	<string>%s</string>
</dict>
</plist>`, bundleInfo.executableName)
		require.NoError(t, os.WriteFile(infoPlistPath, []byte(infoPlistContent), 0o644))

		// Create the actual executable in Contents/MacOS/
		execPath := filepath.Join(macosDir, bundleInfo.executableName)
		require.NoError(t, os.WriteFile(execPath, bundleInfo.content, 0o644))

		h := sha256.New()
		h.Write(bundleInfo.content)
		expectedHashByBundlePath[bundlePath] = hex.EncodeToString(h.Sum(nil))
		expectedExecPathByBundlePath[bundlePath] = execPath
	}

	rows, err := Generate(t.Context(), table.QueryContext{
		Constraints: map[string]table.ConstraintList{
			colPath: {
				Constraints: []table.Constraint{{
					Expression: filepath.Join(dir, "%.app"),
					Operator:   table.OperatorLike,
				}},
			},
		},
	})
	require.NoError(t, err)
	require.Len(t, rows, 2)

	serviceRows, err := Generate(t.Context(), table.QueryContext{
		Constraints: map[string]table.ConstraintList{
			colPath: {
				Constraints: []table.Constraint{{
					Expression: filepath.Join(dir, "%.service"),
					Operator:   table.OperatorLike,
				}},
			},
		},
	})
	require.NoError(t, err)
	require.Len(t, serviceRows, 2)
	rows = append(rows, serviceRows...)

	got := make(map[string]fileInfo, 4)
	for _, row := range rows {
		got[row[colPath]] = fileInfo{
			Path:       row[colPath],
			ExecPath:   row[colExecPath],
			ExecSha256: row[colExecHash],
			PathType:   row[colPathType],
		}
	}

	for bundlePath, expectedHash := range expectedHashByBundlePath {
		require.Contains(t, got, bundlePath)
		info := got[bundlePath]
		require.Equal(t, bundlePath, info.Path)
		require.Equal(t, expectedExecPathByBundlePath[bundlePath], info.ExecPath)
		require.Equal(t, expectedHash, info.ExecSha256)
		require.Equal(t, pathTypeBundle, info.PathType)
	}
}

// Headers used by the file tests below. Only the first 8 bytes matter to the
// table; the rest of each fixture is arbitrary content to hash.
var (
	machOHeader = []byte{0xcf, 0xfa, 0xed, 0xfe, 0x0c, 0x00, 0x00, 0x01} // 64-bit Mach-O
	fatHeader   = []byte{0xca, 0xfe, 0xba, 0xbe, 0x00, 0x00, 0x00, 0x02} // fat binary, nfat_arch = 2
	classHeader = []byte{0xca, 0xfe, 0xba, 0xbe, 0x00, 0x00, 0x00, 0x3d} // Java 17 class: minor 0, major 61
)

func TestIsMachO(t *testing.T) {
	tests := []struct {
		name   string
		header []byte
		want   bool
	}{
		{"64-bit little-endian", machOHeader, true},
		{"32-bit little-endian", []byte{0xce, 0xfa, 0xed, 0xfe, 0x0c, 0x00, 0x00, 0x00}, true},
		{"64-bit big-endian", []byte{0xfe, 0xed, 0xfa, 0xcf, 0x01, 0x00, 0x00, 0x0c}, true},
		{"fat with 2 architectures", fatHeader, true},
		{"fat with 19 architectures", []byte{0xca, 0xfe, 0xba, 0xbe, 0x00, 0x00, 0x00, 0x13}, true},
		{"fat64 with 2 architectures", []byte{0xca, 0xfe, 0xba, 0xbf, 0x00, 0x00, 0x00, 0x02}, true},
		{"fat with 20 architectures", []byte{0xca, 0xfe, 0xba, 0xbe, 0x00, 0x00, 0x00, 0x14}, false},
		{"Java 8 class", []byte{0xca, 0xfe, 0xba, 0xbe, 0x00, 0x00, 0x00, 0x34}, false},
		{"Java 17 class", classHeader, false},
		{"Java 21 class", []byte{0xca, 0xfe, 0xba, 0xbe, 0x00, 0x00, 0x00, 0x41}, false},
		{"Java preview class", []byte{0xca, 0xfe, 0xba, 0xbe, 0xff, 0xff, 0x00, 0x41}, false},
		{"shell script", []byte("#!/bin/sh\n"), false},
		{"shorter than a header", []byte{0xcf, 0xfa, 0xed}, false},
		{"empty", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "f")
			require.NoError(t, os.WriteFile(path, tt.header, 0o644))
			f, err := os.Open(path)
			require.NoError(t, err)
			defer f.Close()

			require.Equal(t, tt.want, isMachO(f))

			// The offset must stay at 0 so the hash covers the whole file.
			offset, err := f.Seek(0, io.SeekCurrent)
			require.NoError(t, err)
			require.Zero(t, offset)
		})
	}
}

// resetHashState gives the test a clean package-level cache and the default byte
// budget, both of which outlive a single Generate call.
func resetHashState(t *testing.T) {
	t.Helper()
	previousBudget, previousWindow, previousNow := hashByteBudget, hashBudgetWindow, hashBudgetNow
	fileHashCache = newHashCache(hashCacheMaxEntries)
	fakeNow = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	hashBudgetNow = func() time.Time { return fakeNow }
	resetSharedBudget()
	t.Cleanup(func() {
		hashByteBudget, hashBudgetWindow, hashBudgetNow = previousBudget, previousWindow, previousNow
		fileHashCache = newHashCache(hashCacheMaxEntries)
		resetSharedBudget()
	})
}

// fakeNow is the clock the budget window reads during tests.
var fakeNow time.Time

func resetSharedBudget() {
	sharedHashBudget.mu.Lock()
	defer sharedHashBudget.mu.Unlock()
	sharedHashBudget.current = nil
}

// nextRun moves the clock past the budget window, as the next hourly detail run would.
func nextRun(t *testing.T) {
	t.Helper()
	fakeNow = fakeNow.Add(hashBudgetWindow)
}

func writeFixture(t *testing.T, path string, header []byte, body string) []byte {
	t.Helper()
	content := append(append([]byte{}, header...), body...)
	require.NoError(t, os.WriteFile(path, content, 0o644))
	return content
}

// resolve is what the table reports as executable_path: the fully symlink-resolved
// path. On macOS a temp dir is itself reached through a symlink (/var -> private/var).
func resolve(t *testing.T, path string) string {
	t.Helper()
	resolved, err := filepath.EvalSymlinks(path)
	require.NoError(t, err)
	return resolved
}

func sha256Hex(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

func generateLike(t *testing.T, pattern string) []map[string]string {
	t.Helper()
	rows, err := Generate(t.Context(), table.QueryContext{
		Constraints: map[string]table.ConstraintList{
			colPath: {
				Constraints: []table.Constraint{{Expression: pattern, Operator: table.OperatorLike}},
			},
		},
	})
	require.NoError(t, err)
	return rows
}

func generateExact(t *testing.T, path string) []map[string]string {
	t.Helper()
	rows, err := Generate(t.Context(), table.QueryContext{
		Constraints: map[string]table.ConstraintList{
			colPath: {
				Constraints: []table.Constraint{{Expression: path, Operator: table.OperatorEquals}},
			},
		},
	})
	require.NoError(t, err)
	return rows
}

func rowsByPath(rows []map[string]string) map[string]map[string]string {
	byPath := make(map[string]map[string]string, len(rows))
	for _, row := range rows {
		byPath[row[colPath]] = row
	}
	return byPath
}

func TestGenerateWithWildcardMachOFiles(t *testing.T) {
	resetHashState(t)
	dir := t.TempDir()

	machOPath := filepath.Join(dir, "thin")
	machOContent := writeFixture(t, machOPath, machOHeader, "thin mach-o content")
	resolvedMachOPath := resolve(t, machOPath)

	fatPath := filepath.Join(dir, "fat")
	fatContent := writeFixture(t, fatPath, fatHeader, "fat mach-o content")
	resolvedFatPath := resolve(t, fatPath)

	// Neither of these is a Mach-O file, so neither produces a row. The Java class
	// file shares the fat binary's magic and is told apart by the bytes that follow.
	writeFixture(t, filepath.Join(dir, "script"), []byte("#!/bin/sh"), "\necho hello\n")
	writeFixture(t, filepath.Join(dir, "Some.class"), classHeader, "java class content")

	rows := generateLike(t, filepath.Join(dir, "%"))
	require.Len(t, rows, 2)

	byPath := rowsByPath(rows)
	require.Contains(t, byPath, machOPath)
	require.Equal(t, resolvedMachOPath, byPath[machOPath][colExecPath])
	require.Equal(t, sha256Hex(machOContent), byPath[machOPath][colExecHash])
	require.Equal(t, pathTypeFile, byPath[machOPath][colPathType])

	require.Contains(t, byPath, fatPath)
	require.Equal(t, resolvedFatPath, byPath[fatPath][colExecPath])
	require.Equal(t, sha256Hex(fatContent), byPath[fatPath][colExecHash])
	require.Equal(t, pathTypeFile, byPath[fatPath][colPathType])
}

func TestGenerateWithSymlinkToMachOFile(t *testing.T) {
	resetHashState(t)
	// The target lives outside the globbed directory so the glob only matches the symlink.
	targetDir := t.TempDir()
	dir := t.TempDir()

	targetPath := filepath.Join(targetDir, "tool")
	content := writeFixture(t, targetPath, machOHeader, "symlinked mach-o content")
	resolvedTarget := resolve(t, targetPath)

	linkPath := filepath.Join(dir, "tool")
	require.NoError(t, os.Symlink(targetPath, linkPath))

	rows := generateLike(t, filepath.Join(dir, "%"))
	require.Len(t, rows, 1)
	require.Equal(t, linkPath, rows[0][colPath])
	require.Equal(t, resolvedTarget, rows[0][colExecPath])
	require.Equal(t, sha256Hex(content), rows[0][colExecHash])
	require.Equal(t, pathTypeFile, rows[0][colPathType])
}

func TestGenerateSkipsUnreadableFile(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root, every file is readable")
	}
	resetHashState(t)
	dir := t.TempDir()

	readablePath := filepath.Join(dir, "readable")
	content := writeFixture(t, readablePath, machOHeader, "readable mach-o content")

	unreadablePath := filepath.Join(dir, "unreadable")
	writeFixture(t, unreadablePath, machOHeader, "unreadable mach-o content")
	require.NoError(t, os.Chmod(unreadablePath, 0o000))

	rows := generateLike(t, filepath.Join(dir, "%"))
	require.Len(t, rows, 1)
	require.Equal(t, readablePath, rows[0][colPath])
	require.Equal(t, sha256Hex(content), rows[0][colExecHash])
}

// A bundle whose executable exists but cannot be read keeps its row with an empty hash, and
// does not abort the rest of the batch. This is the bundle counterpart of the file case above.
func TestGenerateBundleWithUnreadableExecutable(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root, every file is readable")
	}
	resetHashState(t)
	dir := t.TempDir()

	goodContent := []byte("readable bundle executable")
	goodPath, goodExec := writeBundle(t, dir, "Good", "Good", goodContent)
	badPath, badExec := writeBundle(t, dir, "Bad", "Bad", []byte("unreadable bundle executable"))
	require.NoError(t, os.Chmod(badExec, 0o000))

	rows := generateLike(t, filepath.Join(dir, "%.app"))
	require.Len(t, rows, 2)

	byPath := rowsByPath(rows)
	require.Equal(t, goodExec, byPath[goodPath][colExecPath])
	require.Equal(t, sha256Hex(goodContent), byPath[goodPath][colExecHash])
	require.Equal(t, pathTypeBundle, byPath[goodPath][colPathType])

	require.Equal(t, badExec, byPath[badPath][colExecPath])
	require.Empty(t, byPath[badPath][colExecHash])
	require.Equal(t, pathTypeBundle, byPath[badPath][colPathType])
}

func TestGenerateWithExactPathToFile(t *testing.T) {
	resetHashState(t)
	dir := t.TempDir()

	machOPath := filepath.Join(dir, "tool")
	content := writeFixture(t, machOPath, machOHeader, "mach-o content")

	rows := generateExact(t, machOPath)
	require.Len(t, rows, 1)
	require.Equal(t, machOPath, rows[0][colPath])
	require.Equal(t, resolve(t, machOPath), rows[0][colExecPath])
	require.Equal(t, sha256Hex(content), rows[0][colExecHash])
	require.Equal(t, pathTypeFile, rows[0][colPathType])

	scriptPath := filepath.Join(dir, "script")
	writeFixture(t, scriptPath, []byte("#!/bin/sh"), "\necho hello\n")
	require.Empty(t, generateExact(t, scriptPath))
}

func TestHashCacheHitAndInvalidation(t *testing.T) {
	resetHashState(t)
	dir := t.TempDir()

	path := filepath.Join(dir, "tool")
	original := writeFixture(t, path, machOHeader, "first content")
	info, err := os.Stat(path)
	require.NoError(t, err)

	rows := generateLike(t, filepath.Join(dir, "%"))
	require.Len(t, rows, 1)
	require.Equal(t, sha256Hex(original), rows[0][colExecHash])

	// Plant a sentinel in the only entry to prove the next call is served from the cache.
	require.Len(t, fileHashCache.entries, 1)
	for key, entry := range fileHashCache.entries {
		entry.hash = "sentinel"
		fileHashCache.entries[key] = entry
	}
	rows = generateLike(t, filepath.Join(dir, "%"))
	require.Len(t, rows, 1)
	require.Equal(t, "sentinel", rows[0][colExecHash])

	// Same size, same mtime, different bytes. The rewrite still moves ctime, which cannot
	// be set from userland, so the cache misses and the real hash comes back.
	updated := writeFixture(t, path, machOHeader, "other content")
	require.Len(t, updated, len(original))
	require.NoError(t, os.Chtimes(path, info.ModTime(), info.ModTime()))

	rows = generateLike(t, filepath.Join(dir, "%"))
	require.Len(t, rows, 1)
	require.Equal(t, sha256Hex(updated), rows[0][colExecHash])
}

func TestHashCacheClearedWhenFull(t *testing.T) {
	resetHashState(t)
	fileHashCache = newHashCache(2)
	dir := t.TempDir()

	expected := make(map[string]string, 3)
	for _, name := range []string{"a", "b", "c"} {
		path := filepath.Join(dir, name)
		expected[path] = sha256Hex(writeFixture(t, path, machOHeader, "content "+name))
	}

	assertRows := func() {
		t.Helper()
		rows := generateLike(t, filepath.Join(dir, "%"))
		require.Len(t, rows, 3)
		for _, row := range rows {
			require.Equal(t, expected[row[colPath]], row[colExecHash])
		}
	}

	// The third insertion finds the cache full and clears both indexes before adding, so only
	// the last file is left cached. Every row is still returned with the right hash.
	assertRows()
	require.Len(t, fileHashCache.entries, 1)
	require.Len(t, fileHashCache.byIdentity, 1)

	// Cleared files are rehashed on the next call; the cache stays bounded.
	assertRows()
	require.LessOrEqual(t, len(fileHashCache.entries), fileHashCache.max)
	require.LessOrEqual(t, len(fileHashCache.byIdentity), fileHashCache.max)
}

func TestHashCacheFollowsSymlinkRetarget(t *testing.T) {
	resetHashState(t)
	targetDir := t.TempDir()
	dir := t.TempDir()

	firstPath := filepath.Join(targetDir, "first")
	secondPath := filepath.Join(targetDir, "second")
	first := writeFixture(t, firstPath, machOHeader, "content one")
	second := writeFixture(t, secondPath, machOHeader, "content two")
	require.Len(t, second, len(first))

	// Same mtime on both targets, so only identity and contents tell them apart.
	sameTime := time.Now().Add(-time.Hour).Truncate(time.Second)
	require.NoError(t, os.Chtimes(firstPath, sameTime, sameTime))
	require.NoError(t, os.Chtimes(secondPath, sameTime, sameTime))

	link := filepath.Join(dir, "tool")
	require.NoError(t, os.Symlink(firstPath, link))

	rows := generateExact(t, link)
	require.Len(t, rows, 1)
	require.Equal(t, resolve(t, firstPath), rows[0][colExecPath])
	require.Equal(t, sha256Hex(first), rows[0][colExecHash])

	require.NoError(t, os.Remove(link))
	require.NoError(t, os.Symlink(secondPath, link))

	rows = generateExact(t, link)
	require.Len(t, rows, 1)
	require.Equal(t, resolve(t, secondPath), rows[0][colExecPath])
	require.Equal(t, sha256Hex(second), rows[0][colExecHash])
}

func TestHashByteBudgetDefersFiles(t *testing.T) {
	resetHashState(t)
	dir := t.TempDir()

	var size int64
	for _, name := range []string{"a", "b", "c"} {
		content := writeFixture(t, filepath.Join(dir, name), machOHeader, "mach-o content "+name)
		size = int64(len(content))
	}
	// Room for one uncached file per call. Cached files are always returned, so the
	// deferred files are picked up one per call until every file is hashed.
	hashByteBudget = size

	require.Len(t, generateLike(t, filepath.Join(dir, "%")), 1)
	nextRun(t)
	require.Len(t, generateLike(t, filepath.Join(dir, "%")), 2)
	nextRun(t)
	require.Len(t, generateLike(t, filepath.Join(dir, "%")), 3)
}

// writeBundle creates dir/name.app with an executable holding content.
func writeBundle(t *testing.T, dir, name, executableName string, content []byte) (bundlePath, execPath string) {
	t.Helper()
	bundlePath = filepath.Join(dir, name+".app")
	macosDir := filepath.Join(bundlePath, "Contents", "MacOS")
	require.NoError(t, os.MkdirAll(macosDir, 0o755))

	infoPlist := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>CFBundleExecutable</key>
	<string>%s</string>
</dict>
</plist>`, executableName)
	require.NoError(t, os.WriteFile(filepath.Join(bundlePath, "Contents", "Info.plist"), []byte(infoPlist), 0o644))

	execPath = filepath.Join(macosDir, executableName)
	require.NoError(t, os.WriteFile(execPath, content, 0o644))
	return bundlePath, execPath
}

// A bundle executable cached without the Mach-O check must not satisfy a file query.
func TestBundleScriptExecutableNotReportedAsFile(t *testing.T) {
	resetHashState(t)
	content := []byte("#!/bin/sh\necho launcher\n")
	bundlePath, execPath := writeBundle(t, t.TempDir(), "Launcher", "Launcher", content)

	rows := generateExact(t, bundlePath)
	require.Len(t, rows, 1)
	require.Equal(t, sha256Hex(content), rows[0][colExecHash])
	require.Equal(t, pathTypeBundle, rows[0][colPathType])

	require.Empty(t, generateExact(t, execPath))
}

func TestHashByteBudgetDefersFileExceedingRemainder(t *testing.T) {
	resetHashState(t)
	dir := t.TempDir()

	small := writeFixture(t, filepath.Join(dir, "a-small"), machOHeader, "small")
	big := writeFixture(t, filepath.Join(dir, "b-big"), machOHeader, "big content that is longer")
	// The big file fits the whole budget on its own but not what is left after the small one.
	hashByteBudget = int64(len(small)) + int64(len(big))/2

	rows := generateLike(t, filepath.Join(dir, "%"))
	require.Len(t, rows, 1)
	require.Equal(t, filepath.Join(dir, "a-small"), rows[0][colPath])

	nextRun(t)
	require.Len(t, generateLike(t, filepath.Join(dir, "%")), 2)
}

func TestHashByteBudgetAdmitsOversizeFileOnlyFirst(t *testing.T) {
	resetHashState(t)
	dir := t.TempDir()

	content := writeFixture(t, filepath.Join(dir, "a"), machOHeader, "oversize content")
	writeFixture(t, filepath.Join(dir, "b"), machOHeader, "oversize content")
	// Each file alone is larger than the whole budget.
	hashByteBudget = int64(len(content)) - 1

	// Only the first file of a call may exceed the budget, so the pair takes two calls.
	rows := generateLike(t, filepath.Join(dir, "%"))
	require.Len(t, rows, 1)
	require.Equal(t, filepath.Join(dir, "a"), rows[0][colPath])

	nextRun(t)
	require.Len(t, generateLike(t, filepath.Join(dir, "%")), 2)
}

func TestHashCacheDeduplicatesByIdentity(t *testing.T) {
	resetHashState(t)
	dir := t.TempDir()

	target := filepath.Join(dir, "tool")
	content := writeFixture(t, target, machOHeader, "shared content")
	require.NoError(t, os.Symlink(target, filepath.Join(dir, "alias1")))
	require.NoError(t, os.Symlink(target, filepath.Join(dir, "alias2")))
	require.NoError(t, os.Link(target, filepath.Join(dir, "hardlink")))

	// Budget for exactly one read: every alias must be served from the first hash.
	hashByteBudget = int64(len(content))

	rows := generateLike(t, filepath.Join(dir, "%"))
	require.Len(t, rows, 4)

	byPath := rowsByPath(rows)
	for _, name := range []string{"alias1", "alias2", "tool"} {
		require.Equal(t, resolve(t, target), byPath[filepath.Join(dir, name)][colExecPath], name)
	}
	// A hard link is its own name for the same file.
	require.Equal(t, resolve(t, filepath.Join(dir, "hardlink")), byPath[filepath.Join(dir, "hardlink")][colExecPath])
	for _, row := range rows {
		require.Equal(t, sha256Hex(content), row[colExecHash])
	}
}

func TestHashByteBudgetSharedWithinWindow(t *testing.T) {
	resetHashState(t)
	dirA, dirB := t.TempDir(), t.TempDir()
	content := writeFixture(t, filepath.Join(dirA, "a"), machOHeader, "same length")
	writeFixture(t, filepath.Join(dirB, "b"), machOHeader, "same length")
	hashByteBudget = int64(len(content))

	// One call per keg, as a correlated join issues them. The first spends the cap.
	require.Len(t, generateExact(t, filepath.Join(dirA, "a")), 1)
	require.Empty(t, generateExact(t, filepath.Join(dirB, "b")))

	// Still the same run partway through the window.
	fakeNow = fakeNow.Add(hashBudgetWindow / 2)
	require.Empty(t, generateExact(t, filepath.Join(dirB, "b")))
}

func TestHashByteBudgetRenewsAfterWindow(t *testing.T) {
	resetHashState(t)
	dirA, dirB := t.TempDir(), t.TempDir()
	content := writeFixture(t, filepath.Join(dirA, "a"), machOHeader, "same length")
	b := writeFixture(t, filepath.Join(dirB, "b"), machOHeader, "same length")
	hashByteBudget = int64(len(content))

	require.Len(t, generateExact(t, filepath.Join(dirA, "a")), 1)
	require.Empty(t, generateExact(t, filepath.Join(dirB, "b")))

	nextRun(t)
	rows := generateExact(t, filepath.Join(dirB, "b"))
	require.Len(t, rows, 1)
	require.Equal(t, sha256Hex(b), rows[0][colExecHash])
}

// App bundles never draw from the budget: the server has no deferral tolerance for apps.
func TestBundleHashedWhenBudgetSpent(t *testing.T) {
	resetHashState(t)
	dir := t.TempDir()
	content := writeFixture(t, filepath.Join(dir, "tool"), machOHeader, "spend the budget")
	hashByteBudget = int64(len(content))
	require.Len(t, generateExact(t, filepath.Join(dir, "tool")), 1)

	bundleContent := []byte("bundle executable content, longer than the whole budget")
	bundlePath, _ := writeBundle(t, t.TempDir(), "Big", "Big", bundleContent)
	rows := generateExact(t, bundlePath)
	require.Len(t, rows, 1)
	require.Equal(t, sha256Hex(bundleContent), rows[0][colExecHash])
	require.Equal(t, pathTypeBundle, rows[0][colPathType])
}
