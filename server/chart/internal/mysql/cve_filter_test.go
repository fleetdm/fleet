package mysql

import (
	"crypto/sha256"
	"testing"

	"github.com/fleetdm/fleet/v4/server/chart/api"
	"github.com/fleetdm/fleet/v4/server/chart/internal/testutils"
	"github.com/fleetdm/fleet/v4/server/chart/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// seedSoftware inserts a `software` row and a linking `software_cve` row, so a
// CVE is attributed to software of the given name+source. Returns nothing — the
// resolver/collector queries match on name/source, not id.
func seedSoftware(t *testing.T, tdb *testutils.TestDB, name, source, cve string) {
	t.Helper()
	ctx := t.Context()

	// checksum is binary(16) UNIQUE NOT NULL; derive it from the row's own
	// identifying inputs so each seeded row is unique.
	sum := sha256.Sum256([]byte(name + "\x00" + source + "\x00" + cve))
	checksum := sum[:16]

	res, err := tdb.DB.ExecContext(ctx,
		`INSERT INTO software (name, version, source, checksum) VALUES (?, '1.0', ?, ?)`,
		name, source, checksum)
	require.NoError(t, err)
	swID, err := res.LastInsertId()
	require.NoError(t, err)

	_, err = tdb.DB.ExecContext(ctx,
		`INSERT INTO software_cve (software_id, cve) VALUES (?, ?)`, swID, cve)
	require.NoError(t, err)
}

// seedOSVuln attributes a CVE to an operating system via
// operating_system_vulnerabilities.
func seedOSVuln(t *testing.T, tdb *testutils.TestDB, osID uint, cve string) {
	t.Helper()
	_, err := tdb.DB.ExecContext(t.Context(),
		`INSERT INTO operating_system_vulnerabilities (operating_system_id, cve) VALUES (?, ?)`,
		osID, cve)
	require.NoError(t, err)
}

// seedCVEMeta inserts cve_meta for a CVE. Pass knownExploit to set the CISA flag.
func seedCVEMeta(t *testing.T, tdb *testutils.TestDB, cve string, cvss, epss float64, knownExploit bool) {
	t.Helper()
	_, err := tdb.DB.ExecContext(t.Context(),
		`INSERT INTO cve_meta (cve, cvss_score, epss_probability, cisa_known_exploit) VALUES (?, ?, ?, ?)`,
		cve, cvss, epss, knownExploit)
	require.NoError(t, err)
}

// TestCollectibleCVEs verifies the wide collection set: all severities on
// tracked software and OS vulnerabilities are collected (even without cve_meta),
// while CVEs on untracked software are excluded.
func TestCollectibleCVEs(t *testing.T) {
	tdb := testutils.SetupTestDB(t, "chart_mysql")
	defer tdb.TruncateTables(t)
	ds := NewDatastore(tdb.Conns(), tdb.Logger)
	ctx := t.Context()

	// Tracked software, low-severity CVE — collected despite low/absent severity.
	seedSoftware(t, tdb, "Google Chrome", "apps", "CVE-2026-1000")
	seedCVEMeta(t, tdb, "CVE-2026-1000", 3.0, 0.1, false)
	// Tracked software, no cve_meta at all — still collected (severity unknown).
	seedSoftware(t, tdb, "Adobe Acrobat", "programs", "CVE-2026-1001")
	// OS vulnerability — collected.
	seedOSVuln(t, tdb, 1, "CVE-2026-1002")
	// Untracked software — NOT collected.
	seedSoftware(t, tdb, "Slack", "apps", "CVE-2026-9000")

	got, err := ds.CollectibleCVEs(ctx)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"CVE-2026-1000", "CVE-2026-1001", "CVE-2026-1002"}, got)
	assert.NotContains(t, got, "CVE-2026-9000", "CVE on untracked software must not be collected")
}

