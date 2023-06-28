package nvd

import (
	"errors"
	"testing"

	"github.com/facebookincubator/nvdtools/wfn"
	"github.com/stretchr/testify/require"
)

func TestCPEProcessingRules(t *testing.T) {
	t.Run("FindMatch", func(t *testing.T) {
		t.Run("no rules", func(t *testing.T) {
			var rules CPEProcessingRules
			rule, found := rules.FindMatch(nil, "CVE-123")
			require.Nil(t, rule)
			require.False(t, found)
		})
	})
}

func TestCPEProcessingRule(t *testing.T) {
	buildRule := func() CPEProcessingRule {
		return CPEProcessingRule{
			Vendor:           "microsoft",
			Product:          "word",
			TargetSW:         "windows",
			SemVerConstraint: "1.2.3",
			CVEs:             []string{"CVE-123"},
		}
	}

	buildCPEMeta := func() *wfn.Attributes {
		cpeMeta, err := wfn.Parse("cpe:2.3:a:microsoft:word:1.2.3:*:*:*:*:windows:*:*")
		require.NoError(t, err)
		return cpeMeta
	}

	t.Run("getCPEAttrs", func(t *testing.T) {
		rule := buildRule()
		result := rule.getCPEMeta()
		require.NotNil(t, result)

		expected := wfn.Attributes{
			Vendor:   rule.Vendor,
			Product:  rule.Product,
			TargetSW: rule.TargetSW,
		}
		require.True(t, expected.MatchWithoutVersion(result))
	})

	t.Run("Matches", func(t *testing.T) {
		t.Run("is a match", func(t *testing.T) {
			rule := buildRule()
			cpeMeta := buildCPEMeta()
			require.True(t, rule.Matches(cpeMeta, rule.CVEs[0]))
		})

		t.Run("CVE don't match", func(t *testing.T) {
			rule := buildRule()
			cpeMeta := buildCPEMeta()
			require.False(t, rule.Matches(cpeMeta, "CVE-452"))
		})

		t.Run("CPEMeta info is null", func(t *testing.T) {
			rule := buildRule()
			require.False(t, rule.Matches(nil, rule.CVEs[0]))
		})

		t.Run("CPEs don't match", func(t *testing.T) {
			rule := buildRule()
			rule.Vendor = "AMD"
			cpeMeta := buildCPEMeta()
			require.False(t, rule.Matches(cpeMeta, rule.CVEs[0]))
		})

		t.Run("SemVer is a range constraint", func(t *testing.T) {
			rule := buildRule()
			rule.SemVerConstraint = "1.0.0 - 2.0.0"
			cpeMeta := buildCPEMeta()
			require.True(t, rule.Matches(cpeMeta, rule.CVEs[0]))
		})
	})

	t.Run("Validate", func(t *testing.T) {
		testCases := []struct {
			rule CPEProcessingRule
			err  error
		}{
			{
				rule: CPEProcessingRule{
					Vendor:           "",
					Product:          "word",
					TargetSW:         "windows",
					SemVerConstraint: "1.2.3",
					CVEs:             []string{"CVE-123"},
				}, err: errors.New("Vendor can't be empty"),
			},
			{
				rule: CPEProcessingRule{
					Vendor:           "*",
					Product:          "word",
					TargetSW:         "windows",
					SemVerConstraint: "1.2.3",
					CVEs:             []string{"CVE-123"},
				}, err: errors.New("Vendor can't be 'ANY'"),
			},
			{
				rule: CPEProcessingRule{
					Vendor:           "-",
					Product:          "word",
					TargetSW:         "windows",
					SemVerConstraint: "1.2.3",
					CVEs:             []string{"CVE-123"},
				}, err: errors.New("Vendor can't be 'NA'"),
			},
			{
				rule: CPEProcessingRule{
					Vendor:           "microsoft",
					Product:          "",
					TargetSW:         "windows",
					SemVerConstraint: "1.2.3",
					CVEs:             []string{"CVE-123"},
				}, err: errors.New("Product can't be empty"),
			},
			{
				rule: CPEProcessingRule{
					Vendor:           "microsoft",
					Product:          "*",
					TargetSW:         "windows",
					SemVerConstraint: "1.2.3",
					CVEs:             []string{"CVE-123"},
				}, err: errors.New("Product can't be 'ANY'"),
			},
			{
				rule: CPEProcessingRule{
					Vendor:           "microsoft",
					Product:          "-",
					TargetSW:         "windows",
					SemVerConstraint: "1.2.3",
					CVEs:             []string{"CVE-123"},
				}, err: errors.New("Product can't be 'NA'"),
			},
			{
				rule: CPEProcessingRule{
					Vendor:           "microsoft",
					Product:          "word",
					TargetSW:         "",
					SemVerConstraint: "1.2.3",
					CVEs:             []string{"CVE-123"},
				}, err: errors.New("TargetSW can't be empty"),
			},
			{
				rule: CPEProcessingRule{
					Vendor:           "microsoft",
					Product:          "word",
					TargetSW:         "*",
					SemVerConstraint: "1.2.3",
					CVEs:             []string{"CVE-123"},
				}, err: errors.New("TargetSW can't be 'ANY'"),
			},
			{
				rule: CPEProcessingRule{
					Vendor:           "microsoft",
					Product:          "word",
					TargetSW:         "-",
					SemVerConstraint: "1.2.3",
					CVEs:             []string{"CVE-123"},
				}, err: errors.New("TargetSW can't be 'NA'"),
			},
			{
				rule: CPEProcessingRule{
					Vendor:           "microsoft",
					Product:          "word",
					TargetSW:         "windows",
					SemVerConstraint: ".as.-as",
					CVEs:             []string{"CVE-123"},
				}, err: errors.New("improper constraint: .as.-as"),
			},
			{
				rule: CPEProcessingRule{
					Vendor:           "microsoft",
					Product:          "word",
					TargetSW:         "windows",
					SemVerConstraint: "1.2.3",
				}, err: errors.New("At least one CVE is required"),
			},
			{
				rule: CPEProcessingRule{
					Vendor:           "microsoft",
					Product:          "word",
					TargetSW:         "windows",
					SemVerConstraint: "1.2.3",
					CVEs:             []string{"", "CVE-123"},
				}, err: errors.New("CVE can't be empty"),
			},

			{
				rule: CPEProcessingRule{
					Vendor:           "microsoft",
					Product:          "word",
					TargetSW:         "windows",
					SemVerConstraint: "1.2.3",
					CVEs:             []string{"CVE-123", "CVE-123"},
				}, err: errors.New("duplicated CVE 'CVE-123'"),
			},
		}

		for _, tc := range testCases {
			result := tc.rule.Validate()
			require.Equal(t, tc.err, result)
		}
	})
}
