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

// getMajorRelease returns the major version of an 'os_version'.
// ex: 'Ubuntu 20.4.0' => '20'
func getMajorRelease(osVersion string) string {
	re := regexp.MustCompile(` (?P<major>\d+)\.?(\d+)?\.?(\*|\d+)?$`)
	m := re.FindStringSubmatch(osVersion)
	idx := re.SubexpIndex("major")

	if idx < len(m) {
		return m[idx]
	}
	return ""
}

// NewPlatform combines the host platform and os version into 'platform-os major version' string.
// Ex: ('ubuntu', 'Ubuntu 20.4.0') => 'ubuntu-20'.
func NewPlatform(hostPlatform, hostOsVersion string) Platform {
	nPlatform := strings.Trim(strings.ToLower(hostPlatform), " ")
	majorVer := getMajorRelease(hostOsVersion)
	return Platform(fmt.Sprintf("%s_%s", nPlatform, majorVer))
}

// ToFilename combines 'date' with the contents of 'platform' to produce a 'standard' filename.
func (op Platform) ToFilename(date time.Time, extension string) string {
	return fmt.Sprintf("%s_%s-%d_%02d_%02d.%s", OvalFilePrefix, op, date.Year(), date.Month(), date.Day(), extension)
}

// IsSupported returns whether the given platform is currently supported or not.
func (op Platform) IsSupported() bool {
	for _, p := range SupportedHostPlatforms {
		if strings.HasPrefix(string(op), p) {
			return true
		}
	}
	return false
}

func (op Platform) IsUbuntu() bool {
	return strings.HasPrefix(string(op), "ubuntu")
}
