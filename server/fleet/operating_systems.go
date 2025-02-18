package fleet

import (
	"fmt"
	"strings"

	"github.com/Masterminds/semver/v3"
)

// OperatingSystem is an operating system uniquely identified according to its name and version.
type OperatingSystem struct {
	ID uint `json:"id" db:"id"`
	// Name is the name of the operating system, e.g., "Debian/GNU Linus", "Ubuntu", or "Microsoft Windows 11 Enterprise"
	Name string `json:"name" db:"name"`
	// Version is the version of the operating system, e.g., "14.1.2"(macOS), "22.04 LTS"(Ubuntu)
	// On Windows, this is the build number, which will always match KernelVersion e.g., "10.0.19042.1348"
	Version string `json:"version" db:"version"`
	// Arch is the architecture of the operating system, e.g., "x86_64" or "64-bit"
	Arch string `json:"arch,omitempty" db:"arch"`
	// KernelVersion is the kernel version of the operating system, e.g., "5.10.76-linuxkit"
	// On Windows, this is the build number, which will always match Version e.g., "10.0.19042.1348"
	KernelVersion string `json:"kernel_version,omitempty" db:"kernel_version"`
	// Platform is the platform of the operating system, e.g., "darwin" or "rhel"
	Platform string `json:"platform" db:"platform"`
	// DisplayVersion is the display version of a Windows operating system, e.g. "22H2"
	DisplayVersion string `json:"display_version" db:"display_version"`
	// OSVersionID is a unique Name/Version combination for the operating system
	OSVersionID uint `json:"os_version_id" db:"os_version_id"`
}

// IsWindows returns whether the OperatingSystem record references a Windows OS
func (os OperatingSystem) IsWindows() bool {
	return strings.ToLower(os.Platform) == "windows"
}

var macOSNudgeLastVersion = semver.MustParse("14")

// RequiresNudge returns whether the target platform is darwin and
// below version 14. Starting at macOS 14 nudge is no longer required,
// as the mechanism to notify users about updates is built in.
func (os *OperatingSystem) RequiresNudge() (bool, error) {
	if os.Platform != "darwin" {
		return false, nil
	}

	// strip Rapid Security Response suffix (e.g. version 13.3.7 (a)) if any
	version, err := VersionToSemverVersion(os.Version)
	if err != nil {
		return false, fmt.Errorf("parsing macos version \"%s\": %w", os.Version, err)
	}

	if version.LessThan(macOSNudgeLastVersion) {
		return true, nil
	}

	return false, nil
}
