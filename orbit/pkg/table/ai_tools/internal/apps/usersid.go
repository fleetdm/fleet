package apps

import "strings"

// Prefixes of the SIDs that belong to an actual person's account. Everything
// else with a loaded hive under HKEY_USERS is a machine or service account that
// never installs a desktop app.
//
// This lives in a platform-neutral file (rather than in apps_windows.go) so the
// filter can be tested on any platform; only the Windows build calls it.
var realUserSIDPrefixes = []string{
	"s-1-5-21-", // local and domain accounts: machine/domain SID + account RID
	"s-1-12-1-", // Entra ID (Azure AD) accounts
}

// isRealUserHive reports whether an HKEY_USERS subkey name is a real user's
// loaded hive. Registry key names are case-insensitive, so the comparison is too.
//
// The "<SID>_Classes" hives are skipped: they hold the same user's per-user class
// registrations, not a second account, and carry no uninstall entries.
func isRealUserHive(name string) bool {
	low := strings.ToLower(name)
	if strings.HasSuffix(low, "_classes") {
		return false
	}
	for _, p := range realUserSIDPrefixes {
		if strings.HasPrefix(low, p) && len(low) > len(p) {
			return true
		}
	}
	return false
}
