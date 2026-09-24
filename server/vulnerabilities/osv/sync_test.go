package osv

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/vulnrepo"
	"github.com/stretchr/testify/require"
)

func TestRemoveOldArtifacts(t *testing.T) {
	today := time.Date(2026, 3, 30, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		prefix   string
		upToDate []string
		files    []string
		dirs     []string
		wantGone []string
	}{
		{
			name:     "removes older artifacts of up-to-date versions only",
			prefix:   OSVFilePrefix,
			upToDate: []string{"2204"},
			files: []string{
				"osv-ubuntu-2204-2026-03-30.json.gz",
				"osv-ubuntu-2204-2026-03-29.json.gz",
				"osv-ubuntu-2204-2026-03-28.json.gz",
				// 2404's download failed, so its last-known-good artifact has to survive.
				"osv-ubuntu-2404-2026-03-29.json.gz",
				"osv-ubuntu-2204-delta-2026-03-29.json.gz",
				"some-other-file.json",
			},
			dirs:     []string{"osv-ubuntu-2204-2026-03-01.json.gz"},
			wantGone: []string{"osv-ubuntu-2204-2026-03-29.json.gz", "osv-ubuntu-2204-2026-03-28.json.gz"},
		},
		{
			// The cron ran on March 30 but the release only carried March 29 artifacts, so every
			// version is NotInRelease rather than up to date and nothing may be removed.
			name:   "nothing up to date leaves everything in place",
			prefix: OSVFilePrefix,
			files:  []string{"osv-ubuntu-2404-2026-03-29.json.gz"},
		},
		{
			name:     "other families are not touched",
			prefix:   OSVRHELFilePrefix,
			upToDate: []string{"9"},
			files: []string{
				"osv-rhel-9-2026-03-30.json.gz",
				"osv-rhel-9-2026-03-29.json.gz",
				"osv-rhel-8-2026-03-29.json.gz",
				"osv-ubuntu-2204-2026-03-29.json.gz",
				"osv-android-16-2026-03-29.json.gz",
			},
			wantGone: []string{"osv-rhel-9-2026-03-29.json.gz"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			for _, f := range tt.files {
				require.NoError(t, os.WriteFile(filepath.Join(dir, f), []byte("test"), 0o644))
			}
			for _, d := range tt.dirs {
				require.NoError(t, os.Mkdir(filepath.Join(dir, d), 0o755))
			}

			require.NoError(t, removeOldArtifacts(tt.prefix, today, dir, tt.upToDate))

			gone := make(map[string]struct{}, len(tt.wantGone))
			for _, f := range tt.wantGone {
				gone[f] = struct{}{}
			}
			for _, f := range append(tt.files, tt.dirs...) {
				_, err := os.Stat(filepath.Join(dir, f))
				if _, want := gone[f]; want {
					require.Truef(t, os.IsNotExist(err), "%s should have been removed", f)
				} else {
					require.NoErrorf(t, err, "%s should have been kept", f)
				}
			}
		})
	}
}

func TestArtifactFilename(t *testing.T) {
	date := time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC)

	for _, tt := range []struct{ prefix, version, want string }{
		{OSVFilePrefix, "2204", "osv-ubuntu-2204-2026-04-08.json.gz"},
		{OSVRHELFilePrefix, "10", "osv-rhel-10-2026-04-08.json.gz"},
		{OSVAndroidFilePrefix, "8.1", "osv-android-8.1-2026-04-08.json.gz"},
		{OSVAndroidFilePrefix, "12L", "osv-android-12L-2026-04-08.json.gz"},
	} {
		require.Equal(t, tt.want, artifactFilename(tt.prefix, tt.version, date))
	}
}

