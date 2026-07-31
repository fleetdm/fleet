package update

import (
	"regexp"

	"github.com/fleetdm/fleet/v4/server/mdm/microsoft/syncml"
)

// Exported so that it can be used in tools/ (so that it can be built for
// Windows and tested on a Windows machine). Otherwise not meant to be used
// from outside this package.
type WindowsMDMEnrollmentArgs struct {
	DiscoveryURL string
	HostUUID     string
	OrbitNodeKey string
}

// windowsEnrollmentStateUnknown is the EnrollmentState value that means "unknown / not enrolled".
const windowsEnrollmentStateUnknown = 0

// windowsEnrollmentGUIDRe matches a standard enrollment GUID (8-4-4-4-12 hex).
var windowsEnrollmentGUIDRe = regexp.MustCompile(`^[0-9A-Fa-f]{8}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{12}$`)

// isActiveFleetEnrollment reports whether an HKLM\SOFTWARE\Microsoft\Enrollments\<subkeyName> entry is Fleet's active Windows MDM
// enrollment. It matches when the ProviderID is Fleet's, the EnrollmentState is a non-zero (enrolled) value, and the subkey name is a
// well-formed enrollment GUID.
func isActiveFleetEnrollment(providerID string, state uint64, subkeyName string) bool {
	return providerID == syncml.DocProvisioningAppProviderID &&
		state != windowsEnrollmentStateUnknown &&
		windowsEnrollmentGUIDRe.MatchString(subkeyName)
}
