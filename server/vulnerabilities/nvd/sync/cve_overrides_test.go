package nvdsync

import (
	"path/filepath"
	"testing"

	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/cvefeed/nvd/schema"
	"github.com/stretchr/testify/require"
)

func TestCVEOverridesFileIsValid(t *testing.T) {
	t.Parallel()

	overrides, err := parseCVEOverrides(cveOverridesJSON)
	require.NoError(t, err)
	require.NotEmpty(t, overrides)
}

func TestParseCVEOverridesRejectsInvalidEntries(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"unknown key":                `[{"cve": "CVE-2026-1000", "reason": "r", "match": {"criteria_contains": ":a:b:"}, "set": {"versionEndExcluding": "1.0"}, "typo": true}]`,
		"unknown match key":          `[{"cve": "CVE-2026-1000", "reason": "r", "match": {"criteria_contains": ":a:b:", "versionEnd": "1.0"}, "set": {"versionEndExcluding": "1.0"}}]`,
		"malformed CVE ID":           `[{"cve": "2026-1000", "reason": "r", "match": {"criteria_contains": ":a:b:"}, "set": {"versionEndExcluding": "1.0"}}]`,
		"missing reason":             `[{"cve": "CVE-2026-1000", "match": {"criteria_contains": ":a:b:"}, "set": {"versionEndExcluding": "1.0"}}]`,
		"missing criteria_contains":  `[{"cve": "CVE-2026-1000", "reason": "r", "match": {"versionEndIncluding": "1.0"}, "set": {"versionEndExcluding": "1.0"}}]`,
		"nothing to set":             `[{"cve": "CVE-2026-1000", "reason": "r", "match": {"criteria_contains": ":a:b:"}, "set": {}}]`,
		"empty version":              `[{"cve": "CVE-2026-1000", "reason": "r", "match": {"criteria_contains": ":a:b:"}, "set": {"versionEndExcluding": ""}}]`,
		"same CVE and criteria twice": `[{"cve": "CVE-2026-1000", "reason": "r", "match": {"criteria_contains": ":a:b:"}, "set": {"versionEndExcluding": "1.0"}},
		                                 {"cve": "CVE-2026-1000", "reason": "r", "match": {"criteria_contains": ":a:b:"}, "set": {"versionEndExcluding": "2.0"}}]`,
	}
	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, err := parseCVEOverrides([]byte(raw))
			require.Error(t, err)
		})
	}
}