func TestGetNeededUbuntuVersions(t *testing.T) {
	tests := []struct {
		name     string
		osVers   *fleet.OSVersions
		expected []string
	}{
		{
			name: "multiple ubuntu versions",
			osVers: &fleet.OSVersions{
				OSVersions: []fleet.OSVersion{
					{Platform: "ubuntu", Version: "22.04.8 LTS"},
					{Platform: "ubuntu", Version: "20.04.1 LTS"},
				},
			},
			expected: []string{"2204", "2004"},
		},
		{
			name: "duplicate ubuntu versions",
			osVers: &fleet.OSVersions{
				OSVersions: []fleet.OSVersion{
					{Platform: "ubuntu", Version: "22.04.8 LTS"},
					{Platform: "ubuntu", Version: "22.04.3 LTS"},
					{Platform: "ubuntu", Version: "22.04.1 LTS"},
				},
			},
			expected: []string{"2204"},
		},
		{
			name: "non-ubuntu platforms ignored",
			osVers: &fleet.OSVersions{
				OSVersions: []fleet.OSVersion{
					{Platform: "ubuntu", Version: "22.04.8 LTS"},
					{Platform: "rhel", Version: "8.5"},
					{Platform: "windows", Version: "10.0.19041"},
				},
			},
			expected: []string{"2204"},
		},
		{
			name: "no ubuntu platforms",
			osVers: &fleet.OSVersions{
				OSVersions: []fleet.OSVersion{
					{Platform: "rhel", Version: "8.5"},
					{Platform: "windows", Version: "10.0.19041"},
				},
			},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getNeededUbuntuVersions(tt.osVers)
			require.ElementsMatch(t, tt.expected, result)
		})
	}
}

func TestGetNeededRHELVersions(t *testing.T) {
	tests := []struct {
		name     string
		osVers   *fleet.OSVersions
		expected []string
	}{
		{
			name: "multiple RHEL versions",
			osVers: &fleet.OSVersions{
				OSVersions: []fleet.OSVersion{
					{Platform: "rhel", Name: "Red Hat Enterprise Linux 8.10.0", Version: "8.10.0"},
					{Platform: "rhel", Name: "Red Hat Enterprise Linux 9.4.0", Version: "9.4.0"},
				},
			},
			expected: []string{"8", "9"},
		},
		{
			name: "duplicate major versions deduplicated",
			osVers: &fleet.OSVersions{
				OSVersions: []fleet.OSVersion{
					{Platform: "rhel", Name: "Red Hat Enterprise Linux 9.2.0", Version: "9.2.0"},
					{Platform: "rhel", Name: "Red Hat Enterprise Linux 9.4.0", Version: "9.4.0"},
				},
			},
			expected: []string{"9"},
		},
		{
			name: "Fedora skipped",
			osVers: &fleet.OSVersions{
				OSVersions: []fleet.OSVersion{
					{Platform: "rhel", Name: "Red Hat Enterprise Linux 9.4.0", Version: "9.4.0"},
					{Platform: "rhel", Name: "Fedora Linux 36.0.0", Version: "36.0.0"},
				},
			},
			expected: []string{"9"},
		},
		{
			name: "non-RHEL platforms ignored",
			osVers: &fleet.OSVersions{
				OSVersions: []fleet.OSVersion{
					{Platform: "rhel", Name: "Red Hat Enterprise Linux 9.4.0", Version: "9.4.0"},
					{Platform: "ubuntu", Name: "Ubuntu 22.04.8 LTS", Version: "22.04.8 LTS"},
					{Platform: "windows", Name: "Windows 10", Version: "10.0.19041"},
				},
			},
			expected: []string{"9"},
		},
		{
			name: "no RHEL platforms",
			osVers: &fleet.OSVersions{
				OSVersions: []fleet.OSVersion{
					{Platform: "ubuntu", Name: "Ubuntu 22.04.8 LTS", Version: "22.04.8 LTS"},
				},
			},
			expected: []string{},
		},
		{
			name: "only Fedora",
			osVers: &fleet.OSVersions{
				OSVersions: []fleet.OSVersion{
					{Platform: "rhel", Name: "Fedora Linux 36.0.0", Version: "36.0.0"},
				},
			},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getNeededRHELVersions(tt.osVers)
			require.ElementsMatch(t, tt.expected, result)
		})
	}
}

