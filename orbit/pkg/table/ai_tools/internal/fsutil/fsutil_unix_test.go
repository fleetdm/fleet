//go:build !windows

package fsutil

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

// TestRejectsSymlink verifies the read helpers refuse a symlink instead of
// following it. Running as root, following a symlink a low-priv user planted
// would disclose the content/hash of a root-only target.
func TestRejectsSymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	if err := os.WriteFile(target, []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}

	if got := SHA256(link); got != "" {
		t.Errorf("SHA256(symlink) = %q, want \"\" (must not follow symlink)", got)
	}
	if Exists(link) {
		t.Error("Exists(symlink) = true, want false")
	}
	if _, err := ReadFileBounded(link); err == nil {
		t.Error("ReadFileBounded(symlink) err = nil, want error")
	}
	// A regular file is still read/hashed normally.
	if got := SHA256(target); got == "" {
		t.Error("SHA256(regular file) = \"\", want a hash")
	}
}

// TestFIFODoesNotBlock verifies the read helpers refuse a FIFO and return
// promptly. Opening a FIFO O_RDONLY without O_NONBLOCK blocks until a writer
// appears; as root this is a trivial local DoS, so it must never happen.
func TestFIFODoesNotBlock(t *testing.T) {
	dir := t.TempDir()
	fifo := filepath.Join(dir, "pipe")
	if err := syscall.Mkfifo(fifo, 0o600); err != nil {
		t.Skipf("mkfifo unsupported: %v", err)
	}

	done := make(chan struct{})
	go func() {
		_ = SHA256(fifo)
		_, _ = ReadFileBounded(fifo)
		_ = Exists(fifo)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("read helpers blocked on a FIFO (DoS): did not return within 5s")
	}

	if got := SHA256(fifo); got != "" {
		t.Errorf("SHA256(fifo) = %q, want \"\"", got)
	}
	if Exists(fifo) {
		t.Error("Exists(fifo) = true, want false")
	}
}
