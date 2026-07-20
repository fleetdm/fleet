//go:build darwin

package apps

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/internal/homes"
)

// TestScanAppsRejectsExecutableTraversal verifies that a CFBundleExecutable
// containing path traversal in a user-writable Info.plist does not yield an
// execPath escaping the bundle — Scan would otherwise hash that path as root.
func TestScanAppsRejectsExecutableTraversal(t *testing.T) {
	home := t.TempDir()
	contents := filepath.Join(home, "Applications", "Msty.app", "Contents")
	if err := os.MkdirAll(contents, 0o755); err != nil {
		t.Fatal(err)
	}
	// "msty" is a known AI app that is very unlikely to be installed in the
	// machine's real /Applications (which scanApps also walks), so the match
	// below refers to this fixture.
	plist := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
<key>CFBundleName</key><string>Msty</string>
<key>CFBundleIdentifier</key><string>com.msty.app</string>
<key>CFBundleExecutable</key><string>../../../../../../../../etc/passwd</string>
</dict></plist>`
	if err := os.WriteFile(filepath.Join(contents, "Info.plist"), []byte(plist), 0o644); err != nil {
		t.Fatal(err)
	}

	var msty *App
	for _, a := range scanApps([]homes.Home{{Dir: home}}) {
		if a.Name == "msty" {
			found := a
			msty = &found
		}
	}
	if msty == nil {
		t.Fatal("Msty.app fixture not detected")
	}
	if msty.execPath != "" {
		t.Errorf("execPath = %q, want empty (traversal CFBundleExecutable must be rejected)", msty.execPath)
	}
}
