package maintained_apps

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
)

// knownSharedDarwinIdentifiers allowlists macOS bundle identifiers that are
// intentionally shared by more than one differently-named Fleet-maintained app.
//
// Sharing a bundle identifier is ambiguous on macOS: osquery reports the same
// identifier for every app, so there is no way to tell them apart from inventory
// alone. The server handles this deliberately — see ReconcileMaintainedAppSoftwareNames
// (renames by identifier only when it maps to a single FMA name, otherwise uses the
// installer link) and fleetMaintainedAppsTeamJoin (marks an app "added" via its
// installer link rather than the shared identifier). This test fails on any NEW
// shared identifier so the collision is consciously reviewed against those paths
// instead of silently reintroducing https://github.com/fleetdm/fleet/issues/42445.
//
// If a new collision is intentional, confirm the paths above handle it and then add
// the identifier here with a comment naming the apps.
var knownSharedDarwinIdentifiers = map[string]string{
	"org.mozilla.firefox": "Mozilla Firefox and Mozilla Firefox ESR",
}

// TestNoUnexpectedSharedDarwinIdentifiers guards the maintained-apps manifest
// against new macOS FMAs that share a bundle identifier with a differently-named
// app without being accounted for in knownSharedDarwinIdentifiers.
func TestNoUnexpectedSharedDarwinIdentifiers(t *testing.T) {
	_, filename, _, _ := runtime.Caller(0)
	base := filepath.Dir(filepath.Dir(filepath.Dir(filepath.Dir(filename))))
	b, err := os.ReadFile(filepath.Join(base, "ee/maintained-apps/outputs/apps.json"))
	require.NoError(t, err)

	var appsList AppsList
	require.NoError(t, json.Unmarshal(b, &appsList))

	namesByIdentifier := make(map[string]map[string]struct{})
	for _, app := range appsList.Apps {
		if app.Platform != "darwin" || app.UniqueIdentifier == "" {
			continue
		}
		if namesByIdentifier[app.UniqueIdentifier] == nil {
			namesByIdentifier[app.UniqueIdentifier] = make(map[string]struct{})
		}
		namesByIdentifier[app.UniqueIdentifier][app.Name] = struct{}{}
	}

	for identifier, nameSet := range namesByIdentifier {
		if len(nameSet) <= 1 {
			continue
		}
		names := make([]string, 0, len(nameSet))
		for name := range nameSet {
			names = append(names, name)
		}
		sort.Strings(names)

		_, allowed := knownSharedDarwinIdentifiers[identifier]
		require.Truef(t, allowed,
			"macOS bundle identifier %q is shared by multiple differently-named Fleet-maintained apps (%v) "+
				"but is not in knownSharedDarwinIdentifiers.\n"+
				"Shared identifiers are ambiguous and must be handled by ReconcileMaintainedAppSoftwareNames and "+
				"fleetMaintainedAppsTeamJoin. If this is intentional, confirm those paths handle it and add %q to "+
				"knownSharedDarwinIdentifiers with a comment.",
			identifier, names, identifier)
	}
}
