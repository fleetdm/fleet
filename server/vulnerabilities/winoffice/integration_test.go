package winoffice_test

import (
	"slices"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/pkg/nettest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/winoffice"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegrationCheckVersion(t *testing.T) {
	nettest.Run(t)

	client := fleethttp.NewClient(fleethttp.WithTimeout(60 * time.Second))

	bulletin, err := winoffice.FetchBulletin(client)
	require.NoError(t, err)
	require.NotEmpty(t, bulletin.Versions)
	require.NotEmpty(t, bulletin.BuildPrefixes)

	// Get the newest build prefix
	var testPrefix string
	for prefix := range bulletin.BuildPrefixes {
		if prefix > testPrefix {
			testPrefix = prefix
		}
	}

	t.Logf("Bulletin: %d versions", len(bulletin.Versions))

	t.Run("old version is vulnerable", func(t *testing.T) {
		vulns := winoffice.CheckVersion("16.0."+testPrefix+".00001", bulletin)
		assert.NotEmpty(t, vulns, "old build should have vulnerabilities")
		t.Logf("Version 16.0.%s.00001: %d CVEs", testPrefix, len(vulns))
	})

	t.Run("latest version is not vulnerable", func(t *testing.T) {
		vulns := winoffice.CheckVersion("16.0."+testPrefix+".99999", bulletin)
		assert.Empty(t, vulns, "latest build should have no vulnerabilities")
	})

	t.Run("unknown version returns no vulnerabilities", func(t *testing.T) {
		vulns := winoffice.CheckVersion("16.0.99999.99999", bulletin)
		assert.Empty(t, vulns, "unknown version should return empty")
	})

	t.Run("invalid version returns no vulnerabilities", func(t *testing.T) {
		vulns := winoffice.CheckVersion("invalid", bulletin)
		assert.Empty(t, vulns, "invalid version should return empty")
	})

	// These tests use GreaterOrEqual for counts since new security updates will increase CVE counts over time.
	// The specific CVEs and resolved versions checked will always be present (historical data isn't removed).

	t.Run("known suffix version includes March CVEs", func(t *testing.T) {
		vulns := winoffice.CheckVersion("16.0.19628.20204", bulletin)
		assert.GreaterOrEqual(t, len(vulns), 7, "expected at least 7 CVEs for 16.0.19628.20204")
		// Version 2601 (19628) is dropped, so fix points to newer version 2602 (19725)
		assertCVEWithResolution(t, vulns, "CVE-2026-26107", "16.0.19725.20170")
	})

	t.Run("unknown suffix version includes March CVEs", func(t *testing.T) {
		vulns := winoffice.CheckVersion("16.0.19628.20205", bulletin)
		assert.GreaterOrEqual(t, len(vulns), 7, "expected at least 7 CVEs for 16.0.19628.20205")
		// Version 2601 (19628) is dropped, so fix points to newer version 2602 (19725)
		assertCVEWithResolution(t, vulns, "CVE-2026-26107", "16.0.19725.20170")
	})

	t.Run("Monthly channel version includes March CVEs with a lower resolved version", func(t *testing.T) {
		vulns := winoffice.CheckVersion("16.0.19530.20226", bulletin)
		assert.GreaterOrEqual(t, len(vulns), 7, "expected at least 7 CVEs for 16.0.19530.20226")
		assertCVEWithResolution(t, vulns, "CVE-2026-26107", "16.0.19530.20260")
	})

	t.Run("deprecated monthly channel version includes March CVEs with a higher resolved version", func(t *testing.T) {
		vulns := winoffice.CheckVersion("16.0.19328.20306", bulletin)
		assert.GreaterOrEqual(t, len(vulns), 7, "expected at least 7 CVEs for 16.0.19328.20306")
		// Version 2510 (19328) is dropped, so fix points to newer version 2511 (19426)
		assertCVEWithResolution(t, vulns, "CVE-2026-26107", "16.0.19426.20314")
	})

	t.Run("January release includes March and Feb CVEs", func(t *testing.T) {
		vulns := winoffice.CheckVersion("16.0.19530.20144", bulletin)
		assert.GreaterOrEqual(t, len(vulns), 14, "expected at least 14 CVEs for 16.0.19530.20144")
		assertCVEWithResolution(t, vulns, "CVE-2026-26107", "16.0.19530.20260") // March 2026
		assertCVEWithResolution(t, vulns, "CVE-2026-21509", "16.0.19530.20226") // February 2026
	})
}

func assertCVEWithResolution(t *testing.T, vulns []fleet.SoftwareVulnerability, cve, resolvedIn string) {
	t.Helper()
	idx := slices.IndexFunc(vulns, func(v fleet.SoftwareVulnerability) bool {
		return v.CVE == cve
	})
	require.NotEqual(t, -1, idx, "CVE %s should be in the list", cve)
	require.NotNil(t, vulns[idx].ResolvedInVersion, "CVE %s should have a resolved version", cve)
	assert.Equal(t, resolvedIn, *vulns[idx].ResolvedInVersion, "CVE %s resolved version mismatch", cve)
}