func TestSyncOSVFaultTolerance(t *testing.T) {
	tmpDir := t.TempDir()
	date := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)

	// Create a mock release with only some artifacts available
	release := &vulnrepo.Release{
		TagName: "cve-202604010000",
		Assets: map[string]*vulnrepo.Asset{
			"osv-ubuntu-2204-2026-04-01.json.gz": {
				Name:   "osv-ubuntu-2204-2026-04-01.json.gz",
				ID:     12345,
				Digest: "sha256:abc123",
			},
		},
	}

	versions := []string{"2204", "2504"}

	// Mock download function that always fails
	mockDownload := func(ctx context.Context, assetID int64, dstPath string) error {
		return errors.New("mock download failure")
	}

	result, err := syncOSVWithDownloader(context.Background(), tmpDir, OSVFilePrefix, versions, date, release, mockDownload)
	require.Error(t, err)
	require.NotNil(t, result)

	require.Contains(t, result.NotInRelease, "2504", "2504 artifact not in release")
	require.Contains(t, result.Failed, "2204", "2204 download failed, should be in Failed")
}

func TestSyncOSVChecksumMatch(t *testing.T) {
	tmpDir := t.TempDir()
	date := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)

	// Create a test file with known content
	testContent := []byte("test content")
	filename := "osv-ubuntu-2204-2026-04-01.json.gz"
	testFile := filepath.Join(tmpDir, filename)
	err := os.WriteFile(testFile, testContent, 0o644)
	require.NoError(t, err)

	// Compute the digest
	digest, err := vulnrepo.FileSHA256(testFile)
	require.NoError(t, err)

	// Create a mock release with matching digest
	release := &vulnrepo.Release{
		TagName: "cve-202604010000",
		Assets: map[string]*vulnrepo.Asset{
			filename: {
				Name:   filename,
				ID:     12345,
				Digest: digest,
			},
		},
	}

	result, err := syncArtifacts(context.Background(), tmpDir, OSVFilePrefix, []string{"2204"}, date, release)
	require.NoError(t, err)
	require.NotNil(t, result)

	require.Contains(t, result.Skipped, "2204")
	require.Empty(t, result.Downloaded)
	require.Empty(t, result.Failed)
}

func TestSyncOSVPartialFailureNotReturnedAsError(t *testing.T) {
	// Documents the behavior RefreshAll guards against: syncArtifacts reports per-
	// version failures via SyncResult.Failed but does NOT return an error when
	// some downloads succeed.
	tmpDir := t.TempDir()
	date := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)

	good := "osv-ubuntu-2204-2026-04-01.json.gz"
	bad := "osv-ubuntu-2404-2026-04-01.json.gz"

	release := &vulnrepo.Release{
		TagName: "cve-202604010000",
		Assets: map[string]*vulnrepo.Asset{
			good: {Name: good, ID: 1},
			bad:  {Name: bad, ID: 2},
		},
	}

	mockDownload := func(ctx context.Context, assetID int64, dstPath string) error {
		if assetID == 2 {
			return errors.New("simulated download failure")
		}
		return os.WriteFile(dstPath, []byte("ok"), 0o644)
	}

	result, err := syncOSVWithDownloader(context.Background(), tmpDir, OSVFilePrefix, []string{"2204", "2404"}, date, release, mockDownload)
	require.NoError(t, err, "syncArtifacts does not return error on partial failure")
	require.Contains(t, result.Downloaded, "2204")
	require.Contains(t, result.Failed, "2404")
}

// TestIsOSVReleaseAsset guards getLatestRelease's asset filter. Android assets
// were silently dropped because the filter only matched Ubuntu and RHEL
// prefixes, so RefreshAndroid could never find an asset to download.
func TestIsOSVReleaseAsset(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"osv-ubuntu-2204-2026-07-14.json.gz", true},
		{"osv-rhel-9-2026-07-14.json.gz", true},
		{"osv-android-16-2026-07-14.json.gz", true},
		{"osv-android-8.1-2026-07-14.json.gz", true},
		// delta artifacts are excluded.
		{"osv-ubuntu-2204-delta-2026-07-14.json.gz", false},
		{"osv-android-16-delta-2026-07-14.json.gz", false},
		// unrelated assets are excluded.
		{"osv-2026-07-14.json.gz", false},
		{"some-other-file.json.gz", false},
		{"", false},
	}
	for _, tt := range tests {
		require.Equalf(t, tt.want, isOSVReleaseAsset(tt.name), "isOSVReleaseAsset(%q)", tt.name)
	}
}

