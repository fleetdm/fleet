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
	"github.com/stretchr/testify/assert"
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
			assert.Equal(t, tt.wantMajor, major)
			assert.Equal(t, tt.wantSPL, spl)
		})
	}
}

func TestResolvedVersion(t *testing.T) {
	got := resolvedVersion("16", "2026-06-01")
	require.NotNil(t, got)
	assert.Equal(t, "16 (2026-06-01)", *got)
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

		result, err := Analyze(ctx, ds, os, vulnDir, true, nil)
		require.NoError(t, err)
		require.Len(t, result, 2)

		cves := map[string]string{}
		for _, v := range result {
			require.Equal(t, uint(1), v.OSID)
			require.Equal(t, fleet.AndroidOSVSource, v.Source)
			cves[v.CVE] = *v.ResolvedInVersion
		}
		assert.Equal(t, "16 (2026-05-01)", cves["CVE-2026-1111"])
		assert.Equal(t, "16 (2026-06-01)", cves["CVE-2026-2222"])
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

		result, err := Analyze(ctx, ds, os, vulnDir, true, nil)
		require.NoError(t, err)
		require.Len(t, result, 1)
		assert.Equal(t, "CVE-2026-2222", result[0].CVE)
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

		result, err := Analyze(ctx, ds, os, vulnDir, true, nil)
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

		result, err := Analyze(ctx, ds, os, vulnDir, true, nil)
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

		_, err := Analyze(ctx, ds, os, vulnDir, false, nil)
		require.NoError(t, err)
		// Both previously existing vulns should be deleted
		require.Len(t, deleted, 2)
	})

	t.Run("no artifact for version returns nil", func(t *testing.T) {
		ds := new(mock.Store)
		os := fleet.OperatingSystem{
			ID:       6,
			Name:     "Android",
			Version:  "99 (2099-01-01)", // no artifact for version 99
			Platform: "android",
		}

		result, err := Analyze(ctx, ds, os, vulnDir, true, nil)
		require.NoError(t, err)
		require.Nil(t, result)
	})
}
