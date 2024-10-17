package oval_parsed

import (
	"regexp"
	"strings"
)

// ReplaceFedoraOSVersion Replaces `version` with the equivalent RHEL version, this is so that we
// can use RHEL OVAL definitions when scanning/running tests - note that even though Fedora is the
// upstream of RHEL, there is no 1:1 mapping between Fedora's versions and RHEL (for example RHEL 6
// is based on both Fedora 12 and 13) - so this is an approximation.
// Examples:
// Red Hat Enterprise Linux 8.1.0
// 'Fedora Linux 36.0.0' => 'Red Hat Enterprise Linux 9.0.0'
// 'Fedora Linux 12.0.0' => 'Red Hat Enterprise Linux 6.0.0'
// 'Fedora Linux 13.0.0' => 'Red Hat Enterprise Linux 6.0.0'
func ReplaceFedoraOSVersion(version string) string {
	if strings.Contains(version, "Fedora") {
		rules := map[string]*regexp.Regexp{
			"Red Hat Enterprise Linux 6.0.0": regexp.MustCompile(`Fedora Linux (12|13|14|15|16|17|18)\.`),
			"Red Hat Enterprise Linux 7.0.0": regexp.MustCompile(`Fedora Linux (19|20|21|22|23|24|25|26|27)\.`),
			"Red Hat Enterprise Linux 8.0.0": regexp.MustCompile(`Fedora Linux (28|29|30|31|32|33)\.`),
			"Red Hat Enterprise Linux 9.0.0": regexp.MustCompile(`Fedora Linux (34|35|36|37|38|39|40)\.`),
		}
		for rep, pattern := range rules {
			if pattern.ReplaceAllString(version, rep) != version {
				return rep
			}
		}
		return "Red Hat Enterprise Linux 9.0.0"
	}
	return version
}
