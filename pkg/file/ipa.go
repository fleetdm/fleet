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

	var plistData struct {
		BundleID         string `plist:"CFBundleIdentifier"`
		Name             string `plist:"CFBundleName"`
		Version          string `plist:"CFBundleShortVersionString"`
		RequiresIPhoneOS bool   `plist:"LSRequiresIPhoneOS"`
	}
	var hasInfoPlist, isIPA bool

	for _, f := range r.File {
		// Matches any Info.plist and the last wins, so a nested framework or
		// extension plist can override the app's own plist.
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
			_, err = plist.Unmarshal(rawData, &plistData)
			if err != nil {
				return nil, err
			}

			hasInfoPlist = true
			// LSRequiresIPhoneOS is set on iOS/iPadOS apps and never on macOS
			// apps, so it is probably an .ipa
			if plistData.RequiresIPhoneOS {
				isIPA = true
			}
		}
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
