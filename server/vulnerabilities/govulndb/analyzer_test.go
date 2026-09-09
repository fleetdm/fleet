package govulndb

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/stretchr/testify/require"
)

// testArtifact mirrors the shapes seen in the real database: a module advisory with a single
// open-ended range, a module advisory with a fix, an unfixed advisory, an advisory with no CVE
// alias, and the multi-range stdlib shape (a fix on the current minor plus a fix on the next).
func testArtifact() *Artifact {
	return &Artifact{
		SchemaVersion: "1",
		Modules: map[string][]Advisory{
			"github.com/air-verse/air": {
				{
					ID:     "GO-2024-1000",
					CVEs:   []string{"CVE-2024-1000"},
					Ranges: []VersionRange{{Introduced: "0", Fixed: "1.49.0"}},
				},
				{
					ID:     "GO-2024-1001",
					CVEs:   []string{"CVE-2024-1001"},
					Ranges: []VersionRange{{Introduced: "1.20.0", Fixed: "1.40.0"}},
				},
				{
					ID:     "GO-2026-1002",
					CVEs:   []string{"CVE-2026-1002"},
					Ranges: []VersionRange{{Introduced: "1.48.0"}},
				},
				{
					ID:     "GO-2026-1003",
					Ranges: []VersionRange{{Introduced: "0"}},
				},
			},
			stdlibModule: {
				{
					ID:   "GO-2024-2963",
					CVEs: []string{"CVE-2024-24791"},
					Ranges: []VersionRange{
						{Introduced: "0", Fixed: "1.21.12"},
						{Introduced: "1.22.0-0", Fixed: "1.22.5"},
					},
				},
			},
			"toolchain": {
				{
					ID:     "GO-2024-3000",
					CVEs:   []string{"CVE-2024-3000"},
					Ranges: []VersionRange{{Introduced: "0"}},
				},
			},
		},
	}
}

func cves(vulns []fleet.SoftwareVulnerability) []string {
	out := make([]string, 0, len(vulns))
	for _, v := range vulns {
		out = append(out, v.CVE)
	}
	return out
}

