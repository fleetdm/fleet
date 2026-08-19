package apps

import (
	"os"
	"path/filepath"
	"testing"
)

const chatGPTPFN = "OpenAI.ChatGPT-Desktop_1.2026.190.0_arm64__2p2nqsd0c76g0"

const chatGPTManifest = `<?xml version="1.0" encoding="utf-8"?>
<Package xmlns="http://schemas.microsoft.com/appx/manifest/foundation/windows10">
  <Identity Name="OpenAI.ChatGPT-Desktop" Publisher="CN=OpenAI, O=OpenAI OpCo, LLC, C=US" Version="1.2026.190.0" />
  <Properties>
    <DisplayName>ChatGPT</DisplayName>
    <PublisherDisplayName>OpenAI</PublisherDisplayName>
  </Properties>
</Package>`

// mkPackageDir creates <root>/<name>, writing an AppxManifest.xml when manifest
// is non-empty.
func mkPackageDir(t *testing.T, root, name, manifest string) string {
	t.Helper()
	dir := filepath.Join(root, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	if manifest != "" {
		if err := os.WriteFile(filepath.Join(dir, "AppxManifest.xml"), []byte(manifest), 0o644); err != nil {
			t.Fatalf("write manifest: %v", err)
		}
	}
	return dir
}

func TestScanAppxDirsInstallRoot(t *testing.T) {
	root := t.TempDir()
	dir := mkPackageDir(t, root, chatGPTPFN, chatGPTManifest)
	// Packages that must not produce rows.
	mkPackageDir(t, root, "Microsoft.WindowsCalculator_11.2210.0.0_x64__8wekyb3d8bbwe", "")
	mkPackageDir(t, root, "Microsoft.VCLibs.140.00_14.0.30704.0_x64__8wekyb3d8bbwe", "")
	mkPackageDir(t, root, "NVIDIACorp.NVIDIAControlPanel_8.1.966.0_x64__56jybvy8sckqj", "")

	c := newAppCollector()
	scanAppxDirs(c, root, nil)

	got := c.apps()
	if len(got) != 1 {
		t.Fatalf("got %d apps, want 1: %+v", len(got), got)
	}
	want := App{
		Name: "chatgpt", DisplayName: "ChatGPT", Vendor: "OpenAI", Version: "1.2026.190.0",
		Path: dir, Scope: "system", PlatformSource: "appx",
	}
	if got[0] != want {
		t.Errorf("got %+v\nwant %+v", got[0], want)
	}
}

// TestScanAppxDirsResourcePackageSkipped guards against a satellite package
// winning the dedup race and contributing the wrong install path. That race is
// real: os.ReadDir returns sorted names, and "..._neutral_scale-100_pub" sorts
// before "..._x64__pub", so on an x64 host the satellite is seen first.
func TestScanAppxDirsResourcePackageSkipped(t *testing.T) {
	const resourceManifest = `<?xml version="1.0" encoding="utf-8"?>
<Package xmlns="http://schemas.microsoft.com/appx/manifest/foundation/windows10">
  <Identity Name="OpenAI.ChatGPT-Desktop" Publisher="CN=OpenAI, O=OpenAI OpCo, LLC, C=US" Version="1.2026.190.0" />
  <Properties>
    <DisplayName>ChatGPT</DisplayName>
    <PublisherDisplayName>OpenAI</PublisherDisplayName>
    <ResourcePackage>true</ResourcePackage>
  </Properties>
</Package>`

	cases := []struct {
		name     string
		pfn      string
		manifest string
	}{
		{
			// Modern convention: caught by the name alone, no manifest needed.
			name: "split prefix",
			pfn:  "OpenAI.ChatGPT-Desktop_1.2026.190.0_neutral_split.scale-100_2p2nqsd0c76g0",
		},
		{
			// Older convention: the resource id carries no "split." prefix, so only
			// the manifest identifies it as a satellite.
			name:     "bare resource id with manifest",
			pfn:      "OpenAI.ChatGPT-Desktop_1.2026.190.0_neutral_scale-100_2p2nqsd0c76g0",
			manifest: resourceManifest,
		},
		{
			name:     "bare language resource id with manifest",
			pfn:      "OpenAI.ChatGPT-Desktop_1.2026.190.0_neutral_language-en_2p2nqsd0c76g0",
			manifest: resourceManifest,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := t.TempDir()
			mkPackageDir(t, root, c.pfn, c.manifest)

			col := newAppCollector()
			scanAppxDirs(col, root, nil)
			if got := col.apps(); len(got) != 0 {
				t.Errorf("got %d apps from a resource package alone, want 0: %+v", len(got), got)
			}
		})
	}
}

// TestScanAppxDirsResourcePackageLosesToMainPackage is the case the satellite
// filter exists for: both packages present, the satellite sorted first, and the
// app still reported at its real install path.
func TestScanAppxDirsResourcePackageLosesToMainPackage(t *testing.T) {
	root := t.TempDir()
	mkPackageDir(t, root, "OpenAI.ChatGPT-Desktop_1.2026.190.0_neutral_split.scale-100_2p2nqsd0c76g0", "")
	mainDir := mkPackageDir(t, root, "OpenAI.ChatGPT-Desktop_1.2026.190.0_x64__2p2nqsd0c76g0", chatGPTManifest)

	c := newAppCollector()
	scanAppxDirs(c, root, nil)

	got := c.apps()
	if len(got) != 1 {
		t.Fatalf("got %d apps, want 1: %+v", len(got), got)
	}
	if got[0].Path != mainDir {
		t.Errorf("got path %q, want the main package at %q", got[0].Path, mainDir)
	}
}

