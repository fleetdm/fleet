package settings

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadWriteTextFileRejections(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home dir")
	}

	// Wrong extension (path is under home so it clears the home guard first).
	bad := filepath.Join(home, "notes.json")
	if _, err := ReadTextFile(bad); err == nil {
		t.Error("ReadTextFile should reject non-yaml extension")
	}
	if err := WriteTextFile(bad, "x"); err == nil {
		t.Error("WriteTextFile should reject non-yaml extension")
	}

	// Outside $HOME.
	if _, err := ReadTextFile("/etc/anything.yml"); err == nil {
		t.Error("ReadTextFile should reject path outside home")
	}
	if err := WriteTextFile("/etc/anything.yml", "x"); err == nil {
		t.Error("WriteTextFile should reject path outside home")
	}
}

func TestReadWriteTextFileRoundTrip(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home dir")
	}
	dir, err := os.MkdirTemp(home, ".hangar-test-*")
	if err != nil {
		t.Skip("cannot create temp dir under home: " + err.Error())
	}
	t.Cleanup(func() { os.RemoveAll(dir) })

	p := filepath.Join(dir, "config.yml")
	const body = "version: \"3\"\n"
	if err := WriteTextFile(p, body); err != nil {
		t.Fatalf("WriteTextFile: %v", err)
	}
	got, err := ReadTextFile(p)
	if err != nil {
		t.Fatalf("ReadTextFile: %v", err)
	}
	if got != body {
		t.Errorf("round-trip mismatch: got %q want %q", got, body)
	}
}

func TestOpenURLRejections(t *testing.T) {
	for _, bad := range []string{"ftp://x", "file:///etc/passwd", "javascript:alert(1)", "x.com"} {
		if err := OpenURL(bad); err == nil {
			t.Errorf("OpenURL(%q) should be rejected", bad)
		}
	}
}

func TestOpenPathRejections(t *testing.T) {
	// Outside home.
	if err := OpenPath("/etc/passwd", false); err == nil {
		t.Error("OpenPath should reject path outside home")
	}
	// Disallowed extension under home.
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home dir")
	}
	if err := OpenPath(filepath.Join(home, "thing.sh"), false); err == nil {
		t.Error("OpenPath should reject .sh file")
	}
}
