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

func ExtractIPAMetadata(tfr *fleet.TempFileReader) (*InstallerMetadata, error) {
	h := sha256.New()
	_, _ = io.Copy(h, tfr) // writes to a hash cannot fail
	if err := tfr.Rewind(); err != nil {
		return nil, fmt.Errorf("rewind reader: %w", err)
	}

	fmt.Printf("tfr.Name(): %v\n", tfr.Name())

	r, err := zip.OpenReader(tfr.Name())
	if err != nil {
		return nil, err
	}

	var plistData struct {
		BundleID string `plist:"CFBundleIdentifier"`
		Name     string `plist:"CFBundleName"`
		Version  string `plist:"CFBundleShortVersionString"`
	}
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
			_, err = plist.Unmarshal(rawData, &plistData)
			if err != nil {
				return nil, err
			}
		}
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
