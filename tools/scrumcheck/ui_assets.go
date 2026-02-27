package main

import (
	"embed"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
)

// uiAssets contains static frontend assets served from /assets/*.
//
//go:embed ui/assets/*
var uiAssets embed.FS

var (
	uiAssetsSubFS     fs.FS
	uiAssetsSubFSOnce sync.Once
)

// embeddedUIAssets returns the embedded /ui/assets subtree.
func embeddedUIAssets() fs.FS {
	uiAssetsSubFSOnce.Do(func() {
		sub, err := fs.Sub(uiAssets, "ui/assets")
		if err != nil {
			panic("embed ui assets subfs: " + err.Error())
		}
		uiAssetsSubFS = sub
	})
	return uiAssetsSubFS
}

// activeUIAssets resolves the currently active assets filesystem.
//
// In production this is the embedded `/ui/assets` subtree. In dev mode
// (`-ui-dev-dir`), assets are loaded from `<ui-dev-dir>/assets`.
func activeUIAssets() fs.FS {
	if dir := uiRuntimeDirValue(); dir != "" {
		return os.DirFS(filepath.Join(dir, "assets"))
	}
	return embeddedUIAssets()
}
