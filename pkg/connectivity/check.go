// Package connectivity probes a Fleet server to verify that the HTTP endpoints
// required for the agent, MDM, and device-facing features are reachable from
// the network path of the caller. It is intended for support and preflight use
// from orbit (on-host) and fleetctl (off-host); it does not authenticate and
// does not require an enroll secret.
//
// See the public guide for the full set of endpoints and when each is needed:
// https://fleetdm.com/guides/what-api-endpoints-to-expose-to-the-public-internet
package connectivity

// Feature names a group of endpoints from the Fleet exposure guide. Users
// filter probes by Feature via the CLI.
type Feature string

const (
	FeatureOsquery    Feature = "osquery"
	FeatureDesktop    Feature = "fleet-desktop"
	FeatureFleetctl   Feature = "fleetctl"
	FeatureMDMMacOS   Feature = "mdm-macos"
	FeatureMDMWindows Feature = "mdm-windows"
	FeatureMDMIOS     Feature = "mdm-ios"
	FeatureMDMAndroid Feature = "mdm-android"
	FeatureSCEPProxy  Feature = "scep-proxy"
)

// Check is a single endpoint probe definition.
type Check struct {
	Feature     Feature
	Method      string
	Path        string
	Description string
}