func TestVersionsFromRelease(t *testing.T) {
	release := &vulnrepo.Release{
		TagName: "cve-202604270000",
		Assets: map[string]*vulnrepo.Asset{
			"osv-ubuntu-2204-2026-04-27.json.gz": {Name: "osv-ubuntu-2204-2026-04-27.json.gz"},
			"osv-ubuntu-2404-2026-04-27.json.gz": {Name: "osv-ubuntu-2404-2026-04-27.json.gz"},
			"osv-rhel-8-2026-04-27.json.gz":      {Name: "osv-rhel-8-2026-04-27.json.gz"},
			"osv-rhel-9-2026-04-27.json.gz":      {Name: "osv-rhel-9-2026-04-27.json.gz"},
			"osv-android-15-2026-04-27.json.gz":  {Name: "osv-android-15-2026-04-27.json.gz"},
			"osv-android-16-2026-04-27.json.gz":  {Name: "osv-android-16-2026-04-27.json.gz"},
		},
	}

	versions := versionsFromRelease(release)
	require.Len(t, versions, 3)
	require.ElementsMatch(t, []string{"2204", "2404"}, versions[OSVFilePrefix])
	require.ElementsMatch(t, []string{"8", "9"}, versions[OSVRHELFilePrefix])
	require.ElementsMatch(t, []string{"15", "16"}, versions[OSVAndroidFilePrefix])

	require.Empty(t, versionsFromRelease(&vulnrepo.Release{Assets: map[string]*vulnrepo.Asset{}}))
}

func TestVersionFromAssetName(t *testing.T) {
	tests := []struct {
		name     string
		prefix   string
		expected string
	}{
		{"osv-ubuntu-2204-2026-04-27.json.gz", OSVFilePrefix, "2204"},
		{"osv-rhel-9-2026-04-27.json.gz", OSVRHELFilePrefix, "9"},
		{"osv-rhel-10-2026-04-27.json.gz", OSVRHELFilePrefix, "10"},
		{"osv-ubuntu-.json.gz", OSVFilePrefix, ""},
		{"osv-ubuntu", OSVFilePrefix, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.expected, versionFromAssetName(tt.name, tt.prefix))
		})
	}
}

func TestReleaseDateFromAssets(t *testing.T) {
	release := &vulnrepo.Release{
		Assets: map[string]*vulnrepo.Asset{
			"osv-ubuntu-2204-2026-04-27.json.gz": {Name: "osv-ubuntu-2204-2026-04-27.json.gz"},
		},
	}

	d, ok := releaseDateFromAssets(release)
	require.True(t, ok)
	require.Equal(t, time.Date(2026, 4, 27, 0, 0, 0, 0, time.UTC), d)

	emptyRelease := &vulnrepo.Release{Assets: map[string]*vulnrepo.Asset{}}
	_, ok = releaseDateFromAssets(emptyRelease)
	require.False(t, ok)
}

