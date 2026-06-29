package update

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRemoveTarget(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	const target = "osqueryd"
	info := TargetInfo{
		Platform:             "macos-app",
		Channel:              "5.22.1",
		TargetFile:           "osqueryd.app.tar.gz",
		ExtractedExecSubPath: []string{"osquery.app", "Contents", "MacOS", "osqueryd"},
	}
	u := &Updater{opt: Options{
		RootDirectory: tmpDir,
		Targets:       Targets{target: info},
	}}

	archivePath, execPath, dirPath := LocalTargetPaths(tmpDir, target, info)
	hashPath := archivePath + ".sha512"

	// Lay down the archive, cached hash and the extracted (corrupt) binary.
	require.NoError(t, os.MkdirAll(filepath.Dir(archivePath), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Dir(execPath), 0o755))
	require.NoError(t, os.WriteFile(archivePath, []byte("archive"), 0o644))
	require.NoError(t, os.WriteFile(hashPath, []byte("deadbeef"), 0o644))
	require.NoError(t, os.WriteFile(execPath, []byte("truncated"), 0o755)) // #nosec G306

	require.NoError(t, u.RemoveTarget(target))

	// All three artifacts should be gone so the next Get re-downloads.
	for _, p := range []string{archivePath, hashPath, dirPath, execPath} {
		_, err := os.Stat(p)
		require.ErrorIs(t, err, os.ErrNotExist, "expected %q removed", p)
	}
}

func TestCheckExec(t *testing.T) {
	t.Parallel()

	platform := map[string]string{
		"darwin":  "macos",
		"linux":   "linux",
		"windows": "windows",
	}[runtime.GOOS]
	require.NotEmpty(t, platform, "unsupported test platform %s", runtime.GOOS)

	const target = "osqueryd"

	t.Run("corrupt binary surfaces a corruption error", func(t *testing.T) {
		u := &Updater{opt: Options{
			RootDirectory: t.TempDir(),
			Targets: Targets{target: TargetInfo{
				Platform:   platform,
				Channel:    "stable",
				TargetFile: "osqueryd",
				CustomCheckExec: func(string) error {
					return fmt.Errorf("fork/exec: %w", syscall.ENOEXEC)
				},
			}},
		}}
		err := u.CheckExec(target)
		require.Error(t, err)
	})

	t.Run("healthy binary passes", func(t *testing.T) {
		u := &Updater{opt: Options{
			RootDirectory: t.TempDir(),
			Targets: Targets{target: TargetInfo{
				Platform:        platform,
				Channel:         "stable",
				TargetFile:      "osqueryd",
				CustomCheckExec: func(string) error { return nil },
			}},
		}}
		require.NoError(t, u.CheckExec(target))
	})
}

// TestCheckExecRealBinary exercises the default `--help` exec branch (no
// CustomCheckExec) against real files on disk. This is the branch osqueryd
// actually uses, so it must run an actual executable rather than a stub.
func TestCheckExecRealBinary(t *testing.T) {
	t.Parallel()

	platform := map[string]string{
		"darwin":  "macos",
		"linux":   "linux",
		"windows": "windows",
	}[runtime.GOOS]
	require.NotEmpty(t, platform, "unsupported test platform %s", runtime.GOOS)

	const target = "osqueryd"
	info := TargetInfo{
		Platform:   platform,
		Channel:    "stable",
		TargetFile: "osqueryd",
	}
	root := t.TempDir()
	u := &Updater{opt: Options{RootDirectory: root, Targets: Targets{target: info}}}

	_, execPath, _ := LocalTargetPaths(root, target, info)
	require.NoError(t, os.MkdirAll(filepath.Dir(execPath), 0o755))

	t.Run("healthy binary passes --help", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("the shell-script stand-in binary is unix-only")
		}
		// A script that exits 0 for any args (including --help).
		require.NoError(t, os.WriteFile(execPath, []byte("#!/bin/sh\nexit 0\n"), 0o755)) // #nosec G306
		require.NoError(t, u.CheckExec(target))
	})

	t.Run("corrupt binary fails the exec check", func(t *testing.T) {
		// Non-executable garbage (no shebang, not a valid Mach-O/ELF/PE) fails to
		// fork/exec with a format error on every platform: ENOEXEC ("exec format
		// error") on Linux/macOS, ERROR_BAD_EXE_FORMAT on Windows.
		require.NoError(t, os.WriteFile(execPath, []byte("\x00\x01\x02not a binary"), 0o755)) // #nosec G306
		err := u.CheckExec(target)
		require.Error(t, err)
	})
}
