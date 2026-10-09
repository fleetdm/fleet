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

// testArtifact covers the shapes seen in the real database: open-ended, fixed, unfixed and
// CVE-less module advisories, and the multi-range stdlib shape.
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
			StdlibModule: {
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
			name:     "no module path and no toolchain",
			software: fleet.Software{ID: 1, Name: "air", Version: "v1.48.0"},
			wantCVEs: nil,
		},
		{
			name:     "devel version skips only the module lookup",
			software: fleet.Software{ID: 1, Name: "air", Version: "(devel)", ExtensionID: "github.com/air-verse/air", Release: "go1.21.11"},
			wantCVEs: []string{"CVE-2024-24791"},
		},
		{
			// `go install` on an untagged module or a commit records v0.0.0-<pseudo>, which
			// sorts below "0"'s v0.0.0.
			name:     "pseudo-version of an untagged module",
			software: fleet.Software{ID: 1, Name: "air", Version: "v0.0.0-20240101000000-abcdef123456", ExtensionID: "github.com/air-verse/air"},
			wantCVEs: []string{"CVE-2024-1000"},
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
			name:     "release candidate toolchain inside the next minor's range",
			software: fleet.Software{ID: 2, Name: "tool", Version: "v1.0.0", Release: "go1.22rc1"},
			wantCVEs: []string{"CVE-2024-24791"},
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

func TestModuleVersion(t *testing.T) {
	for _, tc := range []struct {
		version string
		want    string
		ok      bool
	}{
		{version: "v1.48.0", want: "v1.48.0", ok: true},
		{version: "1.48.0", want: "v1.48.0", ok: true},
		{version: "v0.0.0-20240101000000-abcdef123456", want: "v0.0.0-20240101000000-abcdef123456", ok: true},
		{version: "(devel)"},
		{version: "not-a-version"},
		{version: ""},
	} {
		t.Run(tc.version, func(t *testing.T) {
			got, ok := moduleVersion(tc.version)
			require.Equal(t, tc.ok, ok)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestToolchainVersion(t *testing.T) {
	for _, tc := range []struct {
		release string
		want    string
		ok      bool
	}{
		{release: "go1.21.12", want: "v1.21.12", ok: true},
		{release: "go1.21", want: "v1.21", ok: true},
		{release: "go1.22rc1", want: "v1.22.0-rc.1", ok: true},
		{release: "go1.22beta1", want: "v1.22.0-beta.1", ok: true},
		// Binaries built with a GOEXPERIMENT report it after the version.
		{release: "go1.21.11 X:boringcrypto", want: "v1.21.11", ok: true},
		{release: "devel go1.27-abcdef1234 Mon Jan 1 00:00:00 2026 +0000"},
		{release: ""},
	} {
		t.Run(tc.release, func(t *testing.T) {
			got, ok := toolchainVersion(tc.release)
			require.Equal(t, tc.ok, ok)
			require.Equal(t, tc.want, got)
		})
	}
}

type fakeSoftwareIterator struct {
	index     int
	softwares []*fleet.Software
}

func (f *fakeSoftwareIterator) Next() bool { return f.index < len(f.softwares) }

func (f *fakeSoftwareIterator) Value() (*fleet.Software, error) {
	s := f.softwares[f.index]
	f.index++
	return s, nil
}

func (f *fakeSoftwareIterator) Err() error   { return nil }
func (f *fakeSoftwareIterator) Close() error { return nil }

// analyzeRecorder captures what the analyzer wrote to the mock datastore.
type analyzeRecorder struct {
	inserted         []fleet.SoftwareVulnerability
	includedSources  []string
	deletedOlderThan *time.Time
}

// analyzeTestStore returns a mock datastore that serves the given Go binaries and records what
// the analyzer writes.
func analyzeTestStore(software []*fleet.Software) (*mock.Store, *analyzeRecorder) {
	ds := new(mock.Store)
	rec := new(analyzeRecorder)

	ds.AllSoftwareIteratorFunc = func(ctx context.Context, q fleet.SoftwareIterQueryOptions) (fleet.SoftwareIterator, error) {
		rec.includedSources = q.IncludedSources
		return &fakeSoftwareIterator{softwares: software}, nil
	}
	ds.InsertSoftwareVulnerabilitiesFunc = func(
		ctx context.Context, vulns []fleet.SoftwareVulnerability, source fleet.VulnerabilitySource,
	) ([]fleet.SoftwareVulnerability, error) {
		if source != fleet.GoVulnDBSource {
			return nil, fmt.Errorf("unexpected source %d", source)
		}
		rec.inserted = append(rec.inserted, vulns...)
		return vulns, nil
	}
	ds.DeleteOutOfDateVulnerabilitiesFunc = func(
		ctx context.Context, source fleet.VulnerabilitySource, olderThan time.Time,
	) error {
		if source != fleet.GoVulnDBSource {
			return fmt.Errorf("unexpected source %d", source)
		}
		rec.deletedOlderThan = &olderThan
		return nil
	}

	return ds, rec
}

func TestAnalyze(t *testing.T) {
	logger := slog.New(slog.DiscardHandler)

	t.Run("no artifact on disk is not an error", func(t *testing.T) {
		ds, _ := analyzeTestStore(nil)

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

		ds, _ := analyzeTestStore(nil)

		_, err := Analyze(t.Context(), ds, dir, true, time.Now(), logger)
		require.ErrorContains(t, err, "schema version")
		require.False(t, ds.AllSoftwareIteratorFuncInvoked)
		require.False(t, ds.DeleteOutOfDateVulnerabilitiesFuncInvoked)
	})

	t.Run("an artifact with no modules is refused", func(t *testing.T) {
		dir := t.TempDir()
		writeArtifact(t, dir, "govulndb-2026-09-09.json.gz", &Artifact{SchemaVersion: "1"})

		ds, _ := analyzeTestStore(nil)

		_, err := Analyze(t.Context(), ds, dir, true, time.Now(), logger)
		require.ErrorContains(t, err, "no modules")
		require.False(t, ds.DeleteOutOfDateVulnerabilitiesFuncInvoked)
	})

	t.Run("matches Go binaries and removes stale rows", func(t *testing.T) {
		dir := t.TempDir()
		writeArtifact(t, dir, "govulndb-2026-09-09.json.gz", testArtifact())

		startTime := time.Now().Add(-time.Hour)
		ds, rec := analyzeTestStore([]*fleet.Software{
			{ID: 1, Name: "air", Version: "v1.48.0", ExtensionID: "github.com/air-verse/air", Release: "go1.26.1"},
			{ID: 2, Name: "air", Version: "v1.49.0", ExtensionID: "github.com/air-verse/air", Release: "go1.21.11"},
			{ID: 3, Name: "devtool", Version: "(devel)", ExtensionID: "github.com/fleetdm/fleet/v4", Release: "go1.26.1"},
		})

		vulns, err := Analyze(t.Context(), ds, dir, true, startTime, logger)
		require.NoError(t, err)

		require.Equal(t, []string{softwareSource}, rec.includedSources)
		require.ElementsMatch(t, []fleet.SoftwareVulnerability{
			{SoftwareID: 1, CVE: "CVE-2024-1000", ResolvedInVersion: new("v1.49.0")},
			{SoftwareID: 1, CVE: "CVE-2026-1002"},
			{SoftwareID: 2, CVE: "CVE-2026-1002"},
			{SoftwareID: 2, CVE: "CVE-2024-24791"},
		}, rec.inserted)
		require.ElementsMatch(t, rec.inserted, vulns)

		require.NotNil(t, rec.deletedOlderThan)
		require.Equal(t, startTime, *rec.deletedOlderThan)
	})

	t.Run("collectVulns off returns nothing but still writes", func(t *testing.T) {
		dir := t.TempDir()
		writeArtifact(t, dir, "govulndb-2026-09-09.json.gz", testArtifact())

		ds, rec := analyzeTestStore([]*fleet.Software{
			{ID: 1, Name: "air", Version: "v1.48.0", ExtensionID: "github.com/air-verse/air"},
		})

		vulns, err := Analyze(t.Context(), ds, dir, false, time.Now(), logger)
		require.NoError(t, err)
		require.Empty(t, vulns)
		require.Len(t, rec.inserted, 2)
	})

	t.Run("stale rows are kept when the insert fails", func(t *testing.T) {
		dir := t.TempDir()
		writeArtifact(t, dir, "govulndb-2026-09-09.json.gz", testArtifact())

		ds, _ := analyzeTestStore([]*fleet.Software{
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

		ds, rec := analyzeTestStore([]*fleet.Software{
			{ID: 1, Name: "gopls", Version: "v0.20.0", ExtensionID: "golang.org/x/tools/gopls", Release: "go1.26.1"},
		})

		vulns, err := Analyze(t.Context(), ds, dir, true, time.Now(), logger)
		require.NoError(t, err)
		require.Empty(t, vulns)
		require.Empty(t, rec.inserted)
		require.NotNil(t, rec.deletedOlderThan)
	})
}
