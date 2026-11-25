package file

import (
	"archive/zip"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"path"
	"slices"
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
		// The platform markers are decoded loosely so an unexpected type
		// can't fail an upload.
		SupportedPlatforms any `plist:"CFBundleSupportedPlatforms"`
		PlatformName       any `plist:"DTPlatformName"`
		DeviceFamily       any `plist:"UIDeviceFamily"`
	}
	var platforms []fleet.InstallableDevicePlatform

	for _, f := range r.File {
		// Only the app's own plist describes the app. Embedded frameworks,
		// extensions and watch apps ship their own, and zip entry order is not
		// guaranteed, so anything below the .app directory is ignored. Zip names
		// always use forward slashes, hence path and not filepath. Some tools
		// write names as "./Payload/..." or "/Payload/...", which iOS installs.
		name := strings.TrimPrefix(path.Clean("/"+f.Name), "/")
		if matched, _ := path.Match("Payload/*.app/Info.plist", name); !matched {
			continue
		}

		archiveFile, err := f.Open()
		if err != nil {
			return nil, fmt.Errorf("could not open archive %s: %w", f.Name, err)
		}
		defer archiveFile.Close()

		rawData, err := readAllLimited(archiveFile, maxPlistSize)
		if err != nil {
			return nil, err
		}
		_, err = plist.Unmarshal(rawData, &plistData)
		if err != nil {
			return nil, err
		}

		// iOS and tvOS are separate SDK platforms with separate binaries, so an
		// archive is one or the other. The tvOS markers are checked first
		// because a tvOS app may also set LSRequiresIPhoneOS, which is set on
		// iOS/iPadOS apps and never on macOS apps.
		switch {
		case isAppleTVApp(plistData.SupportedPlatforms, plistData.PlatformName, plistData.DeviceFamily):
			platforms = []fleet.InstallableDevicePlatform{fleet.TVOSPlatform}
		case plistData.RequiresIPhoneOS:
			platforms = []fleet.InstallableDevicePlatform{fleet.IOSPlatform, fleet.IPadOSPlatform}
		}
		break
	}

	if len(platforms) == 0 {
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
		Platforms:        platforms,
	}, nil
}

// isAppleTVApp reports whether an app's Info.plist marks it as a tvOS app:
// CFBundleSupportedPlatforms includes AppleTVOS, DTPlatformName is appletvos,
// or UIDeviceFamily includes 3 (1 is iPhone, 2 is iPad).
func isAppleTVApp(supportedPlatforms, platformName, deviceFamily any) bool {
	if platforms, ok := supportedPlatforms.([]any); ok && slices.Contains(platforms, any("AppleTVOS")) {
		return true
	}
	if name, ok := platformName.(string); ok && strings.EqualFold(name, "appletvos") {
		return true
	}
	families, _ := deviceFamily.([]any)
	return slices.ContainsFunc(families, func(family any) bool {
		const appleTV = 3
		switch v := family.(type) {
		case uint64:
			return v == appleTV
		case int64:
			return v == appleTV
		}
		return false
	})
}
