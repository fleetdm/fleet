package browserext

import "strings"

// broadHostPatterns are host-permission match patterns that grant read/modify
// access to effectively every site — the headline risk for an AI extension.
var broadHostPatterns = map[string]struct{}{
	"<all_urls>":  {},
	"*://*/*":     {},
	"http://*/*":  {},
	"https://*/*": {},
}

func hasBroadHostPerms(patterns []string) bool {
	for _, p := range patterns {
		if _, ok := broadHostPatterns[strings.ToLower(strings.TrimSpace(p))]; ok {
			return true
		}
	}
	return false
}

// Chromium Manifest::Location values we act on: kInvalidLocation — which is also
// the Go zero value, so it covers a Preferences entry with no "location" key —
// and the two we treat as trusted-origin.
const (
	chromiumLocUnknown   = 0
	chromiumLocInternal  = 1
	chromiumLocComponent = 5
)

// chromiumSideloaded reports whether a Chromium extension was installed outside
// the Web Store (unpacked/dev, external, or policy-forced). Conservative: when
// both signals are unknown it returns false to avoid false positives.
//
// A trusted location is checked first and wins over from_webstore. The browser
// installs its own first-party components (Edge Copilot Bridge, for instance),
// so by definition they are never web-store-installed and always report
// from_webstore:false — checking that signal first would make the trusted-origin
// exemption unreachable and flag every built-in component as sideloaded.
func chromiumSideloaded(fromWebstore, location int) bool {
	if location == chromiumLocInternal || location == chromiumLocComponent {
		return false
	}
	if fromWebstore == 0 { // explicitly not from the web store
		return true
	}
	// The trusted locations already returned above, so any location we can read
	// at this point is one an ordinary install does not land in; only an
	// unreadable location is left alone.
	return location != chromiumLocUnknown
}

// geckoSideloaded reports whether a Gecko addon is unsigned/temporary or was
// installed by another application (foreignInstall). Conservative on a truly
// unknown signedState.
func geckoSideloaded(signedState int, foreignInstall bool) bool {
	if foreignInstall {
		return true
	}
	if signedState != signedStateUnknown && signedState <= 0 {
		return true
	}
	return false
}

// computeRisk fills RiskFlags from the parsed host permissions and the
// per-engine Sideloaded determination. Stable token order.
func (e *Extension) computeRisk() {
	var flags []string
	if hasBroadHostPerms(e.HostPerms) {
		flags = append(flags, "broad_host_permissions")
	}
	if e.Sideloaded {
		flags = append(flags, "sideloaded_unverified")
	}
	e.RiskFlags = strings.Join(flags, ",")
}
