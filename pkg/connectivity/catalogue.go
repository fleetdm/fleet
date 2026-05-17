package connectivity

import (
	"fmt"
	"strings"
)

// AllFeatures returns the full set of features in rendering order.
func AllFeatures() []Feature {
	return []Feature{
		FeatureOsquery,
		FeatureDesktop,
		FeatureFleetctl,
		FeatureMDMMacOS,
		FeatureMDMWindows,
		FeatureMDMIOS,
		FeatureMDMAndroid,
		FeatureSCEPProxy,
	}
}

// ParseFeatures accepts a comma-separated list of feature names and returns
// the parsed features. Whitespace around entries is tolerated. An empty input
// returns nil, which callers interpret as "all features".
func ParseFeatures(s string) ([]Feature, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	valid := make(map[Feature]struct{}, len(AllFeatures()))
	for _, f := range AllFeatures() {
		valid[f] = struct{}{}
	}
	parts := strings.Split(s, ",")
	out := make([]Feature, 0, len(parts))
	seen := make(map[Feature]struct{}, len(parts))
	for _, p := range parts {
		f := Feature(strings.TrimSpace(p))
		if f == "" {
			continue
		}
		if _, ok := valid[f]; !ok {
			return nil, fmt.Errorf("unknown feature %q", f)
		}
		if _, dup := seen[f]; dup {
			continue
		}
		seen[f] = struct{}{}
		out = append(out, f)
	}
	return out, nil
}

// Catalogue returns the checks to run for the requested features. Passing no
// features returns the full catalogue. /enroll is shared between iOS and
// Android enrollment and is emitted once.
func Catalogue(features ...Feature) []Check {
	raw := rawCatalogue()
	if len(features) == 0 {
		return dedupeByPath(raw)
	}
	wanted := make(map[Feature]struct{}, len(features))
	for _, f := range features {
		wanted[f] = struct{}{}
	}
	out := make([]Check, 0, len(raw))
	for _, c := range raw {
		if _, ok := wanted[c.Feature]; ok {
			out = append(out, c)
		}
	}
	return dedupeByPath(out)
}

