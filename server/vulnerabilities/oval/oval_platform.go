package oval

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	oval_parsed "github.com/fleetdm/fleet/v4/server/vulnerabilities/oval/parsed"
)

type Platform string

// OvalFilePrefix is the file prefix used when saving an OVAL artifact.
const OvalFilePrefix = "fleet_oval"
const GovalDictionaryFilePrefix = "fleet_goval_dictionary"

// SupportedSoftwareSources are the software sources for which we are using OVAL or goval-dictionary for vulnerability detection.
var SupportedSoftwareSources = []string{"deb_packages", "rpm_packages"}

// getMajorMinorVer returns the major and minor version of an 'os_version'.
// ex: 'Ubuntu 20.4.0' => '(20, 04)'
func getMajorMinorVer(osVersion string) (string, string) {
	re := regexp.MustCompile(` (?P<major>\d+)\.?(?P<minor>\d+)?`)
	m := re.FindStringSubmatch(osVersion)

	if len(m) < 2 {
		return "", ""
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
		return major, minor
	}
	return "", ""
}

func format(platform string, major string, minor string) string {
	if platform == "ubuntu" {
		return fmt.Sprintf("%s_%s%s", platform, major, minor)
	}
	// RHEL based platforms only use the major version for their OVAL definitions
	return fmt.Sprintf("%s_%s", platform, major)
}

// NewPlatform combines the host platform and os version into a string used to match OVAL
// definitions.
// Examples:
// ('ubuntu', 'Ubuntu 20.4.0') => 'ubuntu_2004'.
// ('rhel', 'CentOS Linux 7.9.2009') => 'rhel_07'.
func NewPlatform(hostPlatform, hostOsVersion string) Platform {
	nPlatform := strings.Trim(strings.ToLower(hostPlatform), " ")
	hostOsVersion = oval_parsed.ReplaceFedoraOSVersion(hostOsVersion)
	major, minor := getMajorMinorVer(strings.Trim(hostOsVersion, " "))
	return Platform(format(nPlatform, major, minor))
}

// ToFilename combines 'date' with the contents of 'platform' to produce a 'standard' filename.
func (op Platform) ToFilename(date time.Time, extension string) string {
	return fmt.Sprintf("%s_%s-%d_%02d_%02d.%s", OvalFilePrefix, op, date.Year(), date.Month(), date.Day(), extension)
}

func (op Platform) ToGovalDictionaryFilename() string {
	return fmt.Sprintf("%s_%s.sqlite3", GovalDictionaryFilePrefix, op)
}

// IsSupported returns whether the given platform is currently supported.
func (op Platform) IsSupported() bool {
	supported := []string{
		"ubuntu_1404",
		"ubuntu_1604",
		"ubuntu_1804",
		"ubuntu_1910",
		"ubuntu_2004",
		"ubuntu_2104",
		"ubuntu_2110",
		"ubuntu_2204",
		"ubuntu_2210",
		"ubuntu_2304",
		"ubuntu_2310",
		"ubuntu_2404",
		"rhel_05",
		"rhel_06",
		"rhel_07",
		"rhel_08",
		"rhel_09",
	}
	for _, p := range supported {
		if strings.HasPrefix(string(op), p) {
			return true
		}
	}
	return false
}

func (op Platform) IsGovalDictionarySupported() bool {
	supported := []string{
		"amzn_01",
		"amzn_02",
		"amzn_2022",
		"amzn_2023",
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

// IsRedHat checks whether the current Platform targets Redhat based systems.
func (op Platform) IsRedHat() bool {
	return strings.HasPrefix(string(op), "rhel") || strings.HasPrefix(string(op), "amzn")
}