func TestMatchSoftware(t *testing.T) {
	artifact := testArtifact()

	for _, tc := range []struct {
		name     string
		software fleet.Software
		wantCVEs []string
	}{
		{
			name:     "module advisory below the fix",
			software: fleet.Software{ID: 1, Name: "air", Version: "v1.48.0", ExtensionID: "github.com/air-verse/air"},
			wantCVEs: []string{"CVE-2024-1000", "CVE-2026-1002"},
		},
		{
			name:     "module advisory at the fix",
			software: fleet.Software{ID: 1, Name: "air", Version: "v1.49.0", ExtensionID: "github.com/air-verse/air"},
			wantCVEs: []string{"CVE-2026-1002"},
		},
		{
			name:     "version below the introduced bound",
			software: fleet.Software{ID: 1, Name: "air", Version: "v1.10.0", ExtensionID: "github.com/air-verse/air"},
			wantCVEs: []string{"CVE-2024-1000"},
		},
		{
			name:     "version inside a closed range",
			software: fleet.Software{ID: 1, Name: "air", Version: "v1.30.0", ExtensionID: "github.com/air-verse/air"},
			wantCVEs: []string{"CVE-2024-1000", "CVE-2024-1001"},
		},
		{
			name:     "unknown module",
			software: fleet.Software{ID: 1, Name: "air", Version: "v1.0.0", ExtensionID: "github.com/other/air"},
			wantCVEs: nil,
		},
		{
			name:     "empty module path",
			software: fleet.Software{ID: 1, Name: "air", Version: "v1.48.0"},
			wantCVEs: nil,
		},
		{
			name:     "devel version",
			software: fleet.Software{ID: 1, Name: "air", Version: "(devel)", ExtensionID: "github.com/air-verse/air"},
			wantCVEs: nil,
		},
		{
			name:     "version without the leading v",
			software: fleet.Software{ID: 1, Name: "air", Version: "1.48.0", ExtensionID: "github.com/air-verse/air"},
			wantCVEs: []string{"CVE-2024-1000", "CVE-2026-1002"},
		},
		{
			name:     "unparseable version",
			software: fleet.Software{ID: 1, Name: "air", Version: "not-a-version", ExtensionID: "github.com/air-verse/air"},
			wantCVEs: nil,
		},
		{
			// `go install pkg@latest` on an untagged module, or pkg@<commit> on any module,
			// records a pseudo-version of v0.0.0, which sorts below "0"'s canonical v0.0.0.
			name:     "pseudo-version of an untagged module",
			software: fleet.Software{ID: 1, Name: "air", Version: "v0.0.0-20240101000000-abcdef123456", ExtensionID: "github.com/air-verse/air"},
			wantCVEs: []string{"CVE-2024-1000"},
		},
		{
			name:     "pseudo-version after a tagged release",
			software: fleet.Software{ID: 1, Name: "air", Version: "v1.48.1-0.20240101000000-abcdef123456", ExtensionID: "github.com/air-verse/air"},
			wantCVEs: []string{"CVE-2024-1000", "CVE-2026-1002"},
		},
		{
			name:     "stdlib advisory below the fix",
			software: fleet.Software{ID: 2, Name: "tool", Version: "v1.0.0", Release: "go1.21.11"},
			wantCVEs: []string{"CVE-2024-24791"},
		},
		{
			name:     "stdlib advisory at the fix",
			software: fleet.Software{ID: 2, Name: "tool", Version: "v1.0.0", Release: "go1.21.12"},
			wantCVEs: nil,
		},
		{
			name:     "stdlib advisory in the next minor's range",
			software: fleet.Software{ID: 2, Name: "tool", Version: "v1.0.0", Release: "go1.22.4"},
			wantCVEs: []string{"CVE-2024-24791"},
		},
		{
			name:     "stdlib advisory past the next minor's fix",
			software: fleet.Software{ID: 2, Name: "tool", Version: "v1.0.0", Release: "go1.22.5"},
			wantCVEs: nil,
		},
		{
			name:     "stdlib advisory on a minor with no patch component",
			software: fleet.Software{ID: 2, Name: "tool", Version: "v1.0.0", Release: "go1.21"},
			wantCVEs: []string{"CVE-2024-24791"},
		},
		{
			name:     "stdlib advisory matches a devel binary",
			software: fleet.Software{ID: 2, Name: "tool", Version: "(devel)", Release: "go1.21.11"},
			wantCVEs: []string{"CVE-2024-24791"},
		},
		{
			name:     "release candidate toolchain inside the next minor's range",
			software: fleet.Software{ID: 2, Name: "tool", Version: "v1.0.0", Release: "go1.22rc1"},
			wantCVEs: []string{"CVE-2024-24791"},
		},
		{
			name:     "beta toolchain inside the next minor's range",
			software: fleet.Software{ID: 2, Name: "tool", Version: "v1.0.0", Release: "go1.22beta1"},
			wantCVEs: []string{"CVE-2024-24791"},
		},
		{
			name:     "release candidate toolchain past every fix",
			software: fleet.Software{ID: 2, Name: "tool", Version: "v1.0.0", Release: "go1.23rc1"},
			wantCVEs: nil,
		},
		{
			// Binaries built with a GOEXPERIMENT report it after the version.
			name:     "toolchain with an experiment suffix",
			software: fleet.Software{ID: 2, Name: "tool", Version: "v1.0.0", Release: "go1.21.11 X:boringcrypto"},
			wantCVEs: []string{"CVE-2024-24791"},
		},
		{
			name:     "development toolchain is skipped",
			software: fleet.Software{ID: 2, Name: "tool", Version: "v1.0.0", Release: "devel go1.27-abcdef1234 Mon Jan 1 00:00:00 2026 +0000"},
			wantCVEs: nil,
		},
		{
			name:     "empty toolchain version",
			software: fleet.Software{ID: 2, Name: "tool", Version: "v1.0.0"},
			wantCVEs: nil,
		},
		{
			name: "module and stdlib advisories together",
			software: fleet.Software{
				ID: 3, Name: "air", Version: "v1.48.0",
				ExtensionID: "github.com/air-verse/air", Release: "go1.21.11",
			},
			wantCVEs: []string{"CVE-2024-1000", "CVE-2026-1002", "CVE-2024-24791"},
		},
		{
			name:     "toolchain advisories never match",
			software: fleet.Software{ID: 4, Name: "go", Version: "v1.21.11", ExtensionID: "toolchain", Release: "go1.21.11"},
			wantCVEs: []string{"CVE-2024-24791"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := matchSoftware(&tc.software, artifact)
			require.ElementsMatch(t, tc.wantCVEs, cves(got))
			for _, v := range got {
				require.Equal(t, tc.software.ID, v.SoftwareID)
			}
		})
	}
}

