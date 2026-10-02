package main

import (
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/vulnerabilities/govulndb"
	"github.com/stretchr/testify/require"
)

var generated = time.Date(2026, 9, 9, 0, 0, 0, 0, time.UTC)

// semverRange is one upstream SEMVER range, spelled the way vuln.go.dev spells it: a flat list
// of single-key events.
func semverRange(events ...osvEvent) osvRange {
	return osvRange{Type: semverRangeType, Events: events}
}

func introduced(v string) osvEvent { return osvEvent{Introduced: v} }
func fixed(v string) osvEvent      { return osvEvent{Fixed: v} }

func affects(module string, ranges ...osvRange) osvAffected {
	return osvAffected{
		Package: osvPackage{Name: module, Ecosystem: goEcosystem},
		Ranges:  ranges,
	}
}

func mustTransform(t *testing.T, reports ...osvReport) *govulndb.Artifact {
	t.Helper()
	artifact, err := transform(reports, generated)
	require.NoError(t, err)
	return artifact
}

func TestTransformArtifactMetadata(t *testing.T) {
	artifact := mustTransform(t, osvReport{
		ID:       "GO-2024-2963",
		Aliases:  []string{"CVE-2024-24791"},
		Affected: []osvAffected{affects(govulndb.StdlibModule, semverRange(introduced("0"), fixed("1.21.12")))},
	})

	require.Equal(t, govulndb.SchemaVersion, artifact.SchemaVersion)
	require.True(t, artifact.Generated.Equal(generated), "generated = %s", artifact.Generated)
}

// The stdlib shape from the artifact contract: two half-open ranges out of four events.
func TestTransformPairsEventsIntoRanges(t *testing.T) {
	artifact := mustTransform(t, osvReport{
		ID:      "GO-2024-2963",
		Aliases: []string{"CVE-2024-24791"},
		Affected: []osvAffected{
			affects(govulndb.StdlibModule, semverRange(
				introduced("0"), fixed("1.21.12"),
				introduced("1.22.0-0"), fixed("1.22.5"),
			)),
		},
	})

	want := []govulndb.Advisory{{
		ID:   "GO-2024-2963",
		CVEs: []string{"CVE-2024-24791"},
		Ranges: []govulndb.VersionRange{
			{Introduced: "0", Fixed: "1.21.12"},
			{Introduced: "1.22.0-0", Fixed: "1.22.5"},
		},
	}}
	require.Equal(t, want, artifact.Modules[govulndb.StdlibModule])
}

// A report with no fix yet ends on an open range, which Fleet reads as "every version from here
// on". The bound must not be invented.
func TestTransformKeepsUnfixedRangeOpen(t *testing.T) {
	artifact := mustTransform(t, osvReport{
		ID:      "GO-2024-1234",
		Aliases: []string{"CVE-2024-1234"},
		Affected: []osvAffected{
			affects("github.com/example/tool", semverRange(
				introduced("0"), fixed("1.2.0"),
				introduced("2.0.0"),
			)),
		},
	})

	advisories := artifact.Modules["github.com/example/tool"]
	require.Len(t, advisories, 1)
	want := []govulndb.VersionRange{{Introduced: "0", Fixed: "1.2.0"}, {Introduced: "2.0.0"}}
	require.Equal(t, want, advisories[0].Ranges)
}

// Toolchain advisories affect the `go` command on the build machine, not the binary it produced.
func TestTransformDropsToolchain(t *testing.T) {
	artifact := mustTransform(t,
		osvReport{
			ID:      "GO-2024-3000",
			Aliases: []string{"CVE-2024-3000"},
			Affected: []osvAffected{
				affects(toolchainModule, semverRange(introduced("0"), fixed("1.23.0"))),
				affects(govulndb.StdlibModule, semverRange(introduced("0"), fixed("1.23.0"))),
			},
		},
		osvReport{
			ID:       "GO-2024-3001",
			Aliases:  []string{"CVE-2024-3001"},
			Affected: []osvAffected{affects(toolchainModule, semverRange(introduced("0"), fixed("1.23.1")))},
		},
	)

	require.NotContains(t, artifact.Modules, toolchainModule)
	require.Len(t, artifact.Modules, 1, "want stdlib only")
	stdlib := artifact.Modules[govulndb.StdlibModule]
	require.Len(t, stdlib, 1)
	require.Equal(t, "GO-2024-3000", stdlib[0].ID)
}