func rawCatalogue() []Check {
	// Fingerprint choices:
	//   - Ping endpoints (orbit/device) set X-Fleet-Capabilities even when
	//     unauthenticated, so the header is the tightest match.
	//   - Most JSON Fleet endpoints return the {"message":..., "errors":[...]}
	//     shape on 401/400.
	//   - MDM and SCEP endpoints speak protocol-specific formats (SOAP/XML,
	//     SCEP DER) with no consistent Fleet marker, so they can't be
	//     fingerprinted from a passive probe and are left at FingerprintNone.
	return []Check{
		{Feature: FeatureOsquery, Method: "POST", Path: "/api/osquery/enroll", Description: "osquery enroll", Fingerprint: FingerprintFleetJSONError},
		{Feature: FeatureOsquery, Method: "POST", Path: "/api/v1/osquery/enroll", Description: "osquery enroll (v1)", Fingerprint: FingerprintFleetJSONError},
		{Feature: FeatureOsquery, Method: "POST", Path: "/api/v1/osquery/config", Description: "osquery config", Fingerprint: FingerprintFleetJSONError},
		{Feature: FeatureOsquery, Method: "POST", Path: "/api/v1/osquery/distributed/read", Description: "osquery distributed read (live query)", Fingerprint: FingerprintFleetJSONError},
		{Feature: FeatureOsquery, Method: "POST", Path: "/api/v1/osquery/distributed/write", Description: "osquery distributed write (live query)", Fingerprint: FingerprintFleetJSONError},
		{Feature: FeatureOsquery, Method: "POST", Path: "/api/v1/osquery/log", Description: "osquery logger", Fingerprint: FingerprintFleetJSONError},
		{Feature: FeatureOsquery, Method: "POST", Path: "/api/v1/osquery/carve/begin", Description: "osquery carve begin", Fingerprint: FingerprintFleetJSONError},
		{Feature: FeatureOsquery, Method: "POST", Path: "/api/v1/osquery/carve/block", Description: "osquery carve block", Fingerprint: FingerprintFleetJSONError},

		// /api/fleet/device/ping has no reliable fingerprint: the endpoint
		// only sets X-Fleet-Capabilities when FLEET_ENABLE_POST_CLIENT_DEBUG_ERRORS
		// is on, and HEAD has no body to inspect. The device-authenticated
		// Desktop route below covers Fleet-identity verification for this
		// feature group.
		{Feature: FeatureDesktop, Method: "HEAD", Path: "/api/fleet/device/ping", Description: "Fleet device ping"},
		{Feature: FeatureDesktop, Method: "HEAD", Path: "/api/fleet/orbit/ping", Description: "orbit ping", Fingerprint: FingerprintCapabilitiesHeader},
		{Feature: FeatureDesktop, Method: "GET", Path: "/api/latest/fleet/device/connectivity-probe/desktop", Description: "device My device page", Fingerprint: FingerprintCapabilitiesHeader | FingerprintFleetJSONError},
		// POST /api/fleet/orbit/config returns the host's current config with
		// just the orbit node key. Side-effect-free and confirms Fleet trusts
		// this host's enrollment. JSON-error fingerprint covers the revoked-
		// key case: the capabilities-header ServerAfter hook doesn't fire on
		// auth-middleware rejection, but the standard {message,errors} 401
		// envelope does.
		{Feature: FeatureDesktop, Method: "POST", Path: "/api/fleet/orbit/config", Description: "authenticated orbit config", Fingerprint: FingerprintCapabilitiesHeader | FingerprintFleetJSONError, Auth: AuthOrbitNodeKey},

		{Feature: FeatureFleetctl, Method: "GET", Path: "/api/latest/fleet/version", Description: "fleetctl API", Fingerprint: FingerprintFleetJSONError},

		// SCEP and Apple MDM checkin speak binary/plist protocols handled by
		// third-party libraries (micromdm/scep, nanoMDM). Error bodies are
		// plain text from those libraries, not Fleet-emitted — no passive
		// fingerprint is reliable.
		{Feature: FeatureMDMMacOS, Method: "GET", Path: "/mdm/apple/scep", Description: "Apple SCEP"},
		{Feature: FeatureMDMMacOS, Method: "POST", Path: "/mdm/apple/mdm", Description: "Apple MDM checkin"},
		// /api/mdm/apple/enroll requires a token query param; without it the
		// endpoint always returns a Fleet JSON error regardless of Apple MDM
		// configuration state.
		{Feature: FeatureMDMMacOS, Method: "GET", Path: "/api/mdm/apple/enroll", Description: "Apple MDM enrollment profile", Fingerprint: FingerprintFleetJSONError},
		{Feature: FeatureMDMMacOS, Method: "GET", Path: "/mdm/sso", Description: "MDM SSO (initiate)", Fingerprint: FingerprintFleetHTMLTitle | FingerprintFleetJSONError},
		{Feature: FeatureMDMMacOS, Method: "POST", Path: "/mdm/sso/callback", Description: "MDM SSO (callback)"},
		{Feature: FeatureMDMMacOS, Method: "GET", Path: "/api/latest/fleet/mdm/setup/eula/metadata", Description: "Apple EULA metadata", Fingerprint: FingerprintFleetJSONError},
		{Feature: FeatureMDMMacOS, Method: "GET", Path: "/api/latest/fleet/mdm/bootstrap/summary", Description: "Apple bootstrap package summary", Fingerprint: FingerprintFleetJSONError},

		// Windows MDM endpoints are Fleet-implemented handlers. When MDM is
		// off they return Fleet JSON. When on, a real SOAP request would
		// return XML — but a GET/POST probe with no SOAP body still falls
		// through Fleet's own error path, so the JSON fingerprint holds.
		{Feature: FeatureMDMWindows, Method: "POST", Path: "/api/mdm/microsoft/discovery", Description: "MDE discovery", Fingerprint: FingerprintFleetJSONError},
		{Feature: FeatureMDMWindows, Method: "POST", Path: "/api/mdm/microsoft/policy", Description: "MS-XCEP policy", Fingerprint: FingerprintFleetJSONError},
		{Feature: FeatureMDMWindows, Method: "POST", Path: "/api/mdm/microsoft/enroll", Description: "MS-WSTEP enroll", Fingerprint: FingerprintFleetJSONError},
		{Feature: FeatureMDMWindows, Method: "POST", Path: "/api/mdm/microsoft/management", Description: "Windows MDM management", Fingerprint: FingerprintFleetJSONError},
		{Feature: FeatureMDMWindows, Method: "GET", Path: "/api/mdm/microsoft/tos", Description: "Windows Terms of Service", Fingerprint: FingerprintFleetJSONError | FingerprintFleetHTMLTitle},
		{Feature: FeatureMDMWindows, Method: "GET", Path: "/api/mdm/microsoft/auth", Description: "Windows MDM auth", Fingerprint: FingerprintFleetJSONError},

		{Feature: FeatureMDMIOS, Method: "GET", Path: "/enroll", Description: "enrollment page (iOS + Android)", Fingerprint: FingerprintFleetHTMLTitle},
		{Feature: FeatureMDMIOS, Method: "GET", Path: "/api/latest/fleet/enrollment_profiles/ota", Description: "iOS OTA enrollment profile", Fingerprint: FingerprintFleetJSONError},
		{Feature: FeatureMDMIOS, Method: "GET", Path: "/api/latest/fleet/software/titles/0/in_house_app", Description: "iOS in-house app", Fingerprint: FingerprintFleetJSONError},

		{Feature: FeatureMDMAndroid, Method: "GET", Path: "/enroll", Description: "enrollment page (iOS + Android)", Fingerprint: FingerprintFleetHTMLTitle},
		{Feature: FeatureMDMAndroid, Method: "GET", Path: "/api/latest/fleet/android_enterprise/enrollment_token", Description: "Android enrollment token", Fingerprint: FingerprintFleetJSONError},
		// pubsub endpoint is pinned to v1 in server/mdm/android/service/handler.go, not _version_ templated.
		{Feature: FeatureMDMAndroid, Method: "POST", Path: "/api/v1/fleet/android_enterprise/pubsub", Description: "Android PubSub webhook", Fingerprint: FingerprintFleetJSONError},
		{Feature: FeatureMDMAndroid, Method: "GET", Path: "/api/fleetd/certificates/0", Description: "Android fleetd certificates", Fingerprint: FingerprintCapabilitiesHeader | FingerprintFleetJSONError},

		{Feature: FeatureSCEPProxy, Method: "GET", Path: "/mdm/scep/proxy/probe", Description: "SCEP proxy"},
	}
}

// dedupeByPath removes later checks that share a method+path with an earlier
// check. HEAD and GET on the same path are intentionally kept as separate
// probes (different expected responses). Earlier entries win so shared paths
// like /enroll are attributed to iOS in rendering.
func dedupeByPath(checks []Check) []Check {
	seen := make(map[string]struct{}, len(checks))
	out := make([]Check, 0, len(checks))
	for _, c := range checks {
		key := c.Method + " " + c.Path
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, c)
	}
	return out
}