func TestMatchSoftwareResolvedInVersion(t *testing.T) {
	artifact := testArtifact()

	t.Run("module advisory carries the fixed version", func(t *testing.T) {
		got := matchSoftware(&fleet.Software{
			ID: 1, Name: "air", Version: "v1.48.0", ExtensionID: "github.com/air-verse/air",
		}, artifact)

		byCVE := make(map[string]*string, len(got))
		for _, v := range got {
			byCVE[v.CVE] = v.ResolvedInVersion
		}
		require.NotNil(t, byCVE["CVE-2024-1000"])
		require.Equal(t, "v1.49.0", *byCVE["CVE-2024-1000"])
		// GO-2026-1002 has no fixed version yet.
		require.Nil(t, byCVE["CVE-2026-1002"])
	})

	t.Run("stdlib advisory carries no fixed version", func(t *testing.T) {
		got := matchSoftware(&fleet.Software{
			ID: 2, Name: "tool", Version: "v1.0.0", Release: "go1.21.11",
		}, artifact)
		require.Len(t, got, 1)
		require.Nil(t, got[0].ResolvedInVersion)
	})
}

type fakeSoftwareIterator struct {
	index     int
	softwares []*fleet.Software
	closed    bool
}

func (f *fakeSoftwareIterator) Next() bool { return f.index < len(f.softwares) }

func (f *fakeSoftwareIterator) Value() (*fleet.Software, error) {
	s := f.softwares[f.index]
	f.index++
	return s, nil
}

func (f *fakeSoftwareIterator) Err() error   { return nil }
func (f *fakeSoftwareIterator) Close() error { f.closed = true; return nil }

// analyzeTestStore returns a mock datastore that serves the given Go binaries and records what
// the analyzer writes.
func analyzeTestStore(software []*fleet.Software) (*mock.Store, *[]fleet.SoftwareVulnerability, *[]string, **time.Time) {
	ds := new(mock.Store)
	inserted := new([]fleet.SoftwareVulnerability)
	includedSources := new([]string)
	deletedOlderThan := new(*time.Time)

	ds.AllSoftwareIteratorFunc = func(ctx context.Context, q fleet.SoftwareIterQueryOptions) (fleet.SoftwareIterator, error) {
		*includedSources = q.IncludedSources
		return &fakeSoftwareIterator{softwares: software}, nil
	}
	ds.InsertSoftwareVulnerabilitiesFunc = func(
		ctx context.Context, vulns []fleet.SoftwareVulnerability, source fleet.VulnerabilitySource,
	) ([]fleet.SoftwareVulnerability, error) {
		if source != fleet.GoVulnDBSource {
			return nil, fmt.Errorf("unexpected source %d", source)
		}
		*inserted = append(*inserted, vulns...)
		return vulns, nil
	}
	ds.DeleteOutOfDateVulnerabilitiesFunc = func(
		ctx context.Context, source fleet.VulnerabilitySource, olderThan time.Time,
	) error {
		if source != fleet.GoVulnDBSource {
			return fmt.Errorf("unexpected source %d", source)
		}
		*deletedOlderThan = &olderThan
		return nil
	}

	return ds, inserted, includedSources, deletedOlderThan
}

