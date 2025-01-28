package nvd

import (
	"errors"
	"testing"

	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/wfn"
	"github.com/stretchr/testify/require"
)

func TestCPEProcessingRule(t *testing.T) {
	buildRule := func() CPEMatchingRule {
		return CPEMatchingRule{
			CPESpecs: []CPEMatchingRuleSpec{
				{
					Vendor:           "microsoft",
					Product:          "word",
					TargetSW:         "windows",
					SemVerConstraint: "1.2.3",
				},
			},

			CVEs: map[string]struct{}{"CVE-123": {}},
		}
	}

	buildCPEMeta := func() *wfn.Attributes {
		cpeMeta, err := wfn.Parse("cpe:2.3:a:microsoft:word:1.2.3:*:*:*:*:windows:*:*")
		require.NoError(t, err)
		return cpeMeta
	}

	t.Run("getCPEAttrs", func(t *testing.T) {
		rule := buildRule()
		result := rule.CPESpecs[0].getCPEMeta()
		require.NotNil(t, result)

		expected := wfn.Attributes{
			Vendor:   rule.CPESpecs[0].Vendor,
			Product:  rule.CPESpecs[0].Product,
			TargetSW: rule.CPESpecs[0].TargetSW,
		}
		require.True(t, expected.MatchWithoutVersion(result))
	})

	t.Run("Matches", func(t *testing.T) {
		t.Run("is a match", func(t *testing.T) {
			rule := buildRule()
			cpeMeta := buildCPEMeta()
			require.True(t, rule.CPEMatches(cpeMeta))
		})

		t.Run("CPEMeta info is null", func(t *testing.T) {
			rule := buildRule()
			require.False(t, rule.CPEMatches(nil))
		})

		t.Run("CPEs don't match", func(t *testing.T) {
			rule := buildRule()
			rule.CPESpecs[0].Vendor = "AMD"
			cpeMeta := buildCPEMeta()
			require.False(t, rule.CPEMatches(cpeMeta))
		})

		t.Run("SemVer is a range constraint", func(t *testing.T) {
			rule := buildRule()
			rule.CPESpecs[0].SemVerConstraint = "1.0.0 - 2.0.0"
			cpeMeta := buildCPEMeta()
			require.True(t, rule.CPEMatches(cpeMeta))
		})
	})

	t.Run("Validate", func(t *testing.T) {
		testCases := []struct {
			rule CPEMatchingRule
			err  error
		}{
			{
				rule: CPEMatchingRule{
					CPESpecs: []CPEMatchingRuleSpec{
						{
							Vendor:           "",
							Product:          "word",
							TargetSW:         "windows",
							SemVerConstraint: "1.2.3",
						},
					},

					CVEs: map[string]struct{}{"CVE-123": {}},
				}, err: errors.New("Vendor can't be empty"),
			},
			{
				rule: CPEMatchingRule{
					CPESpecs: []CPEMatchingRuleSpec{
						{
							Vendor:           "*",
							Product:          "word",
							TargetSW:         "windows",
							SemVerConstraint: "1.2.3",
						},
					},

					CVEs: map[string]struct{}{"CVE-123": {}},
				}, err: errors.New("Vendor can't be 'ANY'"),
			},
			{
				rule: CPEMatchingRule{
					CPESpecs: []CPEMatchingRuleSpec{
						{
							Vendor:           "-",
							Product:          "word",
							TargetSW:         "windows",
							SemVerConstraint: "1.2.3",
						},
					},
					CVEs: map[string]struct{}{"CVE-123": {}},
				}, err: errors.New("Vendor can't be 'NA'"),
			},
			{
				rule: CPEMatchingRule{
					CPESpecs: []CPEMatchingRuleSpec{
						{
							Vendor:           "microsoft",
							Product:          "",
							TargetSW:         "windows",
							SemVerConstraint: "1.2.3",
						},
					},

					CVEs: map[string]struct{}{"CVE-123": {}},
				}, err: errors.New("Product can't be empty"),
			},
			{
				rule: CPEMatchingRule{
					CPESpecs: []CPEMatchingRuleSpec{
						{
							Vendor:           "microsoft",
							Product:          "*",
							TargetSW:         "windows",
							SemVerConstraint: "1.2.3",
						},
					},

					CVEs: map[string]struct{}{"CVE-123": {}},
				}, err: errors.New("Product can't be 'ANY'"),
			},
			{
				rule: CPEMatchingRule{
					CPESpecs: []CPEMatchingRuleSpec{
						{
							Vendor:           "microsoft",
							Product:          "-",
							TargetSW:         "windows",
							SemVerConstraint: "1.2.3",
						},
					},
					CVEs: map[string]struct{}{"CVE-123": {}},
				}, err: errors.New("Product can't be 'NA'"),
			},
			{
				rule: CPEMatchingRule{
					CPESpecs: []CPEMatchingRuleSpec{
						{
							Vendor:           "microsoft",
							Product:          "word",
							TargetSW:         "",
							SemVerConstraint: "1.2.3",
						},
					},
					CVEs: map[string]struct{}{"CVE-123": {}},
				}, err: errors.New("TargetSW can't be empty"),
			},
			{
				rule: CPEMatchingRule{
					CPESpecs: []CPEMatchingRuleSpec{
						{
							Vendor:           "microsoft",
							Product:          "word",
							TargetSW:         "*",
							SemVerConstraint: "1.2.3",
						},
					},
					CVEs: map[string]struct{}{"CVE-123": {}},
				}, err: errors.New("TargetSW can't be 'ANY'"),
			},
			{
				rule: CPEMatchingRule{
					CPESpecs: []CPEMatchingRuleSpec{
						{
							Vendor:           "microsoft",
							Product:          "word",
							TargetSW:         "-",
							SemVerConstraint: "1.2.3",
						},
					},
					CVEs: map[string]struct{}{"CVE-123": {}},
				}, err: errors.New("TargetSW can't be 'NA'"),
			},
			{
				rule: CPEMatchingRule{
					CPESpecs: []CPEMatchingRuleSpec{
						{
							Vendor:           "microsoft",
							Product:          "word",
							TargetSW:         "windows",
							SemVerConstraint: ".as.-as",
						},
					},
					CVEs: map[string]struct{}{"CVE-123": {}},
				}, err: errors.New("improper constraint: .as.-as"),
			},
			{
				rule: CPEMatchingRule{
					CPESpecs: []CPEMatchingRuleSpec{
						{
							Vendor:           "microsoft",
							Product:          "word",
							TargetSW:         "windows",
							SemVerConstraint: "1.2.3",
						},
					},
				}, err: errors.New("At least one CVE is required"),
			},
			{
				rule: CPEMatchingRule{
					CPESpecs: []CPEMatchingRuleSpec{
						{
							Vendor:           "microsoft",
							Product:          "word",
							TargetSW:         "windows",
							SemVerConstraint: "1.2.3",
						},
					},
					CVEs: map[string]struct{}{"": {}, "  ": {}, "CVE-123": {}},
				}, err: errors.New("CVE can't be empty"),
			},
		}

		for _, tc := range testCases {
			result := tc.rule.Validate()
			require.Equal(t, tc.err, result)
		}
	})
}

func TestGetKnownNVDBugRules(t *testing.T) {
	cpeMatchingRules, err := GetKnownNVDBugRules()
	require.NoError(t, err)

	cpeMeta, err := wfn.Parse("cpe:2.3:a:microsoft:teams:*:*:*:*:*:*:*:*")
	require.NoError(t, err)

	// Test that CVE-2020-10146 never matches (i.e. is ignored).
	rule, ok := cpeMatchingRules.FindMatch("CVE-2020-10146")
	require.True(t, ok)
	ok = rule.CPEMatches(cpeMeta)
	require.False(t, ok)

	// Test that CVE-2013-0340 never matches (i.e. is ignored).
	rule, ok = cpeMatchingRules.FindMatch("CVE-2013-0340")
	require.True(t, ok)
	ok = rule.CPEMatches(cpeMeta)
	require.False(t, ok)
}
