package installer

import "github.com/fleetdm/fleet/v4/server/fleet"

// Client defines the methods required for the API requests to the server. The
// fleet.OrbitClient type satisfies this interface.
type Client interface {
	GetHostScript(execID string) (*fleet.HostScriptResult, error)
	GetInstaller(installerID, downloadDir string) (string, error)
	SaveHostScriptResult(result *fleet.HostScriptResultPayload) error
}
