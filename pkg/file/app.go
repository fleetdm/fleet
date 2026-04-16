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
	BundleID     string `plist:"CFBundleIdentifier"`
	Name         string `plist:"CFBundleName"`
	Version      string `plist:"CFBundleShortVersionString"`
	IconFile     string `plist:"CFBundleIconFile"`
	IconFileName string `plist:"CFBundleIconName"`
}

// ExtractZipAppMetadata extracts metadata from a zip archive containing a macOS .app bundle.
// It looks for .app/Contents/Info.plist inside the archive and parses the bundle identifier,
// name, and version from it. It also attempts to extract the app icon from the .icns file
// referenced by CFBundleIconFile or CFBundleIconName in the plist.
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

	appName, plistInfo, err := findAppPlistInZip(r)
	if err != nil {
		return nil, err
	}

	meta := &InstallerMetadata{
		Name:             plistInfo.Name,
		Version:          plistInfo.Version,
		BundleIdentifier: plistInfo.BundleID,
		SHASum:           h.Sum(nil),
		PackageIDs:       []string{plistInfo.BundleID},
	}

	// Try to extract the app icon
	iconPNG, _ := extractIconFromZip(r, appName, plistInfo)
	meta.IconPNG = iconPNG

	return meta, nil
}

// findAppPlistInZip searches for a top-level .app/Contents/Info.plist file inside a zip
// archive and parses its contents. It returns the .app directory name and the parsed plist.
// It only matches plists at the root app level (e.g. "Itsycal.app/Contents/Info.plist")
// and ignores nested ones inside frameworks or plugins.
func findAppPlistInZip(r *zip.ReadCloser) (string, *appPlistData, error) {
	for _, f := range r.File {
		parts := strings.Split(f.Name, "/")
		if len(parts) != 3 {
			continue
		}
		if !strings.HasSuffix(parts[0], ".app") || parts[1] != "Contents" || parts[2] != "Info.plist" {
			continue
		}

		archiveFile, err := f.Open()
		if err != nil {
			return "", nil, fmt.Errorf("open archive file %s: %w", f.Name, err)
		}
		defer archiveFile.Close()

		rawData, err := io.ReadAll(archiveFile)
		if err != nil {
			return "", nil, fmt.Errorf("read plist: %w", err)
		}

		var data appPlistData
		if _, err := plist.Unmarshal(rawData, &data); err != nil {
			return "", nil, fmt.Errorf("unmarshal plist: %w", err)
		}

		return parts[0], &data, nil
	}

	return "", nil, fmt.Errorf("no .app/Contents/Info.plist found in zip archive")
}

// extractIconFromZip attempts to find and extract a PNG icon from the app's .icns file
// inside the zip. It uses CFBundleIconFile or CFBundleIconName from the plist to locate
// the .icns file under Contents/Resources/, then extracts the largest embedded PNG.
func extractIconFromZip(r *zip.ReadCloser, appName string, plistInfo *appPlistData) ([]byte, error) {
	// Determine the icns filename from the plist
	iconName := plistInfo.IconFile
	if iconName == "" {
		iconName = plistInfo.IconFileName
	}
	if iconName == "" {
		iconName = "AppIcon"
	}
	// Add .icns extension if not present
	if !strings.HasSuffix(iconName, ".icns") {
		iconName += ".icns"
	}

	// Look for the icns file at <App>.app/Contents/Resources/<iconName>
	targetPath := appName + "/Contents/Resources/" + iconName

	for _, f := range r.File {
		if f.Name != targetPath {
			continue
		}

		archiveFile, err := f.Open()
		if err != nil {
			return nil, fmt.Errorf("open icon file: %w", err)
		}
		defer archiveFile.Close()

		icnsData, err := io.ReadAll(archiveFile)
		if err != nil {
			return nil, fmt.Errorf("read icon file: %w", err)
		}

		return ExtractPNGFromICNS(icnsData)
	}

	return nil, fmt.Errorf("icon file %s not found in zip", targetPath)
}

