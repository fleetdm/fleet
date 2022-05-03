package oval

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Combines the host platform and os version into 'platform-major version' string,
// ex: ('ubuntu', 'Ubuntu 20.4.0') => 'ubuntu-20'
type Platform string

const OvalFilePrefix = "fleet_oval"

// Returns the major version of an 'os_version',
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

func NewPlatform(hostPlatform, hostOsVersion string) Platform {
	nPlatform := strings.Trim(strings.ToLower(hostPlatform), " ")
	majorVer := getMajorRelease(hostOsVersion)
	return Platform(fmt.Sprintf("%s-%s", nPlatform, majorVer))
}

func (op Platform) ToFilename(date time.Time, extension string) string {
	return fmt.Sprintf("%s_%s_%d-%d-%d.%s", OvalFilePrefix, op, date.Year(), date.Month(), date.Day(), extension)
}

func (op Platform) IsSupported() bool {
	return strings.HasPrefix(string(op), "ubuntu")
}
