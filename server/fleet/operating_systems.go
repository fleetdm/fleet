package fleet

import (
	"strings"
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
	// InstallationType is the Windows installation type, e.g. "Client", "Server", or "Server Core"
	InstallationType string `json:"installation_type" db:"installation_type"`
	// OSVersionID is a unique Name/Version combination for the operating system
	OSVersionID uint `json:"os_version_id" db:"os_version_id"`
}

// IsWindows returns whether the OperatingSystem record references a Windows OS
func (os OperatingSystem) IsWindows() bool {
	return strings.ToLower(os.Platform) == "windows"
}

const (
	// ArchLinuxOSName is the OS inventory name of Arch Linux and of the names in archGroupedOSNames.
	ArchLinuxOSName = "Arch Linux"
	// ArchLinuxPlatform is the platform the Arch Linux operating system always reports, whichever of its hosts'
	// platforms (arch, archarm) its inventory rows have.
	ArchLinuxPlatform = "arch"
)

// archGroupedOSNames are the OS names the OS inventory records as "Arch Linux", so they group with it in the OS list
// and on the dashboard.
var archGroupedOSNames = map[string]struct{}{
	"Arch Linux ARM": {},
}

// InventoryOSName returns the name the OS inventory records for an operating system that reports name.
func InventoryOSName(name string) string {
	if _, ok := archGroupedOSNames[name]; ok {
		return ArchLinuxOSName
	}
	return name
}

// HostAPIOS returns the os_name and os_version host API responses carry, from the host's OS inventory row and its stored
// display string (hosts.os_version). Hosts the inventory groups under "Arch Linux" keep their own reported name and
// version, which the display string holds as "<name> <version>". CachyOS and Omarchy hosts that still have the "Arch
// Linux" row an earlier Fleet recorded for them read as Arch Linux until they report again, like the OS list counts.
func HostAPIOS(inventoryName, inventoryVersion, display string) (name, version string) {
	if inventoryName == ArchLinuxOSName {
		for reported := range archGroupedOSNames {
			if v, ok := strings.CutPrefix(display, reported+" "); ok {
				return reported, v
			}
		}
	}
	return APIOSName(inventoryName), inventoryVersion
}

// APIOSName returns an OS inventory name the way the API shows it. Windows names drop the "Microsoft " prefix that the
// inventory keeps, because OS vulnerabilities are matched on the stored name.
func APIOSName(inventoryName string) string {
	if strings.HasPrefix(inventoryName, "Microsoft Windows") {
		return strings.TrimPrefix(inventoryName, "Microsoft ")
	}
	return inventoryName
}

// InventoryOSNames returns the OS inventory names that an os_name filter value matches. A Windows name matches with or
// without "Microsoft ", so filters written before Windows names dropped it keep working. The prefix check ignores case,
// like the name column does.
func InventoryOSNames(name string) []string {
	if strings.HasPrefix(strings.ToLower(name), "windows") {
		return []string{name, "Microsoft " + name}
	}
	return []string{name}
}

// IsArchGroupedOSName reports whether name is a reported OS name the OS inventory records as "Arch Linux".
func IsArchGroupedOSName(name string) bool {
	_, ok := archGroupedOSNames[name]
	return ok
}