// TestResolveCVEChartEntities exercises the read-time narrowing across every
// filter dimension. The fixture (all critical unless noted):
//
//	CVE-A  Chrome (browsers)   cvss 9.5  epss 0.80  kev=true
//	CVE-B  Firefox (browsers)  cvss 3.0  epss 0.10  kev=false  (low severity)
//	CVE-C  MS Word (office)    cvss 9.9  epss 0.20  kev=false
//	CVE-D  kernel-x (rpm, OS)  cvss 9.1  epss 0.50  kev=false
//	CVE-E  OS vuln (OS)        cvss 9.7  epss 0.90  kev=true
//	CVE-F  Slack (untracked)   cvss 10.0 epss 0.99  kev=true   (never tracked)
func TestResolveCVEChartEntities(t *testing.T) {
	tdb := testutils.SetupTestDB(t, "chart_mysql")
	defer tdb.TruncateTables(t)
	ds := NewDatastore(tdb.Conns(), tdb.Logger)
	ctx := t.Context()

	seedSoftware(t, tdb, "Google Chrome", "apps", "CVE-A")
	seedCVEMeta(t, tdb, "CVE-A", 9.5, 0.80, true)
	seedSoftware(t, tdb, "Firefox", "apps", "CVE-B")
	seedCVEMeta(t, tdb, "CVE-B", 3.0, 0.10, false)
	seedSoftware(t, tdb, "Microsoft Word", "programs", "CVE-C")
	seedCVEMeta(t, tdb, "CVE-C", 9.9, 0.20, false)
	seedSoftware(t, tdb, "kernel-default", "rpm_packages", "CVE-D")
	seedCVEMeta(t, tdb, "CVE-D", 9.1, 0.50, false)
	seedOSVuln(t, tdb, 1, "CVE-E")
	seedCVEMeta(t, tdb, "CVE-E", 9.7, 0.90, true)
	seedSoftware(t, tdb, "Slack", "apps", "CVE-F")
	seedCVEMeta(t, tdb, "CVE-F", 10.0, 0.99, true)

	// critical is the default service-forced severity band.
	const critMin, critMax = 9.0, 10.0
	allCats := types.CVEChartFilter{CVSSMin: critMin, CVSSMax: critMax}

	cases := []struct {
		name   string
		filter types.CVEChartFilter
		want   []string
	}{
		{
			name:   "default critical, all categories",
			filter: allCats,
			want:   []string{"CVE-A", "CVE-C", "CVE-D", "CVE-E"}, // B low-severity, F untracked
		},
		{
			name:   "browsers only",
			filter: types.CVEChartFilter{CVSSMin: critMin, CVSSMax: critMax, Categories: []string{api.CVECategoryBrowsers}},
			want:   []string{"CVE-A"},
		},
		{
			name:   "OS category includes kernel software and OS vulns",
			filter: types.CVEChartFilter{CVSSMin: critMin, CVSSMax: critMax, Categories: []string{api.CVECategoryOS}},
			want:   []string{"CVE-D", "CVE-E"},
		},
		{
			name:   "known-exploit only",
			filter: types.CVEChartFilter{CVSSMin: critMin, CVSSMax: critMax, KnownExploit: true},
			want:   []string{"CVE-A", "CVE-E"},
		},
		{
			name:   "EPSS band 0.85-1.0",
			filter: types.CVEChartFilter{CVSSMin: critMin, CVSSMax: critMax, EPSSMin: new(0.85), EPSSMax: new(1.0)},
			want:   []string{"CVE-E"},
		},
		{
			name:   "exclude a collected CVE",
			filter: types.CVEChartFilter{CVSSMin: critMin, CVSSMax: critMax, ExcludeCVEs: []string{"CVE-A"}},
			want:   []string{"CVE-C", "CVE-D", "CVE-E"},
		},
		{
			name:   "exclude an uncollected CVE is a no-op",
			filter: types.CVEChartFilter{CVSSMin: critMin, CVSSMax: critMax, ExcludeCVEs: []string{"CVE-NOT-COLLECTED"}},
			want:   []string{"CVE-A", "CVE-C", "CVE-D", "CVE-E"},
		},
		{
			name:   "combined: browsers AND known-exploit",
			filter: types.CVEChartFilter{CVSSMin: critMin, CVSSMax: critMax, Categories: []string{api.CVECategoryBrowsers}, KnownExploit: true},
			want:   []string{"CVE-A"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ds.ResolveCVEChartEntities(ctx, tc.filter)
			require.NoError(t, err)
			require.NotNil(t, got, "resolver must return a non-nil slice")
			assert.ElementsMatch(t, tc.want, got)
			assert.NotContains(t, got, "CVE-F", "untracked software CVE must never resolve")
		})
	}
}
