package winget

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// wingetInputsDir is the real inputs directory relative to this package.
const wingetInputsDir = "../../inputs/winget"

// TestInputInstallerScopeIsSet enforces that every winget input declares a valid
// installer_scope ("machine" or "user").
//
// Scope must be explicit because Windows apps can install per-user (HKCU,
// %LOCALAPPDATA%) or per-machine (HKLM, Program Files). An unset scope leaves the
// ingester guessing and, more importantly, prevents remediation scripts from
// knowing which scope's copy to remove/upgrade — the root cause of duplicate/stale
// copies (see https://github.com/fleetdm/fleet/issues/48248).
//
// The detection ("exists") query must stay scope-blind; scope correctness lives in
// the installer_scope field and the install/uninstall scripts, never by narrowing
// the query. See the FMA authoring guide.
func TestInputInstallerScopeIsSet(t *testing.T) {
	entries, err := os.ReadDir(wingetInputsDir)
	require.NoError(t, err)

	checked := 0
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}

		path := filepath.Join(wingetInputsDir, e.Name())
		b, err := os.ReadFile(path)
		require.NoErrorf(t, err, "reading %s", e.Name())

		var input inputApp
		require.NoErrorf(t, json.Unmarshal(b, &input), "unmarshaling %s", e.Name())

		assert.Containsf(t, []string{machineScope, userScope}, input.InstallerScope,
			"%s: installer_scope must be %q or %q, got %q", e.Name(), machineScope, userScope, input.InstallerScope)
		checked++
	}

	require.NotZero(t, checked, "expected to check at least one winget input file")
}
