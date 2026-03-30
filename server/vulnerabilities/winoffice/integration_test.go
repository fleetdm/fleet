package winoffice_test

import (
	"context"
	"slices"
	"strconv"
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

	bulletin, err := winoffice.FetchBulletin(context.Background(), client)
	require.NoError(t, err)
	require.NotEmpty(t, bulletin.Versions)
	require.NotEmpty(t, bulletin.BuildPrefixes)

	// Get the newest build prefix (compare numerically, not lexically)
	var testPrefix string
	var maxPrefixNum int
	for prefix := range bulletin.BuildPrefixes {
		num, err := strconv.Atoi(prefix)
		if err != nil {
			continue
		}
		if num > maxPrefixNum {
			maxPrefixNum = num
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

	// Test versions without data return empty (not false positives)
	t.Run("version not in bulletin returns empty", func(t *testing.T) {
		// Use a made-up build prefix that won't be in the bulletin
		vulns := winoffice.CheckVersion("16.0.12345.20000", bulletin)
		assert.Empty(t, vulns, "unknown version should return empty, not false positives")
	})

	// These tests verify specific CVEs and resolved versions.
	// Counts use GreaterOrEqual since new security updates increase CVE counts over time.

	t.Run("known suffix version includes March CVEs", func(t *testing.T) {
		vulns := winoffice.CheckVersion("16.0.19628.20204", bulletin)
		assert.GreaterOrEqual(t, len(vulns), 7, "expected at least 7 CVEs for 16.0.19628.20204")
		// Version 2601 (19628) is deprecated, so fix points to newer version 2602 (19725)
		assertCVEWithResolution(t, vulns, "CVE-2026-26107", "16.0.19725.20170")
	})

	t.Run("unknown suffix version includes March CVEs", func(t *testing.T) {
		vulns := winoffice.CheckVersion("16.0.19628.20205", bulletin)
		assert.GreaterOrEqual(t, len(vulns), 7, "expected at least 7 CVEs for 16.0.19628.20205")
		// Version 2601 (19628) is deprecated, so fix points to newer version 2602 (19725)
		assertCVEWithResolution(t, vulns, "CVE-2026-26107", "16.0.19725.20170")
		assertCVENotPresent(t, vulns, "CVE-2026-21509") // February CVE should not appear
	})

	t.Run("Monthly channel version includes March CVEs with a lower resolved version", func(t *testing.T) {
		vulns := winoffice.CheckVersion("16.0.19530.20226", bulletin)
		assert.GreaterOrEqual(t, len(vulns), 7, "expected at least 7 CVEs for 16.0.19530.20226")
		assertCVEWithResolution(t, vulns, "CVE-2026-26107", "16.0.19530.20260")
		assertCVENotPresent(t, vulns, "CVE-2026-21509") // February CVE should not appear
	})

	t.Run("deprecated monthly channel version includes March CVEs with a higher resolved version", func(t *testing.T) {
		vulns := winoffice.CheckVersion("16.0.19328.20306", bulletin)
		assert.GreaterOrEqual(t, len(vulns), 7, "expected at least 7 CVEs for 16.0.19328.20306")
		// Version 2510 (19328) is deprecated, so fix points to newer version 2511 (19426)
		assertCVEWithResolution(t, vulns, "CVE-2026-26107", "16.0.19426.20314")
	})

	t.Run("January release includes March and Feb CVEs", func(t *testing.T) {
		vulns := winoffice.CheckVersion("16.0.19530.20144", bulletin)
		assert.GreaterOrEqual(t, len(vulns), 14, "expected at least 14 CVEs for 16.0.19530.20144")
		assertCVEWithResolution(t, vulns, "CVE-2026-26107", "16.0.19530.20260") // March 2026
		assertCVEWithResolution(t, vulns, "CVE-2026-21509", "16.0.19530.20226") // February 2026
	})

	// False positive tests - versions at their fixed build should report no vulnerabilities

	t.Run("no false positives for LTSC at fixed build", func(t *testing.T) {
		// LTSC 2024 version 2408 has had multiple build prefixes (17928, 17932).
		// A host at 17932.20700 should NOT have CVEs with fixes at older prefix 17928.
		vulns := winoffice.CheckVersion("16.0.17932.20700", bulletin)
		// These CVEs have fixes at 17928.XXXXX - should not appear for host at 17932
		oldPrefixCVEs := []string{"CVE-2024-38016", "CVE-2024-38226", "CVE-2024-43463"}
		for _, cve := range oldPrefixCVEs {
			assertCVENotPresent(t, vulns, cve)
		}
	})

	t.Run("no false positives for version at latest build", func(t *testing.T) {
		// A version with the highest possible build suffix should have no vulnerabilities
		vulns := winoffice.CheckVersion("16.0."+testPrefix+".99999", bulletin)
		assert.Empty(t, vulns, "version at latest build should have no vulnerabilities")
	})

	t.Run("no false positives for unknown build prefix", func(t *testing.T) {
		// A build prefix not in the bulletin should return empty, not upgrade paths
		vulns := winoffice.CheckVersion("16.0.55555.20000", bulletin)
		assert.Empty(t, vulns, "unknown build prefix should return empty, not false positives")
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

func assertCVENotPresent(t *testing.T, vulns []fleet.SoftwareVulnerability, cve string) {
	t.Helper()
	idx := slices.IndexFunc(vulns, func(v fleet.SoftwareVulnerability) bool {
		return v.CVE == cve
	})
	assert.Equal(t, -1, idx, "CVE %s should NOT be in the list (false positive)", cve)
}
