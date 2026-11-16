//go:build darwin || linux

package sshd_config

import (
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeFileInfo struct {
	mode fs.FileMode
	uid  uint32
}

func (f fakeFileInfo) Name() string       { return "sshd" }
func (f fakeFileInfo) Size() int64        { return 0 }
func (f fakeFileInfo) Mode() fs.FileMode  { return f.mode }
func (f fakeFileInfo) ModTime() time.Time { return time.Time{} }
func (f fakeFileInfo) IsDir() bool        { return f.mode.IsDir() }
func (f fakeFileInfo) Sys() any           { return &syscall.Stat_t{Uid: f.uid} }

func TestCheckTrusted(t *testing.T) {
	tests := []struct {
		name string
		mode fs.FileMode
		uid  uint32
		ok   bool
	}{
		{"root file", 0o755, 0, true},
		{"root file, group-writable", 0o775, 0, false},
		{"root file, world-writable", 0o757, 0, false},
		{"user file", 0o755, 501, false},
		{"root dir", fs.ModeDir | 0o755, 0, true},
		{"root dir, group-writable", fs.ModeDir | 0o775, 0, false},
		{"root dir, sticky world-writable", fs.ModeDir | fs.ModeSticky | 0o777, 0, true},
		{"root dir, sticky group-writable", fs.ModeDir | fs.ModeSticky | 0o775, 0, true},
		{"user dir", fs.ModeDir | 0o755, 501, false},
		{"root symlink", fs.ModeSymlink | 0o777, 0, true},
		{"user symlink", fs.ModeSymlink | 0o777, 501, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkTrusted("/x", fakeFileInfo{mode: tt.mode, uid: tt.uid})
			if tt.ok {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestResolveTrusted(t *testing.T) {
	// /bin/sh is root-owned on every supported OS, and on most Linux
	// distributions is reached through symlinks (/bin -> usr/bin, sh -> dash).
	want, err := filepath.EvalSymlinks("/bin/sh")
	require.NoError(t, err)
	got, info, err := resolveTrusted("/bin/sh")
	require.NoError(t, err)
	assert.Equal(t, want, got)
	assert.True(t, info.Mode().IsRegular())

	_, _, err = resolveTrusted("bin/sh")
	require.ErrorContains(t, err, "not an absolute path")

	_, _, err = resolveTrusted("/nonexistent-sshd-config-test/sshd")
	require.ErrorIs(t, err, fs.ErrNotExist)

	if os.Geteuid() == 0 {
		t.Skip("files created by root are trusted")
	}
	dir := t.TempDir()
	file := filepath.Join(dir, "sshd")
	require.NoError(t, os.WriteFile(file, []byte("#!/bin/sh\n"), 0o600))
	_, _, err = resolveTrusted(file)
	require.ErrorContains(t, err, "not owned by root")

	// A user-owned symlink is rejected even when its target is trusted.
	link := filepath.Join(dir, "link")
	require.NoError(t, os.Symlink("/bin/sh", link))
	_, _, err = resolveTrusted(link)
	require.ErrorContains(t, err, "not owned by root")
}