func TestAnalyze(t *testing.T) {
	logger := slog.New(slog.DiscardHandler)

	t.Run("no artifact on disk is not an error", func(t *testing.T) {
		ds, _, _, _ := analyzeTestStore(nil)

		vulns, err := Analyze(t.Context(), ds, t.TempDir(), true, time.Now(), logger)
		require.NoError(t, err)
		require.Empty(t, vulns)
		require.False(t, ds.AllSoftwareIteratorFuncInvoked)
	})

	t.Run("an artifact with an unknown schema version is refused", func(t *testing.T) {
		dir := t.TempDir()
		artifact := testArtifact()
		artifact.SchemaVersion = "2"
		writeArtifact(t, dir, "govulndb-2026-09-09.json.gz", artifact)

		ds, _, _, _ := analyzeTestStore(nil)

		_, err := Analyze(t.Context(), ds, dir, true, time.Now(), logger)
		require.ErrorContains(t, err, "schema version")
		require.False(t, ds.AllSoftwareIteratorFuncInvoked)
		require.False(t, ds.DeleteOutOfDateVulnerabilitiesFuncInvoked)
	})

	t.Run("an artifact with no modules is refused", func(t *testing.T) {
		dir := t.TempDir()
		writeArtifact(t, dir, "govulndb-2026-09-09.json.gz", &Artifact{SchemaVersion: "1"})

		ds, _, _, _ := analyzeTestStore(nil)

		_, err := Analyze(t.Context(), ds, dir, true, time.Now(), logger)
		require.ErrorContains(t, err, "no modules")
		require.False(t, ds.DeleteOutOfDateVulnerabilitiesFuncInvoked)
	})

	t.Run("matches Go binaries and removes stale rows", func(t *testing.T) {
		dir := t.TempDir()
		writeArtifact(t, dir, "govulndb-2026-09-09.json.gz", testArtifact())

		startTime := time.Now().Add(-time.Hour)
		ds, inserted, includedSources, deletedOlderThan := analyzeTestStore([]*fleet.Software{
			{ID: 1, Name: "air", Version: "v1.48.0", ExtensionID: "github.com/air-verse/air", Release: "go1.26.1"},
			{ID: 2, Name: "air", Version: "v1.49.0", ExtensionID: "github.com/air-verse/air", Release: "go1.21.11"},
			{ID: 3, Name: "devtool", Version: "(devel)", ExtensionID: "github.com/fleetdm/fleet/v4", Release: "go1.26.1"},
		})

		vulns, err := Analyze(t.Context(), ds, dir, true, startTime, logger)
		require.NoError(t, err)

		require.Equal(t, []string{softwareSource}, *includedSources)
		require.ElementsMatch(t, []fleet.SoftwareVulnerability{
			{SoftwareID: 1, CVE: "CVE-2024-1000", ResolvedInVersion: new("v1.49.0")},
			{SoftwareID: 1, CVE: "CVE-2026-1002"},
			{SoftwareID: 2, CVE: "CVE-2026-1002"},
			{SoftwareID: 2, CVE: "CVE-2024-24791"},
		}, *inserted)
		require.ElementsMatch(t, *inserted, vulns)

		require.NotNil(t, *deletedOlderThan)
		require.Equal(t, startTime, **deletedOlderThan)
	})

	t.Run("collectVulns off returns nothing but still writes", func(t *testing.T) {
		dir := t.TempDir()
		writeArtifact(t, dir, "govulndb-2026-09-09.json.gz", testArtifact())

		ds, inserted, _, _ := analyzeTestStore([]*fleet.Software{
			{ID: 1, Name: "air", Version: "v1.48.0", ExtensionID: "github.com/air-verse/air"},
		})

		vulns, err := Analyze(t.Context(), ds, dir, false, time.Now(), logger)
		require.NoError(t, err)
		require.Empty(t, vulns)
		require.Len(t, *inserted, 2)
	})

	t.Run("stale rows are kept when the insert fails", func(t *testing.T) {
		dir := t.TempDir()
		writeArtifact(t, dir, "govulndb-2026-09-09.json.gz", testArtifact())

		ds, _, _, _ := analyzeTestStore([]*fleet.Software{
			{ID: 1, Name: "air", Version: "v1.48.0", ExtensionID: "github.com/air-verse/air"},
		})
		ds.InsertSoftwareVulnerabilitiesFunc = func(
			ctx context.Context, vulns []fleet.SoftwareVulnerability, source fleet.VulnerabilitySource,
		) ([]fleet.SoftwareVulnerability, error) {
			return nil, errors.New("insert failed")
		}

		_, err := Analyze(t.Context(), ds, dir, true, time.Now(), logger)
		require.ErrorContains(t, err, "insert failed")
		require.False(t, ds.DeleteOutOfDateVulnerabilitiesFuncInvoked)
	})

	t.Run("no matches still removes stale rows", func(t *testing.T) {
		dir := t.TempDir()
		writeArtifact(t, dir, "govulndb-2026-09-09.json.gz", testArtifact())

		ds, inserted, _, deletedOlderThan := analyzeTestStore([]*fleet.Software{
			{ID: 1, Name: "gopls", Version: "v0.20.0", ExtensionID: "golang.org/x/tools/gopls", Release: "go1.26.1"},
		})

		vulns, err := Analyze(t.Context(), ds, dir, true, time.Now(), logger)
		require.NoError(t, err)
		require.Empty(t, vulns)
		require.Empty(t, *inserted)
		require.NotNil(t, *deletedOlderThan)
	})
}
