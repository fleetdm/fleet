package android

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/stretchr/testify/require"
)

func TestParseAndroidVersion(t *testing.T) {
	tests := []struct {
		version   string
		wantMajor string
		wantSPL   string
	}{
		{"16 (2026-05-01)", "16", "2026-05-01"},
		{"14 (2024-09-01)", "14", "2024-09-01"},
		{"16", "16", ""},
		{"8.1 (2021-01-01)", "8.1", "2021-01-01"},
		{"12L (2022-12-01)", "12L", "2022-12-01"},
		{"", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			major, spl := parseAndroidVersion(tt.version)
			require.Equal(t, tt.wantMajor, major)
			require.Equal(t, tt.wantSPL, spl)
		})
	}
}

func TestResolvedVersion(t *testing.T) {
	got := resolvedVersion("16", "2026-06-01")
	require.NotNil(t, got)
	require.Equal(t, "16 (2026-06-01)", *got)
}

func writeTestArtifact(t *testing.T, dir string, artifact *AndroidArtifact) string {
	t.Helper()
	filename := filepath.Join(dir, "osv-android-"+artifact.AndroidVersion+"-2026-07-14.json.gz")

	f, err := os.Create(filename)
	require.NoError(t, err)
	defer f.Close()

	gz := gzip.NewWriter(f)
	defer gz.Close()

	require.NoError(t, json.NewEncoder(gz).Encode(artifact))
	return filename
}

