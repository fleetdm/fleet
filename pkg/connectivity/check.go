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

// FingerprintMode is a bitmask of checks Probe applies to the response body
// and headers to confirm the responder is actually Fleet. When none of a
// check's requested fingerprints match, the result is StatusNotFleet.
type FingerprintMode uint8

const (
	// FingerprintNone skips fingerprinting. Use for MDM, SCEP, and other
	// protocol endpoints that don't emit a Fleet-identifiable response.
	FingerprintNone FingerprintMode = 0
	// FingerprintCapabilitiesHeader matches the X-Fleet-Capabilities response
	// header, which Fleet sets on orbit, device, and Android endpoints.
	FingerprintCapabilitiesHeader FingerprintMode = 1 << iota
	// FingerprintFleetJSONError matches Fleet's standard JSON error body
	// shape: {"message": "...", "errors": [...]}. Covers authenticated JSON
	// endpoints that return 4xx when probed unauthenticated.
	FingerprintFleetJSONError
	// FingerprintFleetHTMLTitle matches an HTML <title>Fleet</title> tag
	// (or <title>Fleet ...</title>) in the response body. Covers
	// enrollment and SSO pages that return the Fleet web bundle.
	FingerprintFleetHTMLTitle
)

// AuthMode indicates which credential, if any, Probe should present.
type AuthMode uint8

const (
	// AuthNone sends the request with no credentials.
	AuthNone AuthMode = iota
	// AuthOrbitNodeKey attaches the host's orbit node key as a JSON body
	// ({"orbit_node_key":"..."}). Falls back to AuthNone when the probe is
	// run without an orbit node key available.
	AuthOrbitNodeKey
)

// Check is a single endpoint probe definition.
type Check struct {
	Feature     Feature
	Method      string
	Path        string
	Description string
	// Fingerprint declares which Fleet-identifying signals the response must
	// carry. Zero means the check accepts any HTTP response as "reachable".
	Fingerprint FingerprintMode
	// Auth selects the credential used. When the credential is unavailable,
	// Probe downgrades to AuthNone and runs the unauthenticated path.
	Auth AuthMode
}
