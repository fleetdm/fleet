package main

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/vulnerabilities/govulndb"
	"github.com/stretchr/testify/require"
)

// artifactOf builds an artifact with modules sized to the given advisory counts, for the checks
// that only care about how big it is.
func artifactOf(counts map[string]int) *govulndb.Artifact {
	modules := make(map[string][]govulndb.Advisory, len(counts))
	for module, n := range counts {
		advisories := make([]govulndb.Advisory, n)
		for i := range advisories {
			advisories[i] = govulndb.Advisory{ID: "GO-2024-0000", CVEs: []string{"CVE-2024-0000"}}
		}
		modules[module] = advisories
	}
	return &govulndb.Artifact{SchemaVersion: govulndb.SchemaVersion, Modules: modules}
}

// modulesOf builds an artifact of n modules, each with one advisory, always including stdlib.
func modulesOf(n int) *govulndb.Artifact {
	counts := map[string]int{govulndb.StdlibModule: 1}
	for i := 1; i < n; i++ {
		counts["github.com/example/m"+string(rune('a'+i%26))+string(rune('a'+i/26))] = 1
	}
	return artifactOf(counts)
}

func requireSuspect(t *testing.T, err error, want ...string) {
	t.Helper()

	require.True(t, isSuspect(err), "err = %v, want a suspect database", err)
	for _, w := range want {
		require.ErrorContains(t, err, w)
	}
}

func TestValidateRejectsEmptyArtifact(t *testing.T) {
	err := validate(&govulndb.Artifact{Modules: map[string][]govulndb.Advisory{}}, nil, defaultMaxDropPercent)

	requireSuspect(t, err, "no modules")
}

func TestValidateRejectsArtifactWithoutStdlib(t *testing.T) {
	artifact := artifactOf(map[string]int{"github.com/example/tool": 3})

	err := validate(artifact, nil, defaultMaxDropPercent)

	requireSuspect(t, err, govulndb.StdlibModule)
}

// The very first run has nothing to compare against, which must not stop it publishing.
func TestValidateWithoutPreviousArtifact(t *testing.T) {
	require.NoError(t, validate(modulesOf(100), nil, defaultMaxDropPercent))
}

func TestValidateModuleCountDrop(t *testing.T) {
	previous := modulesOf(1000)

	for _, tc := range []struct {
		name    string
		modules int
		suspect bool
	}{
		{name: "grew", modules: 1010},
		{name: "unchanged", modules: 1000},
		{name: "dropped within the threshold", modules: 960},
		{name: "dropped exactly the threshold", modules: 950},
		{name: "dropped past the threshold", modules: 949, suspect: true},
		{name: "collapsed", modules: 2, suspect: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := validate(modulesOf(tc.modules), previous, defaultMaxDropPercent)

			if !tc.suspect {
				require.NoError(t, err)
				return
			}
			// The alert has to say which check tripped and by how much, not just that a run
			// was skipped.
			requireSuspect(t, err, "module count", "dropped", "1000", "5.0%")
		})
	}
}

// A database can keep every module and still lose most of its advisories.
func TestValidateAdvisoryCountDrop(t *testing.T) {
	previous := artifactOf(map[string]int{govulndb.StdlibModule: 200, "github.com/example/tool": 800})
	artifact := artifactOf(map[string]int{govulndb.StdlibModule: 200, "github.com/example/tool": 500})

	err := validate(artifact, previous, defaultMaxDropPercent)

	requireSuspect(t, err, "advisory count", "1000 to 700")
}

// The threshold is a flag so it can be widened without a code change.
func TestValidateHonorsThreshold(t *testing.T) {
	previous, artifact := modulesOf(1000), modulesOf(800)

	require.NoError(t, validate(artifact, previous, 25))
	requireSuspect(t, validate(artifact, previous, 5))
}
