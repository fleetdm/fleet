//go:build darwin

package file

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"howett.net/plist"
)

// ExtractDMGMetadata extracts metadata from a DMG file by mounting it with hdiutil,
// finding the .app/Contents/Info.plist, and parsing the bundle identifier, name, and version.
// This only works on macOS where hdiutil is available.
func ExtractDMGMetadata(tfr *fleet.TempFileReader) (*InstallerMetadata, error) {
	h := sha256.New()
	_, _ = io.Copy(h, tfr)
	if err := tfr.Rewind(); err != nil {
		return nil, fmt.Errorf("rewind reader: %w", err)
	}

	mountPoint, err := os.MkdirTemp("", "dmg_mount_")
	if err != nil {
		return nil, fmt.Errorf("create temp mount point: %w", err)
	}
	defer os.RemoveAll(mountPoint)

	// Mount the DMG
	cmd := exec.Command("hdiutil", "attach", "-plist", "-nobrowse", "-readonly", "-mountpoint", mountPoint, tfr.Name())
	if out, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("hdiutil attach: %w: %s", err, string(out))
	}
	defer exec.Command("hdiutil", "detach", mountPoint, "-quiet").Run() //nolint:errcheck

	// Find the top-level .app bundle
	entries, err := os.ReadDir(mountPoint)
	if err != nil {
		return nil, fmt.Errorf("read mount point: %w", err)
	}

	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".app") || !entry.IsDir() {
			continue
		}

		plistPath := filepath.Join(mountPoint, entry.Name(), "Contents", "Info.plist")
		rawData, err := os.ReadFile(plistPath)
		if err != nil {
			continue
		}

		var data appPlistData
		if _, err := plist.Unmarshal(rawData, &data); err != nil {
			continue
		}

		if data.BundleID == "" {
			continue
		}

		return &InstallerMetadata{
			Name:             data.Name,
			Version:          data.Version,
			BundleIdentifier: data.BundleID,
			SHASum:           h.Sum(nil),
			PackageIDs:       []string{data.BundleID},
		}, nil
	}

	return nil, fmt.Errorf("no .app bundle with Info.plist found in DMG")
}
