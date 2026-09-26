package file

import (
	"archive/zip"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"howett.net/plist"
)

// bundleInfo is the subset of an Info.plist that Fleet reads.
type bundleInfo struct {
	BundleID         string `plist:"CFBundleIdentifier"`
	Name             string `plist:"CFBundleName"`
	Version          string `plist:"CFBundleShortVersionString"`
	RequiresIPhoneOS bool   `plist:"LSRequiresIPhoneOS"`
}

// isMainAppInfoPlist reports whether name is the Info.plist of the .app bundle
// at the root of an ipa payload, e.g. "Payload/Example.app/Info.plist".
func isMainAppInfoPlist(name string) bool {
	parts := strings.Split(name, "/")
	return len(parts) == 3 && parts[0] == "Payload" &&
		strings.HasSuffix(parts[1], ".app") && parts[2] == "Info.plist"
}

// ExtractZIPMetadata extracts the metadata from a zip file for an Apple app
func ExtractZIPMetadata(tfr *fleet.TempFileReader) (*InstallerMetadata, error) {
	h := sha256.New()
	_, _ = io.Copy(h, tfr) // writes to a hash cannot fail
	if err := tfr.Rewind(); err != nil {
		return nil, fmt.Errorf("rewind reader: %w", err)
	}

	r, err := zip.OpenReader(tfr.Name())
	if err != nil {
		return nil, err
	}

	// Frameworks, app extensions, and CocoaPods resource bundles each ship an
	// Info.plist describing themselves, so identity comes from the app bundle's
	// own plist. Archives with no Payload/*.app/Info.plist keep the previous
	// behavior of merging every plist in archive order.
	var mainAppData, mergedData bundleInfo
	var hasInfoPlist, isIPA, haveMainApp bool

	for _, f := range r.File {
		if strings.Contains(f.Name, "Info.plist") {
			// Get data from plist file
			archiveFile, err := f.Open()
			if err != nil {
				return nil, fmt.Errorf("could not open archive %s: %w", f.Name, err)
			}
			defer archiveFile.Close()

			rawData, err := io.ReadAll(archiveFile)
			if err != nil {
				return nil, err
			}
			target := &mergedData
			if isMainAppInfoPlist(f.Name) {
				target, haveMainApp = &mainAppData, true
			}
			_, err = plist.Unmarshal(rawData, target)
			if err != nil {
				return nil, err
			}

			hasInfoPlist = true
			// LSRequiresIPhoneOS is set on iOS/iPadOS apps and never on macOS
			// apps, so it is probably an .ipa. Every plist is still checked
			// because only some bundles in the archive carry the key.
			if target.RequiresIPhoneOS {
				isIPA = true
			}
		}
	}

	plistData := mergedData
	if haveMainApp {
		plistData = mainAppData
	}

	if !hasInfoPlist || !isIPA {
		// non Apple file formats based on zip are not supported (msix)
		return nil, ErrInvalidType
	}
	if plistData.BundleID == "" {
		return nil, errors.New("couldn't find bundle identifier for in-house app")
	}

	return &InstallerMetadata{
		BundleIdentifier: plistData.BundleID,
		SHASum:           h.Sum(nil),
		PackageIDs:       []string{plistData.BundleID},
		Name:             plistData.Name,
		Version:          plistData.Version,
	}, nil
}