// TestScanAppxDirsScopeFromUserDir covers attribution: a package with per-user
// state belongs to a user, not to the machine.
func TestScanAppxDirsScopeFromUserDir(t *testing.T) {
	root := t.TempDir()
	mkPackageDir(t, root, chatGPTPFN, chatGPTManifest)

	userPkgs := t.TempDir()
	mkPackageDir(t, userPkgs, "OpenAI.ChatGPT-Desktop_2p2nqsd0c76g0", "")

	c := newAppCollector()
	scanAppxDirs(c, root, []string{userPkgs})

	got := c.apps()
	if len(got) != 1 {
		t.Fatalf("got %d apps, want 1: %+v", len(got), got)
	}
	if got[0].Scope != "user" {
		t.Errorf("got scope %q, want \"user\"", got[0].Scope)
	}
	if got[0].PlatformSource != "appx" {
		t.Errorf("got source %q, want \"appx\": the install root is richer and is read first", got[0].PlatformSource)
	}
}

// TestScanAppxDirsNoRowsFromUserDirsAlone pins down that per-user package
// directories never produce a row. That directory records that a package once
// ran for a user and survives uninstall, so sourcing rows from it would report
// removed apps forever, with no version or path to check them against. An
// unreadable install root must report nothing rather than something unverified.
func TestScanAppxDirsNoRowsFromUserDirsAlone(t *testing.T) {
	userPkgs := t.TempDir()
	mkPackageDir(t, userPkgs, "OpenAI.ChatGPT-Desktop_2p2nqsd0c76g0", "")
	mkPackageDir(t, userPkgs, "Microsoft.WindowsCalculator_8wekyb3d8bbwe", "")

	for _, installRoot := range []string{
		"",
		filepath.Join(t.TempDir(), "does-not-exist"),
		t.TempDir(), // readable but empty
	} {
		c := newAppCollector()
		scanAppxDirs(c, installRoot, []string{userPkgs})

		if got := c.apps(); len(got) != 0 {
			t.Errorf("installRoot=%q: got %d apps, want 0: %+v", installRoot, len(got), got)
		}
	}
}

// TestScanAppxDirsStaleUserDirIsNotAnInstall is the regression test for the
// phantom row: a healthy host where the app was uninstalled but Windows left its
// per-user state directory behind. The install root is authoritative, and it
// does not list the package.
func TestScanAppxDirsStaleUserDirIsNotAnInstall(t *testing.T) {
	root := t.TempDir()
	mkPackageDir(t, root, "Microsoft.WindowsCalculator_11.2210.0.0_x64__8wekyb3d8bbwe", "")
	mkPackageDir(t, root, "Microsoft.VCLibs.140.00_14.0.30704.0_x64__8wekyb3d8bbwe", "")

	userPkgs := t.TempDir()
	mkPackageDir(t, userPkgs, "OpenAI.ChatGPT-Desktop_2p2nqsd0c76g0", "") // left over from an uninstall

	c := newAppCollector()
	scanAppxDirs(c, root, []string{userPkgs})

	if got := c.apps(); len(got) != 0 {
		t.Errorf("got %d apps, want 0: a leftover per-user directory is not an install: %+v", len(got), got)
	}
}

// TestScanAppxDirsMissingManifest covers a manifest that is unreadable or
// carries no usable display name (including a MUI indirect string), which must
// cost only the vendor and the friendly name, never the row: the display name
// falls back to the package identity name.
func TestScanAppxDirsMissingManifest(t *testing.T) {
	const indirectManifest = `<?xml version="1.0" encoding="utf-8"?>
<Package xmlns="http://schemas.microsoft.com/appx/manifest/foundation/windows10">
  <Properties>
    <DisplayName>@{OpenAI.ChatGPT-Desktop_1.2026.190.0_arm64__2p2nqsd0c76g0?ms-resource://OpenAI.ChatGPT-Desktop/Resources/AppName}</DisplayName>
  </Properties>
</Package>`

	for _, manifest := range []string{"", "not xml at all <<<", "<Package><Identity/></Package>", indirectManifest} {
		root := t.TempDir()
		mkPackageDir(t, root, chatGPTPFN, manifest)

		c := newAppCollector()
		scanAppxDirs(c, root, nil)

		got := c.apps()
		if len(got) != 1 {
			t.Fatalf("manifest=%q: got %d apps, want 1", manifest, len(got))
		}
		if got[0].Version != "1.2026.190.0" {
			t.Errorf("manifest=%q: got version %q, want it from the package name", manifest, got[0].Version)
		}
		if got[0].Vendor != "" {
			t.Errorf("manifest=%q: got vendor %q, want empty", manifest, got[0].Vendor)
		}
		if got[0].DisplayName != "OpenAI.ChatGPT-Desktop" {
			t.Errorf("manifest=%q: got display name %q, want the package identity name", manifest, got[0].DisplayName)
		}
	}
}

// TestScanAppxDirsSharedWithUninstallScan covers the ordering the Windows
// collector depends on: an app already found in the uninstall keys keeps that
// richer entry instead of being replaced by a package directory.
func TestScanAppxDirsSharedWithUninstallScan(t *testing.T) {
	root := t.TempDir()
	mkPackageDir(t, root, chatGPTPFN, chatGPTManifest)

	c := newAppCollector()
	c.add(appCandidate{
		MatchTokens: []string{"ChatGPT"},
		Version:     "1.2026.190",
		Path:        `C:\Users\alice\AppData\Local\Programs\ChatGPT`,
		Scope:       "user",
		Source:      "registry",
	})
	scanAppxDirs(c, root, nil)

	got := c.apps()
	if len(got) != 1 {
		t.Fatalf("got %d apps, want 1: %+v", len(got), got)
	}
	if got[0].PlatformSource != "registry" {
		t.Errorf("got source %q, want \"registry\"", got[0].PlatformSource)
	}
}
