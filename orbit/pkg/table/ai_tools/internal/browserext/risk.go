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

// Chromium Manifest::Location values we treat as trusted-origin.
const (
	chromiumLocInternal  = 1
	chromiumLocComponent = 5
)

// chromiumSideloaded reports whether a Chromium extension was installed outside
// the Web Store (unpacked/dev, external, or policy-forced). Conservative: when
// both signals are unknown it returns false to avoid false positives.
func chromiumSideloaded(fromWebstore, location int) bool {
	if fromWebstore == 0 { // explicitly not from the web store
		return true
	}
	if location != 0 && location != chromiumLocInternal && location != chromiumLocComponent {
		return true
	}
	return false
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