// software_cve and every CVE metadata join are keyed on a CVE ID, so a report that carries no
// CVE alias cannot be represented in Fleet at all.
func TestTransformDropsReportsWithoutCVEAlias(t *testing.T) {
	artifact := mustTransform(t,
		osvReport{
			ID:       "GO-2024-4000",
			Aliases:  []string{"GHSA-xxxx-yyyy-zzzz"},
			Affected: []osvAffected{affects("github.com/example/ghsa-only", semverRange(introduced("0")))},
		},
		osvReport{
			ID:       "GO-2024-4001",
			Affected: []osvAffected{affects("github.com/example/no-aliases", semverRange(introduced("0")))},
		},
		osvReport{
			ID:       "GO-2024-4002",
			Aliases:  []string{"GHSA-aaaa-bbbb-cccc", "CVE-2024-4002"},
			Affected: []osvAffected{affects("github.com/example/both", semverRange(introduced("0")))},
		},
	)

	require.Len(t, artifact.Modules, 1, "want only github.com/example/both")
	advisories := artifact.Modules["github.com/example/both"]
	require.Len(t, advisories, 1)
	// The GHSA alias is dropped alongside; only CVE IDs are mirrored.
	require.Equal(t, []string{"CVE-2024-4002"}, advisories[0].CVEs)
}

// Withdrawn reports keep their ranges, and those ranges are usually "every version, no fix".
func TestTransformDropsWithdrawnReports(t *testing.T) {
	artifact := mustTransform(t, osvReport{
		ID:        "GO-2024-2442",
		Withdrawn: "2024-01-23T12:50:23Z",
		Aliases:   []string{"CVE-2024-2442"},
		Affected:  []osvAffected{affects("github.com/example/retracted", semverRange(introduced("0")))},
	})

	require.Empty(t, artifact.Modules)
}

// One report can affect several modules; each key gets only its own entry's ranges.
func TestTransformReportAffectingMultipleModules(t *testing.T) {
	artifact := mustTransform(t, osvReport{
		ID:      "GO-2024-5000",
		Aliases: []string{"CVE-2024-5000"},
		Affected: []osvAffected{
			affects("github.com/example/server", semverRange(introduced("0"), fixed("1.4.0"))),
			affects("github.com/example/client/v2", semverRange(introduced("2.0.0"), fixed("2.1.3"))),
		},
	})

	want := map[string][]govulndb.Advisory{
		"github.com/example/server": {{
			ID:     "GO-2024-5000",
			CVEs:   []string{"CVE-2024-5000"},
			Ranges: []govulndb.VersionRange{{Introduced: "0", Fixed: "1.4.0"}},
		}},
		// The /v2 suffix is part of the module path Fleet matches on, so it is kept verbatim.
		"github.com/example/client/v2": {{
			ID:     "GO-2024-5000",
			CVEs:   []string{"CVE-2024-5000"},
			Ranges: []govulndb.VersionRange{{Introduced: "2.0.0", Fixed: "2.1.3"}},
		}},
	}
	require.Equal(t, want, artifact.Modules)
}

// A report can carry two affected[] entries for the same module. They are one advisory with both
// sets of ranges, not two entries sharing an ID.
func TestTransformMergesRepeatedModuleEntries(t *testing.T) {
	artifact := mustTransform(t, osvReport{
		ID:      "GO-2023-2185",
		Aliases: []string{"CVE-2023-45283"},
		Affected: []osvAffected{
			affects(govulndb.StdlibModule, semverRange(
				introduced("0"), fixed("1.20.11"),
				introduced("1.21.0-0"), fixed("1.21.4"),
			)),
			affects(govulndb.StdlibModule, semverRange(
				introduced("1.20.11"), fixed("1.20.12"),
				introduced("1.21.4"), fixed("1.21.5"),
			)),
		},
	})

	advisories := artifact.Modules[govulndb.StdlibModule]
	require.Len(t, advisories, 1)

	want := []govulndb.VersionRange{
		{Introduced: "0", Fixed: "1.20.11"},
		{Introduced: "1.21.0-0", Fixed: "1.21.4"},
		{Introduced: "1.20.11", Fixed: "1.20.12"},
		{Introduced: "1.21.4", Fixed: "1.21.5"},
	}
	require.Equal(t, want, advisories[0].Ranges)
}

