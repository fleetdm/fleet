package fleet

import "strings"

// OperatingSystem is an operating system uniquely identified according to its name and version.
type OperatingSystem struct {
	ID uint `json:"id" db:"id"`
	// Name is the name of the operating system, e.g., "Debian/GNU Linus", "Ubuntu", or "Microsoft Windows 11 Enterprise"
	Name string `json:"name" db:"name"`
	// Version is the version of the operating system, e.g., "10.0.0", "22.04 LTS", "21H2"
	Version string `json:"version" db:"version"`
	// Arch is the architecture of the operating system, e.g., "x86_64" or "64-bit"
	Arch string `json:"arch,omitempty" db:"arch"`
	// KernelVersion is the kernel version of the operating system, e.g., "5.10.76-linuxkit" or "10.0.22000.795"
	KernelVersion string `json:"kernel_version,omitempty" db:"kernel_version"`
	// Platform is the platform of the operating system, e.g., "darwin" or "rhel"
	Platform string `json:"platform" db:"platform"`
}

// IsWindows returns whether the OperatingSystem record references a Windows OS
func (os OperatingSystem) IsWindows() bool {
	return strings.ToLower(os.Platform) == "windows"
}
