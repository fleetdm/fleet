package file

import (
	"archive/zip"
	"crypto/sha256"
	"fmt"
	"io"
	"strings"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"howett.net/plist"
)

// appPlistData contains the fields we extract from an app's Info.plist.
type appPlistData struct {
	BundleID string `plist:"CFBundleIdentifier"`
	Name     string `plist:"CFBundleName"`
	Version  string `plist:"CFBundleShortVersionString"`
}

// ExtractZipAppMetadata extracts metadata from a zip archive containing a macOS .app bundle.
// It looks for .app/Contents/Info.plist inside the archive and parses the bundle identifier,
// name, and version from it.
func ExtractZipAppMetadata(tfr *fleet.TempFileReader) (*InstallerMetadata, error) {
	h := sha256.New()
	_, _ = io.Copy(h, tfr)
	if err := tfr.Rewind(); err != nil {
		return nil, fmt.Errorf("rewind reader: %w", err)
	}

	r, err := zip.OpenReader(tfr.Name())
	if err != nil {
		return nil, fmt.Errorf("open zip: %w", err)
	}
	defer r.Close()

	plistInfo, err := findAppPlistInZip(r)
	if err != nil {
		return nil, err
	}

	return &InstallerMetadata{
		Name:             plistInfo.Name,
		Version:          plistInfo.Version,
		BundleIdentifier: plistInfo.BundleID,
		SHASum:           h.Sum(nil),
		PackageIDs:       []string{plistInfo.BundleID},
	}, nil
}

// findAppPlistInZip searches for a top-level .app/Contents/Info.plist file inside a zip
// archive and parses its contents. It only matches plists at the root app level
// (e.g. "Itsycal.app/Contents/Info.plist") and ignores nested ones inside frameworks
// or plugins (e.g. "Itsycal.app/Contents/Frameworks/Sparkle.framework/.../Info.plist").
func findAppPlistInZip(r *zip.ReadCloser) (*appPlistData, error) {
	for _, f := range r.File {
		// Only match top-level: <name>.app/Contents/Info.plist
		// The path should have exactly 3 components.
		parts := strings.Split(f.Name, "/")
		if len(parts) != 3 {
			continue
		}
		if !strings.HasSuffix(parts[0], ".app") || parts[1] != "Contents" || parts[2] != "Info.plist" {
			continue
		}

		archiveFile, err := f.Open()
		if err != nil {
			return nil, fmt.Errorf("open archive file %s: %w", f.Name, err)
		}
		defer archiveFile.Close()

		rawData, err := io.ReadAll(archiveFile)
		if err != nil {
			return nil, fmt.Errorf("read plist: %w", err)
		}

		var data appPlistData
		if _, err := plist.Unmarshal(rawData, &data); err != nil {
			return nil, fmt.Errorf("unmarshal plist: %w", err)
		}

		return &data, nil
	}

	return nil, fmt.Errorf("no .app/Contents/Info.plist found in zip archive")
}