// Advisories are sorted so the same database always produces the same bytes.
func TestTransformSortsAdvisoriesByID(t *testing.T) {
	artifact := mustTransform(t,
		osvReport{
			ID:       "GO-2024-0002",
			Aliases:  []string{"CVE-2024-2"},
			Affected: []osvAffected{affects("github.com/example/tool", semverRange(introduced("0")))},
		},
		osvReport{
			ID:       "GO-2024-0001",
			Aliases:  []string{"CVE-2024-1"},
			Affected: []osvAffected{affects("github.com/example/tool", semverRange(introduced("0")))},
		},
	)

	advisories := artifact.Modules["github.com/example/tool"]
	require.Len(t, advisories, 2)
	require.Equal(t, "GO-2024-0001", advisories[0].ID)
	require.Equal(t, "GO-2024-0002", advisories[1].ID)
}

// Mis-pairing events shifts every bound in a report and puts wrong CVEs on customer hosts, and
// no version ordering but SEMVER is implemented. Both are a suspect database, not a report to
// patch around.
func TestTransformSuspectReports(t *testing.T) {
	for _, tc := range []struct {
		name   string
		ranges []osvRange
		want   string
	}{
		{
			name:   "range type other than SEMVER",
			ranges: []osvRange{{Type: "ECOSYSTEM", Events: []osvEvent{introduced("0"), fixed("1.0.0")}}},
			want:   `range type "ECOSYSTEM"`,
		},
		{
			name:   "two introduced in a row",
			ranges: []osvRange{semverRange(introduced("0"), introduced("2.0.0"))},
			want:   "is still open",
		},
		{
			name:   "fixed with no range open",
			ranges: []osvRange{semverRange(fixed("1.0.0"))},
			want:   "with no range open",
		},
		{
			name:   "fixed after a range was closed",
			ranges: []osvRange{semverRange(introduced("0"), fixed("1.0.0"), fixed("2.0.0"))},
			want:   "with no range open",
		},
		{
			name:   "event carrying both bounds",
			ranges: []osvRange{semverRange(osvEvent{Introduced: "0", Fixed: "1.0.0"})},
			want:   "not a single introduced or fixed bound",
		},
		{
			// An event upstream spells some other way, e.g. last_affected, decodes to neither
			// bound rather than being silently treated as one.
			name:   "event carrying neither bound",
			ranges: []osvRange{semverRange(osvEvent{})},
			want:   "not a single introduced or fixed bound",
		},
		{
			name:   "range with no events",
			ranges: []osvRange{semverRange()},
			want:   "carries no events",
		},
		{
			name:   "affected entry with no ranges",
			ranges: nil,
			want:   "no affected version ranges",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := transform([]osvReport{{
				ID:       "GO-2024-9999",
				Aliases:  []string{"CVE-2024-9999"},
				Affected: []osvAffected{affects("github.com/example/tool", tc.ranges...)},
			}}, generated)

			require.Error(t, err, "transform succeeded, want a suspect database")
			require.True(t, isSuspect(err), "error is not suspect: %v", err)
			require.ErrorContains(t, err, tc.want)
			require.ErrorContains(t, err, "GO-2024-9999", "want the error to name the report")
		})
	}
}

// The Go database only publishes the Go ecosystem. A document from any other is one this
// publisher was not written for, not one to re-shape on a guess.
func TestTransformSuspectEcosystem(t *testing.T) {
	_, err := transform([]osvReport{{
		ID:      "GO-2024-9999",
		Aliases: []string{"CVE-2024-9999"},
		Affected: []osvAffected{{
			Package: osvPackage{Name: "github.com/example/tool", Ecosystem: "npm"},
			Ranges:  []osvRange{semverRange(introduced("0"))},
		}},
	}}, generated)

	require.True(t, isSuspect(err), "error is not suspect: %v", err)
	require.ErrorContains(t, err, `ecosystem "npm"`)
	require.ErrorContains(t, err, "GO-2024-9999")
}
