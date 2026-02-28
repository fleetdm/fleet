package main

import (
	"io/fs"
	"testing"
)

// mustReadEmbeddedUIAsset reads one embedded UI asset file for test assertions.
func mustReadEmbeddedUIAsset(name string) []byte {
	raw, err := fs.ReadFile(embeddedUIAssets(), name)
	if err != nil {
		panic(err)
	}
	return raw
}

// TestEmbeddedUIAssetsAvailable verifies CSS and JS assets are embedded.
func TestEmbeddedUIAssetsAvailable(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"app.css", "app.js"} {
		raw, err := fs.ReadFile(embeddedUIAssets(), name)
		if err != nil {
			t.Fatalf("read embedded asset %s: %v", name, err)
		}
		if len(raw) == 0 {
			t.Fatalf("embedded asset %s is empty", name)
		}
	}
}