func TestAnalyze(t *testing.T) {
	ctx := t.Context()
	vulnDir := t.TempDir()

	// Create artifact for Android 16 with two CVEs.
	writeTestArtifact(t, vulnDir, &AndroidArtifact{
		SchemaVersion:  "1.0",
		AndroidVersion: "16",
		Generated:      "2026-07-14T00:00:00Z",
		TotalCVEs:      2,
		Vulnerabilities: []AndroidVuln{
			{CVE: "CVE-2026-1111", FixedSPL: "2026-05-01", Severity: "High"},
			{CVE: "CVE-2026-2222", FixedSPL: "2026-06-01", Severity: "Critical"},
		},
	})

	t.Run("host with old SPL is vulnerable to both", func(t *testing.T) {
		ds := new(mock.Store)
		ds.ListOSVulnerabilitiesByOSFunc = func(ctx context.Context, osID uint) ([]fleet.OSVulnerability, error) {
			return nil, nil // no existing vulns
		}
		ds.DeleteOSVulnerabilitiesFunc = func(ctx context.Context, vulns []fleet.OSVulnerability) error {
			return nil
		}
		var inserted []fleet.OSVulnerability
		ds.InsertOSVulnerabilitiesFunc = func(ctx context.Context, vulns []fleet.OSVulnerability, source fleet.VulnerabilitySource) (int64, error) {
			inserted = append(inserted, vulns...)
			return int64(len(vulns)), nil
		}

		os := fleet.OperatingSystem{
			ID:       1,
			Name:     "Android",
			Version:  "16 (2026-04-01)", // SPL before both fixes
			Platform: "android",
		}

		result, err := Analyze(ctx, ds, os, vulnDir, true, nil, NewArtifactCache())
		require.NoError(t, err)
		require.Len(t, result, 2)

		cves := map[string]string{}
		for _, v := range result {
			require.Equal(t, uint(1), v.OSID)
			require.Equal(t, fleet.AndroidOSVSource, v.Source)
			cves[v.CVE] = *v.ResolvedInVersion
		}
		require.Equal(t, "16 (2026-05-01)", cves["CVE-2026-1111"])
		require.Equal(t, "16 (2026-06-01)", cves["CVE-2026-2222"])
	})

	t.Run("host with recent SPL is only vulnerable to later fix", func(t *testing.T) {
		ds := new(mock.Store)
		ds.ListOSVulnerabilitiesByOSFunc = func(ctx context.Context, osID uint) ([]fleet.OSVulnerability, error) {
			return nil, nil
		}
		ds.DeleteOSVulnerabilitiesFunc = func(ctx context.Context, vulns []fleet.OSVulnerability) error {
			return nil
		}
		var inserted []fleet.OSVulnerability
		ds.InsertOSVulnerabilitiesFunc = func(ctx context.Context, vulns []fleet.OSVulnerability, source fleet.VulnerabilitySource) (int64, error) {
			inserted = append(inserted, vulns...)
			return int64(len(vulns)), nil
		}

		os := fleet.OperatingSystem{
			ID:       2,
			Name:     "Android",
			Version:  "16 (2026-05-01)", // SPL after first fix, before second
			Platform: "android",
		}

		result, err := Analyze(ctx, ds, os, vulnDir, true, nil, NewArtifactCache())
		require.NoError(t, err)
		require.Len(t, result, 1)
		require.Equal(t, "CVE-2026-2222", result[0].CVE)
	})

	t.Run("fully patched host has no vulnerabilities", func(t *testing.T) {
		ds := new(mock.Store)
		ds.ListOSVulnerabilitiesByOSFunc = func(ctx context.Context, osID uint) ([]fleet.OSVulnerability, error) {
			return nil, nil
		}
		ds.DeleteOSVulnerabilitiesFunc = func(ctx context.Context, vulns []fleet.OSVulnerability) error {
			return nil
		}
		ds.InsertOSVulnerabilitiesFunc = func(ctx context.Context, vulns []fleet.OSVulnerability, source fleet.VulnerabilitySource) (int64, error) {
			return 0, nil
		}

		os := fleet.OperatingSystem{
			ID:       3,
			Name:     "Android",
			Version:  "16 (2026-06-01)", // SPL at the latest fix
			Platform: "android",
		}

		result, err := Analyze(ctx, ds, os, vulnDir, true, nil, NewArtifactCache())
		require.NoError(t, err)
		require.Empty(t, result)
	})

	t.Run("bare version with no SPL skips matching", func(t *testing.T) {
		ds := new(mock.Store)
		ds.ListOSVulnerabilitiesByOSFunc = func(ctx context.Context, osID uint) ([]fleet.OSVulnerability, error) {
			return nil, nil
		}
		ds.DeleteOSVulnerabilitiesFunc = func(ctx context.Context, vulns []fleet.OSVulnerability) error {
			return nil
		}
		ds.InsertOSVulnerabilitiesFunc = func(ctx context.Context, vulns []fleet.OSVulnerability, source fleet.VulnerabilitySource) (int64, error) {
			return 0, nil
		}

		os := fleet.OperatingSystem{
			ID:       4,
			Name:     "Android",
			Version:  "16", // No SPL
			Platform: "android",
		}

		result, err := Analyze(ctx, ds, os, vulnDir, true, nil, NewArtifactCache())
		require.NoError(t, err)
		require.Empty(t, result)
	})

	t.Run("delta removes stale vulns when host is patched", func(t *testing.T) {
		ds := new(mock.Store)
		// Simulate existing vuln from a previous scan
		ds.ListOSVulnerabilitiesByOSFunc = func(ctx context.Context, osID uint) ([]fleet.OSVulnerability, error) {
			return []fleet.OSVulnerability{
				{
					OSID:              5,
					CVE:               "CVE-2026-1111",
					Source:            fleet.AndroidOSVSource,
					ResolvedInVersion: resolvedVersion("16", "2026-05-01"),
				},
				{
					OSID:              5,
					CVE:               "CVE-2026-2222",
					Source:            fleet.AndroidOSVSource,
					ResolvedInVersion: resolvedVersion("16", "2026-06-01"),
				},
			}, nil
		}
		var deleted []fleet.OSVulnerability
		ds.DeleteOSVulnerabilitiesFunc = func(ctx context.Context, vulns []fleet.OSVulnerability) error {
			deleted = append(deleted, vulns...)
			return nil
		}
		ds.InsertOSVulnerabilitiesFunc = func(ctx context.Context, vulns []fleet.OSVulnerability, source fleet.VulnerabilitySource) (int64, error) {
			return 0, nil
		}

		// Host is now fully patched
		os := fleet.OperatingSystem{
			ID:       5,
			Name:     "Android",
			Version:  "16 (2026-06-01)",
			Platform: "android",
		}

		_, err := Analyze(ctx, ds, os, vulnDir, false, nil, NewArtifactCache())
		require.NoError(t, err)
		// Both previously existing vulns should be deleted
		require.Len(t, deleted, 2)
	})

	t.Run("bare version (no SPL) leaves existing vulns untouched", func(t *testing.T) {
		ds := new(mock.Store)
		// Existing findings from a previous scan of a patch-level version.
		ds.ListOSVulnerabilitiesByOSFunc = func(ctx context.Context, osID uint) ([]fleet.OSVulnerability, error) {
			return []fleet.OSVulnerability{
				{
					OSID:              8,
					CVE:               "CVE-2026-1111",
					Source:            fleet.AndroidOSVSource,
					ResolvedInVersion: resolvedVersion("16", "2026-05-01"),
				},
			}, nil
		}
		var deleted []fleet.OSVulnerability
		ds.DeleteOSVulnerabilitiesFunc = func(ctx context.Context, vulns []fleet.OSVulnerability) error {
			deleted = append(deleted, vulns...)
			return nil
		}

		// Host reports a bare version with no security patch level, so we
		// can't determine vulnerability status.
		os := fleet.OperatingSystem{
			ID:       8,
			Name:     "Android",
			Version:  "16",
			Platform: "android",
		}

		result, err := Analyze(ctx, ds, os, vulnDir, true, nil, NewArtifactCache())
		require.NoError(t, err)
		require.Nil(t, result)
		require.Empty(t, deleted, "existing findings must not be deleted for a bare version")
		require.False(t, ds.DeleteOSVulnerabilitiesFuncInvoked)
	})

	t.Run("no artifact for version returns nil", func(t *testing.T) {
		ds := new(mock.Store)
		os := fleet.OperatingSystem{
			ID:       6,
			Name:     "Android",
			Version:  "99 (2099-01-01)", // no artifact for version 99
			Platform: "android",
		}

		result, err := Analyze(ctx, ds, os, vulnDir, true, nil, NewArtifactCache())
		require.NoError(t, err)
		require.Nil(t, result)
	})

	t.Run("corrupt artifact propagates the error", func(t *testing.T) {
		corruptDir := t.TempDir()
		// Write a file that is not valid gzip
		require.NoError(t, os.WriteFile(
			filepath.Join(corruptDir, "osv-android-16-2026-07-14.json.gz"),
			[]byte("not gzip data"), 0o644))

		ds := new(mock.Store)
		os := fleet.OperatingSystem{
			ID:       7,
			Name:     "Android",
			Version:  "16 (2026-01-01)",
			Platform: "android",
		}

		result, err := Analyze(ctx, ds, os, corruptDir, true, nil, NewArtifactCache())
		require.Error(t, err) // a corrupt artifact is a real failure, not a missing one
		require.NotErrorIs(t, err, errArtifactNotFound)
		require.Nil(t, result)
	})

	t.Run("empty artifact (zero vulns) is a no-op", func(t *testing.T) {
		emptyDir := t.TempDir()
		writeTestArtifact(t, emptyDir, &AndroidArtifact{
			SchemaVersion:   "1.0",
			AndroidVersion:  "16",
			Generated:       "2026-07-14T00:00:00Z",
			TotalCVEs:       0,
			Vulnerabilities: nil,
		})

		ds := new(mock.Store)
		os := fleet.OperatingSystem{
			ID:       8,
			Name:     "Android",
			Version:  "16 (2026-01-01)",
			Platform: "android",
		}

		result, err := Analyze(ctx, ds, os, emptyDir, true, nil, NewArtifactCache())
		require.NoError(t, err)
		require.Nil(t, result)
		// No datastore methods should have been called
		require.False(t, ds.ListOSVulnerabilitiesByOSFuncInvoked)
	})
}

