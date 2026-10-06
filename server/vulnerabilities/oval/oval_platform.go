package oval

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"

	oval_parsed "github.com/fleetdm/fleet/v4/server/vulnerabilities/oval/parsed"
)

type Platform string

// OvalFilePrefix is the file prefix used when saving an OVAL artifact.
const (
	OvalFilePrefix            = "fleet_oval"
	GovalDictionaryFilePrefix = "fleet_goval_dictionary"
)

// SupportedSoftwareSources are the software sources for which we are using OVAL or goval-dictionary for vulnerability detection.
var SupportedSoftwareSources = []string{"deb_packages", "rpm_packages"}

var SupportedGovalPlatforms = []string{
	"amzn_01",
	"amzn_02",
	"amzn_2022",
	"amzn_2023",
	"rhel_07",
	"rhel_08",
	"rhel_09",
}

// GovalKernelOnlyPlatforms are platforms where goval-dictionary is used only for kernel vulnerability scanning.
// These platforms use the regular OVAL scanning for non-kernel packages.
var GovalKernelOnlyPlatforms = []string{
	"rhel_07",
	"rhel_08",
	"rhel_09",
}

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
	if platform == "zorin" {
		// Zorin OS is Ubuntu-based; map to the underlying Ubuntu LTS OVAL feed.
		// Unknown future versions fall through to "zorin_<major>", which
		// IsSupported() rejects so vuln scanning is skipped rather than served
		// stale data from an aging LTS feed.
		switch major {
		case "16":
			return "ubuntu_2004"
		case "17":
			return "ubuntu_2204"
		case "18":
			return "ubuntu_2404"
		}
	}
	if platform == "ubuntu" {
		return fmt.Sprintf("%s_%s%s", platform, major, minor)
	}
	// RHEL based platforms only use the major version for their OVAL definitions
	return fmt.Sprintf("%s_%s", platform, major)
}

// platformTokenRe keeps separators, dots and percent-encoding out of the file paths and download
// URLs built from a Platform, since the inputs to NewPlatform are reported by hosts.
var platformTokenRe = regexp.MustCompile(`^[a-z0-9_]+$`)

// NewPlatform combines the host platform and os version into a string used to match OVAL
// definitions. It returns an empty Platform, which no feed supports, when the result is not a
// plain token.
// Examples:
// ('ubuntu', 'Ubuntu 20.4.0') => 'ubuntu_2004'.
// ('rhel', 'CentOS Linux 7.9.2009') => 'rhel_07'.
func NewPlatform(hostPlatform, hostOsVersion string) Platform {
	nPlatform := strings.Trim(strings.ToLower(hostPlatform), " ")
	hostOsVersion = oval_parsed.ReplaceFedoraOSVersion(hostOsVersion)
	major, minor := getMajorMinorVer(strings.Trim(hostOsVersion, " "))
	p := format(nPlatform, major, minor)
	if !platformTokenRe.MatchString(p) {
		return ""
	}
	return Platform(p)
}

// ToFilename combines 'date' with the contents of 'platform' to produce a 'standard' filename.
func (op Platform) ToFilename(date time.Time, extension string) string {
	return fmt.Sprintf("%s_%s-%d_%02d_%02d.%s", OvalFilePrefix, op, date.Year(), date.Month(), date.Day(), extension)
}

func (op Platform) ToGovalDictionaryFilename() string {
	return fmt.Sprintf("%s_%s.sqlite3", GovalDictionaryFilePrefix, op)
}

// ToGovalDatabaseFilename returns the filename of the sqlite3 files downloaded using
// the goval-dictionary fetch method in the vulnerabilities generate-cve.yml workflow
func (op Platform) ToGovalDatabaseFilename() string {
	return fmt.Sprintf("%s.sqlite3", op)
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
		"ubuntu_2410",
		"ubuntu_2504",
		"rhel_05",
		"rhel_06",
		"rhel_07",
		"rhel_08",
		"rhel_09",
	}
	return slices.Contains(supported, string(op))
}

// IsGovalDictionarySupported must match exactly: the platform is reported by hosts and is used to
// build goval-dictionary download URLs and file paths.
func (op Platform) IsGovalDictionarySupported() bool {
	return slices.Contains(SupportedGovalPlatforms, string(op))
}

// IsGovalDictionaryKernelOnly returns true if this platform uses goval-dictionary
// only for kernel vulnerability scanning (non-kernel packages use regular OVAL).
func (op Platform) IsGovalDictionaryKernelOnly() bool {
	return slices.Contains(GovalKernelOnlyPlatforms, string(op))
}

// IsUbuntu checks whether the current Platform targets Ubuntu.
func (op Platform) IsUbuntu() bool {
	return strings.HasPrefix(string(op), "ubuntu")
}

// IsRedHat checks whether the current Platform targets Redhat based systems.
func (op Platform) IsRedHat() bool {
	return strings.HasPrefix(string(op), "rhel") || strings.HasPrefix(string(op), "amzn")
}
