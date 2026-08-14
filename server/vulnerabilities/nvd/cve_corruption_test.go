package nvd

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/stretchr/testify/require"
)

// TestTranslateCPEToCVE_EmptyFeedDoesNotDelete verifies that when the NVD feed files exist
// but contain no CVEs (e.g., a corrupted/empty artifact from GitHub), TranslateCPEToCVE
// does NOT call DeleteOutOfDateVulnerabilities/DeleteOutOfDateOSVulnerabilities. Otherwise
// existing software_cve rows would be wiped on every failed sync.
// See https://github.com/fleetdm/fleet/issues/45602.
func TestTranslateCPEToCVE_EmptyFeedDoesNotDelete(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()

	emptyFeed := `{"CVE_data_format":"MITRE","CVE_data_type":"CVE","CVE_data_version":"4.0","CVE_Items":[]}`
	require.NoError(t, os.WriteFile(filepath.Join(tempDir, "nvdcve-1.1-2024.json"), []byte(emptyFeed), 0o644))

	ds := new(mock.Store)
	ds.ListSoftwareCPEsFunc = func(ctx context.Context) ([]fleet.SoftwareCPE, error) {
		return []fleet.SoftwareCPE{
			{CPE: "cpe:2.3:a:google:chrome:120.0.0:*:*:*:*:*:*:*", SoftwareID: 1},
		}, nil
	}
	ds.ListOperatingSystemsForPlatformFunc = func(ctx context.Context, p string) ([]fleet.OperatingSystem, error) {
		return nil, nil
	}
	ds.InsertSoftwareVulnerabilitiesFunc = func(ctx context.Context, vulns []fleet.SoftwareVulnerability, src fleet.VulnerabilitySource) ([]fleet.SoftwareVulnerability, error) {
		require.Empty(t, vulns, "no vulns should be inserted from an empty feed")
		return nil, nil
	}
	ds.InsertOSVulnerabilitiesFunc = func(ctx context.Context, vulns []fleet.OSVulnerability, src fleet.VulnerabilitySource) (int64, error) {
		return 0, nil
	}
	ds.DeleteOutOfDateVulnerabilitiesFunc = func(ctx context.Context, source fleet.VulnerabilitySource, olderThan time.Time) error {
		t.Fatalf("DeleteOutOfDateVulnerabilities should NOT be called when feed produces no matches")
		return nil
	}
	ds.DeleteOutOfDateOSVulnerabilitiesFunc = func(ctx context.Context, source fleet.VulnerabilitySource, olderThan time.Time) error {
		t.Fatalf("DeleteOutOfDateOSVulnerabilities should NOT be called when feed produces no matches")
		return nil
	}

	_, err := TranslateCPEToCVE(context.Background(), ds, tempDir, slog.New(slog.DiscardHandler), false, time.Now().UTC().Add(-time.Hour))
	require.NoError(t, err)

	require.False(t, ds.DeleteOutOfDateVulnerabilitiesFuncInvoked,
		"DeleteOutOfDateVulnerabilities must be skipped when feed has no matches (issue #45602)")
	require.False(t, ds.DeleteOutOfDateOSVulnerabilitiesFuncInvoked,
		"DeleteOutOfDateOSVulnerabilities must be skipped when feed has no matches (issue #45602)")
}
