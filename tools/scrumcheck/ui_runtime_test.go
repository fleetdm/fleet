package main

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

// TestSetUIRuntimeDirValidation verifies dev UI path validation rules.
func TestSetUIRuntimeDirValidation(t *testing.T) {
	t.Parallel()

	if err := setUIRuntimeDir(""); err != nil {
		t.Fatalf("set empty ui dir: %v", err)
	}
	t.Cleanup(func() { _ = setUIRuntimeDir("") })

	tmp := t.TempDir()
	if err := setUIRuntimeDir(tmp); err == nil {
		t.Fatalf("expected validation error for missing index/assets")
	}
	if err := os.WriteFile(filepath.Join(tmp, "index.html"), []byte("<html></html>"), 0o600); err != nil {
		t.Fatalf("write index.html: %v", err)
	}
	if err := os.Mkdir(filepath.Join(tmp, "assets"), 0o700); err != nil {
		t.Fatalf("mkdir assets: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "assets", "app.js"), []byte("console.log('dev');"), 0o600); err != nil {
		t.Fatalf("write app.js: %v", err)
	}
	if err := setUIRuntimeDir(tmp); err != nil {
		t.Fatalf("set valid ui dir: %v", err)
	}
}

// TestActiveUIAssetsDevOverride verifies activeUIAssets switches to dev files.
func TestActiveUIAssetsDevOverride(t *testing.T) {
	t.Parallel()
	_ = setUIRuntimeDir("")
	t.Cleanup(func() { _ = setUIRuntimeDir("") })

	tmp := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmp, "index.html"), []byte("<html></html>"), 0o600); err != nil {
		t.Fatalf("write index.html: %v", err)
	}
	if err := os.Mkdir(filepath.Join(tmp, "assets"), 0o700); err != nil {
		t.Fatalf("mkdir assets: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "assets", "app.css"), []byte("body{color:red;}"), 0o600); err != nil {
		t.Fatalf("write app.css: %v", err)
	}

	if err := setUIRuntimeDir(tmp); err != nil {
		t.Fatalf("set ui dir: %v", err)
	}
	raw, err := fs.ReadFile(activeUIAssets(), "app.css")
	if err != nil {
		t.Fatalf("read active asset: %v", err)
	}
	if string(raw) != "body{color:red;}" {
		t.Fatalf("unexpected active dev asset content: %q", string(raw))
	}
}
