package windows_mdm

import "github.com/fleetdm/fleet/v4/server/mdm/internal/commonmdm"

const (
	// DiscoveryPath is Fleet's HTTP path for the Windows MDM Discovery endpoint.
	DiscoveryPath = "/EnrollmentServer/Discovery.svc"
)

func ResolveWindowsMDMDiscovery(serverURL string) (string, error) {
	return commonmdm.ResolveURL(serverURL, DiscoveryPath, false)
}
