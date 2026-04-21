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
	return []Check{
		{Feature: FeatureOsquery, Method: "POST", Path: "/api/osquery/enroll", Description: "osquery enroll"},
		{Feature: FeatureOsquery, Method: "POST", Path: "/api/v1/osquery/enroll", Description: "osquery enroll (v1)"},

		{Feature: FeatureDesktop, Method: "HEAD", Path: "/api/fleet/device/ping", Description: "Fleet device ping"},
		{Feature: FeatureDesktop, Method: "HEAD", Path: "/api/fleet/orbit/ping", Description: "orbit ping"},
		{Feature: FeatureDesktop, Method: "GET", Path: "/api/latest/fleet/device/connectivity-probe/desktop", Description: "device My device page"},

		{Feature: FeatureFleetctl, Method: "GET", Path: "/api/latest/fleet/version", Description: "fleetctl API"},

		{Feature: FeatureMDMMacOS, Method: "GET", Path: "/mdm/apple/scep", Description: "Apple SCEP"},
		{Feature: FeatureMDMMacOS, Method: "POST", Path: "/mdm/apple/mdm", Description: "Apple MDM checkin"},
		{Feature: FeatureMDMMacOS, Method: "GET", Path: "/api/mdm/apple/enroll", Description: "Apple MDM enrollment profile"},
		{Feature: FeatureMDMMacOS, Method: "GET", Path: "/mdm/sso", Description: "MDM SSO (initiate)"},
		{Feature: FeatureMDMMacOS, Method: "POST", Path: "/mdm/sso/callback", Description: "MDM SSO (callback)"},
		{Feature: FeatureMDMMacOS, Method: "GET", Path: "/api/latest/fleet/mdm/setup/eula/metadata", Description: "Apple EULA metadata"},
		{Feature: FeatureMDMMacOS, Method: "GET", Path: "/api/latest/fleet/mdm/bootstrap/summary", Description: "Apple bootstrap package summary"},

		{Feature: FeatureMDMWindows, Method: "GET", Path: "/api/mdm/microsoft/discovery", Description: "MDE discovery"},
		{Feature: FeatureMDMWindows, Method: "POST", Path: "/api/mdm/microsoft/policy", Description: "MS-XCEP policy"},
		{Feature: FeatureMDMWindows, Method: "POST", Path: "/api/mdm/microsoft/enroll", Description: "MS-WSTEP enroll"},
		{Feature: FeatureMDMWindows, Method: "POST", Path: "/api/mdm/microsoft/management", Description: "Windows MDM management"},
		{Feature: FeatureMDMWindows, Method: "GET", Path: "/api/mdm/microsoft/tos", Description: "Windows Terms of Service"},
		{Feature: FeatureMDMWindows, Method: "GET", Path: "/api/mdm/microsoft/auth", Description: "Windows MDM auth"},

		{Feature: FeatureMDMIOS, Method: "GET", Path: "/enroll", Description: "enrollment page (iOS + Android)"},
		{Feature: FeatureMDMIOS, Method: "GET", Path: "/api/latest/fleet/enrollment_profiles/ota", Description: "iOS OTA enrollment profile"},
		{Feature: FeatureMDMIOS, Method: "GET", Path: "/api/latest/fleet/software/titles/0/in_house_app", Description: "iOS in-house app"},

		{Feature: FeatureMDMAndroid, Method: "GET", Path: "/enroll", Description: "enrollment page (iOS + Android)"},
		{Feature: FeatureMDMAndroid, Method: "GET", Path: "/api/latest/fleet/android_enterprise/enrollment_token", Description: "Android enrollment token"},
		// pubsub endpoint is pinned to v1 in server/mdm/android/service/handler.go, not _version_ templated.
		{Feature: FeatureMDMAndroid, Method: "POST", Path: "/api/v1/fleet/android_enterprise/pubsub", Description: "Android PubSub webhook"},
		{Feature: FeatureMDMAndroid, Method: "GET", Path: "/api/fleetd/certificates/0", Description: "Android fleetd certificates"},

		{Feature: FeatureSCEPProxy, Method: "GET", Path: "/mdm/scep/proxy/probe", Description: "SCEP proxy"},
	}
}

// dedupeByPath removes later checks that share a path with an earlier check.
// Earlier entries win so the enrollment page is attributed to iOS in rendering.
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