func TestApplyCVEOverrides(t *testing.T) {
	t.Parallel()

	overrides, err := parseCVEOverrides(cveOverridesJSON)
	require.NoError(t, err)

	// makeItem builds a legacy feed item shaped like convertAPI20CVEToLegacy's output for a
	// configuration with an operator: the CPE match sits in a child node.
	makeItem := func(cveID string, match schema.NVDCVEFeedJSON10DefCPEMatch) *schema.NVDCVEFeedJSON10DefCVEItem {
		match.Vulnerable = true
		return &schema.NVDCVEFeedJSON10DefCVEItem{
			CVE: &schema.CVEJSON40{CVEDataMeta: &schema.CVEJSON40CVEDataMeta{ID: cveID}},
			Configurations: &schema.NVDCVEFeedJSON10DefConfigurations{
				Nodes: []*schema.NVDCVEFeedJSON10DefNode{{
					Operator: "AND",
					Children: []*schema.NVDCVEFeedJSON10DefNode{{
						Operator: "OR",
						CPEMatch: []*schema.NVDCVEFeedJSON10DefCPEMatch{&match},
					}},
				}},
			},
		}
	}
	apply := func(item *schema.NVDCVEFeedJSON10DefCVEItem) schema.NVDCVEFeedJSON10DefCPEMatch {
		applyCVEOverrides([]*schema.NVDCVEFeedJSON10DefCVEItem{item}, overrides)
		return *item.Configurations.Nodes[0].Children[0].CPEMatch[0]
	}

	const (
		litellmCPE = "cpe:2.3:a:litellm:litellm:*:*:*:*:*:*:*:*"
		ollamaCPE  = "cpe:2.3:a:ollama:ollama:*:*:*:*:*:*:*:*"
		otherCPE   = "cpe:2.3:a:acme:widget:*:*:*:*:*:*:*:*"
	)

	t.Run("CVE-2026-40217 date bound is replaced with the affected version range", func(t *testing.T) {
		got := apply(makeItem("CVE-2026-40217", schema.NVDCVEFeedJSON10DefCPEMatch{Cpe23Uri: litellmCPE, VersionEndIncluding: "2026-04-08"}))
		require.Equal(t, "1.81.8", got.VersionStartIncluding)
		require.Empty(t, got.VersionEndIncluding)
		require.Equal(t, "1.83.10", got.VersionEndExcluding)
	})

	t.Run("CVE-2026-40217 override leaves a corrected NVD range alone", func(t *testing.T) {
		got := apply(makeItem("CVE-2026-40217", schema.NVDCVEFeedJSON10DefCPEMatch{Cpe23Uri: litellmCPE, VersionEndExcluding: "1.83.10"}))
		require.Empty(t, got.VersionStartIncluding)
		require.Empty(t, got.VersionEndIncluding)
		require.Equal(t, "1.83.10", got.VersionEndExcluding)
	})

	t.Run("CVE-2026-40217 override does not apply to other products", func(t *testing.T) {
		got := apply(makeItem("CVE-2026-40217", schema.NVDCVEFeedJSON10DefCPEMatch{Cpe23Uri: otherCPE, VersionEndIncluding: "2026-04-08"}))
		require.Empty(t, got.VersionStartIncluding)
		require.Equal(t, "2026-04-08", got.VersionEndIncluding)
		require.Empty(t, got.VersionEndExcluding)
	})

	t.Run("CVE-2025-63389 gets a resolved version when NVD provides only versionEndIncluding", func(t *testing.T) {
		got := apply(makeItem("CVE-2025-63389", schema.NVDCVEFeedJSON10DefCPEMatch{Cpe23Uri: ollamaCPE, VersionEndIncluding: "0.12.3"}))
		require.Equal(t, "0.12.3", got.VersionEndIncluding)
		require.Equal(t, "0.12.4", got.VersionEndExcluding)
	})

	t.Run("CVE-2025-63389 does not clobber an existing versionEndExcluding", func(t *testing.T) {
		got := apply(makeItem("CVE-2025-63389", schema.NVDCVEFeedJSON10DefCPEMatch{Cpe23Uri: ollamaCPE, VersionEndIncluding: "0.12.3", VersionEndExcluding: "0.12.9"}))
		require.Equal(t, "0.12.9", got.VersionEndExcluding)
	})

	t.Run("CVE-2025-63389 override does not apply to other products", func(t *testing.T) {
		got := apply(makeItem("CVE-2025-63389", schema.NVDCVEFeedJSON10DefCPEMatch{Cpe23Uri: otherCPE, VersionEndIncluding: "0.12.3"}))
		require.Empty(t, got.VersionEndExcluding)
	})

	t.Run("CVE-2024-6286 LTSR resolved version is corrected", func(t *testing.T) {
		got := apply(makeItem("CVE-2024-6286", schema.NVDCVEFeedJSON10DefCPEMatch{Cpe23Uri: "cpe:2.3:a:citrix:workspace:*:*:*:*:ltsr:windows:*:*", VersionEndExcluding: "2203.1"}))
		require.Equal(t, "2402", got.VersionEndExcluding)
	})

	t.Run("CVE-2024-6286 current release match is left alone", func(t *testing.T) {
		got := apply(makeItem("CVE-2024-6286", schema.NVDCVEFeedJSON10DefCPEMatch{Cpe23Uri: "cpe:2.3:a:citrix:workspace:*:*:*:*:current:windows:*:*", VersionEndExcluding: "2403.1"}))
		require.Equal(t, "2403.1", got.VersionEndExcluding)
	})

	t.Run("unrelated CVE is left unchanged", func(t *testing.T) {
		got := apply(makeItem("CVE-2026-00001", schema.NVDCVEFeedJSON10DefCPEMatch{Cpe23Uri: litellmCPE, VersionEndIncluding: "2026-04-08"}))
		require.Equal(t, "2026-04-08", got.VersionEndIncluding)
		require.Empty(t, got.VersionEndExcluding)
	})
}

func TestApplyOverridesToStoredFeeds(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	feed := &schema.NVDCVEFeedJSON10{
		CVEItems: []*schema.NVDCVEFeedJSON10DefCVEItem{{
			CVE: &schema.CVEJSON40{CVEDataMeta: &schema.CVEJSON40CVEDataMeta{ID: "CVE-2026-40217"}},
			Configurations: &schema.NVDCVEFeedJSON10DefConfigurations{
				Nodes: []*schema.NVDCVEFeedJSON10DefNode{{
					Operator: "OR",
					CPEMatch: []*schema.NVDCVEFeedJSON10DefCPEMatch{{
						Cpe23Uri:            "cpe:2.3:a:litellm:litellm:*:*:*:*:*:*:*:*",
						VersionEndIncluding: "2026-04-08",
						Vulnerable:          true,
					}},
				}},
			},
		}},
	}
	require.NoError(t, storeCVEsInLegacyFormat(dir, 2026, feed))

	s, err := NewCVE(dir)
	require.NoError(t, err)
	// The overrides file also has entries for years with no stored feed here, which must be skipped.
	require.NoError(t, s.ApplyOverrides(t.Context()))

	stored, err := readCVEsLegacyFormat(dir, 2026)
	require.NoError(t, err)
	require.Len(t, stored.CVEItems, 1)
	match := stored.CVEItems[0].Configurations.Nodes[0].CPEMatch[0]
	require.Equal(t, "1.81.8", match.VersionStartIncluding)
	require.Empty(t, match.VersionEndIncluding)
	require.Equal(t, "1.83.10", match.VersionEndExcluding)

	require.NoFileExists(t, filepath.Join(dir, "nvdcve-1.1-2025.json"), "a year with no stored feed must not be created")
}
