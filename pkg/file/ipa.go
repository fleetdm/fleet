package file

import (
	"archive/zip"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"howett.net/plist"
)

// appleTVDeviceFamily is the UIDeviceFamily value Apple uses for tvOS
// (1 = iPhone, 2 = iPad, 3 = Apple TV).
const appleTVDeviceFamily = 3

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
		BundleID           string   `plist:"CFBundleIdentifier"`
		Name               string   `plist:"CFBundleName"`
		Version            string   `plist:"CFBundleShortVersionString"`
		RequiresIPhoneOS   bool     `plist:"LSRequiresIPhoneOS"`
		SupportedPlatforms []string `plist:"CFBundleSupportedPlatforms"`
		DeviceFamily       []int    `plist:"UIDeviceFamily"`
	}
	var hasInfoPlist, isIPhoneOSApp, isTvOSApp bool

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
				isIPhoneOSApp = true
			}
			// tvOS apps don't set LSRequiresIPhoneOS, so they are identified by
			// their platform or device family instead. The signals are OR-ed
			// across every plist in the archive rather than letting the last one
			// win: an iOS app can't embed a tvOS binary (they are separate Apple
			// SDK platforms), so any tvOS marker means the whole archive is tvOS.
			if slices.Contains(plistData.SupportedPlatforms, "AppleTVOS") ||
				slices.Contains(plistData.DeviceFamily, appleTVDeviceFamily) {
				isTvOSApp = true
			}
		}
	}

	if !hasInfoPlist || (!isIPhoneOSApp && !isTvOSApp) {
		// non Apple file formats based on zip are not supported (msix)
		return nil, ErrInvalidType
	}
	if plistData.BundleID == "" {
		return nil, errors.New("couldn't find bundle identifier for in-house app")
	}

	// A single .ipa targets one Apple SDK platform. iOS apps run on both
	// iPhone and iPad, so they fan out to two Fleet platforms; a tvOS app is
	// only ever tvOS.
	platforms := []fleet.InstallableDevicePlatform{fleet.IOSPlatform, fleet.IPadOSPlatform}
	if isTvOSApp && !isIPhoneOSApp {
		platforms = []fleet.InstallableDevicePlatform{fleet.TVOSPlatform}
	}

	return &InstallerMetadata{
		BundleIdentifier: plistData.BundleID,
		SHASum:           h.Sum(nil),
		PackageIDs:       []string{plistData.BundleID},
		Name:             plistData.Name,
		Version:          plistData.Version,
		Platforms:        platforms,
	}, nil
}
