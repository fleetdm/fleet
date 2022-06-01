package oval

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

type Platform string

// OvalFilePrefix is the file prefix used when saving an OVAL artifact.
const OvalFilePrefix = "fleet_oval"

var SupportedHostPlatforms = []string{"ubuntu"}

// getMajorMinorVer returns the major and minor version of an 'os_version'.
// ex: 'Ubuntu 20.4.0' => '2004'
func getMajorMinorVer(osVersion string) string {
	re := regexp.MustCompile(` (?P<major>\d+)\.?(?P<minor>\d+)?\.?(\*|\d+)?$`)
	m := re.FindStringSubmatch(osVersion)

	if len(m) < 2 {
		return ""
	}

	maIdx := re.SubexpIndex("major")
	miIdx := re.SubexpIndex("minor")

	if maIdx > 0 && miIdx > 0 {
		major := m[maIdx]
		if len(major) < 2 {
			major = fmt.Sprintf("0%s", major)
		}
		minor := m[miIdx]
		if len(minor) < 2 {
			minor = fmt.Sprintf("0%s", minor)
		}
		return fmt.Sprintf("%s%s", major, minor)
	}
	return ""
}

// NewPlatform combines the host platform and os version into 'platform-os major version' string.
// Ex: ('ubuntu', 'Ubuntu 20.4.0') => 'ubuntu-20'.
func NewPlatform(hostPlatform, hostOsVersion string) Platform {
	nPlatform := strings.Trim(strings.ToLower(hostPlatform), " ")
	majorVer := getMajorMinorVer(strings.Trim(hostOsVersion, " "))
	return Platform(fmt.Sprintf("%s_%s", nPlatform, majorVer))
}

// ToFilename combines 'date' with the contents of 'platform' to produce a 'standard' filename.
func (op Platform) ToFilename(date time.Time, extension string) string {
	return fmt.Sprintf("%s_%s-%d_%02d_%02d.%s", OvalFilePrefix, op, date.Year(), date.Month(), date.Day(), extension)
}

// IsSupported returns whether the given platform is currently supported or not.
func (op Platform) IsSupported() bool {
	supported := []string{
		"ubuntu_1404",
		"ubuntu_1604",
		"ubuntu_1804",
		"ubuntu_2004",
		"ubuntu_2104",
		"ubuntu_2110",
		"ubuntu_2204",
	}
	for _, p := range supported {
		if strings.HasPrefix(string(op), p) {
			return true
		}
	}
	return false
}

// IsUbuntu checks whether the current Platform targets Ubuntu.
func (op Platform) IsUbuntu() bool {
	return strings.HasPrefix(string(op), "ubuntu")
}
