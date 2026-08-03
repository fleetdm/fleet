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

// knownSharedDarwinIdentifiers allowlists macOS bundle identifiers intentionally
// shared by more than one differently-named FMA. Such collisions are ambiguous and
// must be handled by ReconcileMaintainedAppSoftwareNames and fleetMaintainedAppsTeamJoin
// (see https://github.com/fleetdm/fleet/issues/42445). The test below fails on any
// new one so it's reviewed against those paths before being added here.
var knownSharedDarwinIdentifiers = map[string]string{
	"org.mozilla.firefox": "Mozilla Firefox and Mozilla Firefox ESR",
}

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