// TestRefreshAndroidUsesReleaseDate guards the regression where RefreshAndroid
// built artifact filenames from the cron execution time instead of the release
// date. When the two differ (the cron runs on any day after the release was
// cut), filenames derived from "now" don't match the release assets, so nothing
// downloads — and the cleanup would delete a release-dated artifact that was
// downloaded. This exercises the sync + cleanup path RefreshAndroid delegates to.
func TestRefreshAndroidUsesReleaseDate(t *testing.T) {
	releaseDate := time.Date(2026, 7, 14, 0, 0, 0, 0, time.UTC)
	now := time.Date(2026, 7, 16, 0, 0, 0, 0, time.UTC)

	assetName := artifactFilename(OSVAndroidFilePrefix, "16", releaseDate)
	release := &vulnrepo.Release{
		TagName: "cve-202607140000",
		Assets: map[string]*vulnrepo.Asset{
			assetName: {Name: assetName, ID: 1},
		},
	}

	mockDownload := func(ctx context.Context, assetID int64, dstPath string) error {
		return os.WriteFile(dstPath, []byte("ok"), 0o644)
	}

	// Using the release date finds the asset and downloads it.
	dir := t.TempDir()
	result, err := syncOSVWithDownloader(context.Background(), dir, OSVAndroidFilePrefix, []string{"16"}, releaseDate, release, mockDownload)
	require.NoError(t, err)
	require.Contains(t, result.Downloaded, "16")
	require.Empty(t, result.NotInRelease)

	// Using "now" (different from the release date) misses the asset entirely.
	nowDir := t.TempDir()
	nowResult, err := syncOSVWithDownloader(context.Background(), nowDir, OSVAndroidFilePrefix, []string{"16"}, now, release, mockDownload)
	require.NoError(t, err)
	require.Contains(t, nowResult.NotInRelease, "16")
	require.Empty(t, nowResult.Downloaded)

	// Cleanup with the release date preserves the just-downloaded artifact...
	require.NoError(t, removeOldArtifacts(OSVAndroidFilePrefix, releaseDate, dir, []string{"16"}))
	_, err = os.Stat(filepath.Join(dir, assetName))
	require.NoError(t, err, "release-dated artifact must be preserved")

	// ...whereas cleaning up with "now" would delete it, since its date suffix
	// doesn't match and version 16 is in the successful set.
	require.NoError(t, removeOldArtifacts(OSVAndroidFilePrefix, now, dir, []string{"16"}))
	_, err = os.Stat(filepath.Join(dir, assetName))
	require.True(t, os.IsNotExist(err), "cleanup keyed on now wrongly deletes the release-dated artifact")
}

func TestGetNeededAndroidVersions(t *testing.T) {
	tests := []struct {
		name     string
		oses     []fleet.OperatingSystem
		expected []string
	}{
		{
			name:     "empty",
			oses:     nil,
			expected: nil,
		},
		{
			name: "non-android ignored",
			oses: []fleet.OperatingSystem{
				{Name: "Ubuntu", Version: "22.04", Platform: "ubuntu"},
				{Name: "Windows", Version: "10.0.19042", Platform: "windows"},
			},
			expected: nil,
		},
		{
			name: "bare versions",
			oses: []fleet.OperatingSystem{
				{Name: "Android", Version: "16", Platform: "android"},
				{Name: "Android", Version: "14", Platform: "android"},
			},
			expected: []string{"16", "14"},
		},
		{
			name: "versions with SPL deduplicated",
			oses: []fleet.OperatingSystem{
				{Name: "Android", Version: "16 (2026-01-01)", Platform: "android"},
				{Name: "Android", Version: "16 (2026-05-01)", Platform: "android"},
				{Name: "Android", Version: "14 (2025-03-01)", Platform: "android"},
			},
			expected: []string{"16", "14"},
		},
		{
			name: "empty version skipped",
			oses: []fleet.OperatingSystem{
				{Name: "Android", Version: "", Platform: "android"},
				{Name: "Android", Version: "16", Platform: "android"},
			},
			expected: []string{"16"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getNeededAndroidVersions(tt.oses)
			require.ElementsMatch(t, tt.expected, result)
		})
	}
}

func TestExtractAndroidMajorVersion(t *testing.T) {
	tests := []struct {
		version  string
		expected string
	}{
		{"16 (2026-05-01)", "16"},
		{"14", "14"},
		{"8.1 (2021-01-01)", "8.1"},
		{"12L (2022-12-01)", "12L"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			require.Equal(t, tt.expected, extractAndroidMajorVersion(tt.version))
		})
	}
}

func TestDateFromAssetName(t *testing.T) {
	tests := []struct {
		name     string
		expected time.Time
		ok       bool
	}{
		{"osv-ubuntu-2204-2026-04-27.json.gz", time.Date(2026, 4, 27, 0, 0, 0, 0, time.UTC), true},
		{"osv-rhel-9-2026-04-08.json.gz", time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC), true},
		{"osv-ubuntu-2204.json.gz", time.Time{}, false},
		{"some-other-file.json", time.Time{}, false},
		{"short", time.Time{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d, ok := dateFromAssetName(tt.name)
			require.Equal(t, tt.ok, ok)
			require.Equal(t, tt.expected, d)
		})
	}
}
