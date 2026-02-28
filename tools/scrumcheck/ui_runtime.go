package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var (
	uiRuntimeMu  sync.RWMutex
	uiRuntimeDir string
)

// setUIRuntimeDir configures an optional frontend dev directory.
//
// When provided, scrumcheck serves `index.html` and `/assets/*` from this
// directory instead of embedded files. Empty value keeps embedded production
// assets.
func setUIRuntimeDir(dir string) error {
	trimmed := strings.TrimSpace(dir)
	if trimmed == "" {
		uiRuntimeMu.Lock()
		uiRuntimeDir = ""
		uiRuntimeMu.Unlock()
		return nil
	}

	abs, err := filepath.Abs(trimmed)
	if err != nil {
		return fmt.Errorf("resolve path: %w", err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		return fmt.Errorf("stat %q: %w", abs, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("%q is not a directory", abs)
	}
	if _, err := os.Stat(filepath.Join(abs, "index.html")); err != nil {
		return fmt.Errorf("missing %q", filepath.Join(abs, "index.html"))
	}
	if _, err := os.Stat(filepath.Join(abs, "assets")); err != nil {
		return fmt.Errorf("missing %q", filepath.Join(abs, "assets"))
	}

	// Store normalized absolute path so all readers use one canonical value.
	uiRuntimeMu.Lock()
	uiRuntimeDir = abs
	uiRuntimeMu.Unlock()
	return nil
}

// uiRuntimeDirValue returns the configured frontend runtime directory.
func uiRuntimeDirValue() string {
	uiRuntimeMu.RLock()
	defer uiRuntimeMu.RUnlock()
	return uiRuntimeDir
}