func TestLoadArtifactPicksLatest(t *testing.T) {
	dir := t.TempDir()

	// Write two artifacts for the same version with different dates
	writeTestArtifact(t, dir, &AndroidArtifact{
		SchemaVersion:  "1.0",
		AndroidVersion: "16",
		Generated:      "2026-07-13T00:00:00Z",
		TotalCVEs:      1,
		Vulnerabilities: []AndroidVuln{
			{CVE: "CVE-2026-0001", FixedSPL: "2026-01-01"},
		},
	})
	// Overwrite with a different date filename
	older := filepath.Join(dir, "osv-android-16-2026-07-13.json.gz")
	newer := filepath.Join(dir, "osv-android-16-2026-07-14.json.gz")
	// Rename the one writeTestArtifact created to the older date
	require.NoError(t, os.Rename(
		filepath.Join(dir, "osv-android-16-2026-07-14.json.gz"),
		older))

	// Write the newer one with 2 CVEs so we can distinguish
	f, err := os.Create(newer)
	require.NoError(t, err)
	gz := gzip.NewWriter(f)
	require.NoError(t, json.NewEncoder(gz).Encode(&AndroidArtifact{
		SchemaVersion:  "1.0",
		AndroidVersion: "16",
		Generated:      "2026-07-14T00:00:00Z",
		TotalCVEs:      2,
		Vulnerabilities: []AndroidVuln{
			{CVE: "CVE-2026-0001", FixedSPL: "2026-01-01"},
			{CVE: "CVE-2026-0002", FixedSPL: "2026-02-01"},
		},
	}))
	require.NoError(t, gz.Close())
	require.NoError(t, f.Close())

	artifact, err := loadArtifact("16", dir)
	require.NoError(t, err)
	require.Equal(t, 2, artifact.TotalCVEs, "should pick the latest (2026-07-14) artifact")
}
