package customcve

import (
	"context"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/go-kit/log"
	"github.com/stretchr/testify/require"
)

func TestMatchVersion(t *testing.T) {
	ds := new(mock.Store)

	rule := CVEMatchingRule{
		NameLikeMatch:     "Microsoft 365",
		SourceMatch:       "programs",
		ResolvedInVersion: "16.0.17628.20144",
		CVEs:              []string{"CVE-2024-001", "CVE-2024-002"},
	}

	sw := []fleet.Software{
		{
			ID:      1,
			Version: "16.0.17531.20152", // in range
		},
		{
			ID:      2,
			Version: "16.0.17425.20176", // in range
		},
		{
			ID:      3,
			Version: "16.0.17628.20144", // over range equals
		},
		{
			ID:      4,
			Version: "16.0.17628.20145", // over range
		},
	}

	expected := []fleet.SoftwareVulnerability{
		{
			SoftwareID:        1,
			CVE:               "CVE-2024-001",
			ResolvedInVersion: ptr.String("16.0.17628.20144"),
		},
		{
			SoftwareID:        1,
			CVE:               "CVE-2024-002",
			ResolvedInVersion: ptr.String("16.0.17628.20144"),
		},
		{
			SoftwareID:        2,
			CVE:               "CVE-2024-001",
			ResolvedInVersion: ptr.String("16.0.17628.20144"),
		},
		{
			SoftwareID:        2,
			CVE:               "CVE-2024-002",
			ResolvedInVersion: ptr.String("16.0.17628.20144"),
		},
	}

	ds.ListSoftwareForVulnDetectionFunc = func(ctx context.Context, filter fleet.VulnSoftwareFilter) ([]fleet.Software, error) {
		return sw, nil
	}

	actual, err := rule.match(context.Background(), ds)
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}

func TestMatchFilters(t *testing.T) {
	ds := new(mock.Store)

	testCases := []struct {
		name           string
		rule           CVEMatchingRule
		expectedFilter fleet.VulnSoftwareFilter
	}{
		{
			name: "Match all",
			rule: CVEMatchingRule{
				NameLikeMatch:     "Microsoft 365",
				SourceMatch:       "programs",
				ResolvedInVersion: "16.0.17628.20144",
				CVEs:              []string{"CVE-2024-001", "CVE-2024-002"},
			},
			expectedFilter: fleet.VulnSoftwareFilter{
				Name:   "Microsoft 365",
				Source: "programs",
			},
		},
		{
			name: "Match only name",
			rule: CVEMatchingRule{
				NameLikeMatch:     "Microsoft 365",
				ResolvedInVersion: "16.0.17628.20144",
				CVEs:              []string{"CVE-2024-001", "CVE-2024-002"},
			},
			expectedFilter: fleet.VulnSoftwareFilter{
				Name: "Microsoft 365",
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			ds.ListSoftwareForVulnDetectionFunc = func(ctx context.Context, filter fleet.VulnSoftwareFilter) ([]fleet.Software, error) {
				require.Equal(t, tt.expectedFilter, filter)
				return nil, nil
			}

			_, err := tt.rule.match(context.Background(), ds)
			require.NoError(t, err)
		})
	}
}

func TestCVEMatchingRuleValidation(t *testing.T) {
	testCases := []struct {
		name string
		rule CVEMatchingRule
		err  error
	}{
		{
			name: "Valid rule",
			rule: CVEMatchingRule{
				NameLikeMatch:     "Microsoft 365",
				SourceMatch:       "programs",
				CVEs:              []string{"CVE-123"},
				ResolvedInVersion: "1.0.0",
			},
		},
		{
			name: "Valid rule with empty SourceMatch",
			rule: CVEMatchingRule{
				NameLikeMatch:     "Microsoft 365",
				CVEs:              []string{"CVE-123"},
				ResolvedInVersion: "1.0.0",
			},
		},
		{
			name: "Empty CVEs",
			rule: CVEMatchingRule{
				NameLikeMatch:     "Microsoft 365",
				SourceMatch:       "programs",
				ResolvedInVersion: "1.0.0",
			},
			err: MissingCVEsErr,
		},
		{
			name: "Empty NameLikeMatch",
			rule: CVEMatchingRule{
				SourceMatch:       "programs",
				CVEs:              []string{"CVE-123"},
				ResolvedInVersion: "1.0.0",
			},
			err: MissingNameLikeMatch,
		},
		{
			name: "Empty ResolvedInVersion",
			rule: CVEMatchingRule{
				NameLikeMatch: "Microsoft 365",
				SourceMatch:   "programs",
				CVEs:          []string{"CVE-123"},
			},
			err: MissingResolvedInVersionErr,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.rule.validate()
			if tt.err == nil {
				require.NoError(t, err)
			} else {
				require.Equal(t, tt.err, err)
			}
		})
	}
}

func TestValidateAll(t *testing.T) {
	rules := getCVEMatchingRules()
	err := rules.ValidateAll()
	require.NoError(t, err)
}

func TestCheckCustomVulnerabilities(t *testing.T) {
	ds := new(mock.Store)
	sw := []fleet.Software{
		{
			ID:      1,
			Name:    "Microsoft 365 - en-us",
			Version: "16.0.17531.20152",
			Source:  "programs",
		},
		{
			ID:      2,
			Name:    "Microsoft 365 - en-us",
			Version: "16.0.17425.20176",
			Source:  "programs",
		},
		{
			ID:      3,
			Name:    "Microsoft 365 - en-us",
			Version: "16.0.17628.20144",
			Source:  "programs",
		},
	}

	t.Run("New Vulns return all inserted", func(t *testing.T) {
		ds.ListSoftwareForVulnDetectionFunc = func(ctx context.Context, filter fleet.VulnSoftwareFilter) ([]fleet.Software, error) {
			return sw, nil
		}

		var insertCount int
		ds.InsertSoftwareVulnerabilityFunc = func(ctx context.Context, vuln fleet.SoftwareVulnerability, source fleet.VulnerabilitySource) (bool, error) {
			insertCount++
			require.Equal(t, fleet.CustomSource, source)
			return true, nil
		}

		ds.DeleteOutOfDateVulnerabilitiesFunc = func(ctx context.Context, source fleet.VulnerabilitySource, duration time.Duration) error {
			require.Equal(t, fleet.CustomSource, source)
			return nil
		}

		ctx := context.Background()
		vulns, err := CheckCustomVulnerabilities(ctx, ds, log.NewNopLogger(), 1*time.Hour)
		require.NoError(t, err)
		require.Equal(t, 8, insertCount)
		require.Len(t, vulns, 8)
		require.True(t, ds.DeleteOutOfDateVulnerabilitiesFuncInvoked)

		expected := []fleet.SoftwareVulnerability{
			{
				SoftwareID:        1,
				CVE:               "CVE-2024-30101",
				ResolvedInVersion: ptr.String("16.0.17628.20144"),
			},
			{
				SoftwareID:        1,
				CVE:               "CVE-2024-30102",
				ResolvedInVersion: ptr.String("16.0.17628.20144"),
			},
			{
				SoftwareID:        1,
				CVE:               "CVE-2024-30103",
				ResolvedInVersion: ptr.String("16.0.17628.20144"),
			},
			{
				SoftwareID:        1,
				CVE:               "CVE-2024-30104",
				ResolvedInVersion: ptr.String("16.0.17628.20144"),
			},
			{
				SoftwareID:        2,
				CVE:               "CVE-2024-30101",
				ResolvedInVersion: ptr.String("16.0.17628.20144"),
			},
			{
				SoftwareID:        2,
				CVE:               "CVE-2024-30102",
				ResolvedInVersion: ptr.String("16.0.17628.20144"),
			},
			{
				SoftwareID:        2,
				CVE:               "CVE-2024-30103",
				ResolvedInVersion: ptr.String("16.0.17628.20144"),
			},
			{
				SoftwareID:        2,
				CVE:               "CVE-2024-30104",
				ResolvedInVersion: ptr.String("16.0.17628.20144"),
			},
		}

		require.Equal(t, expected, vulns)
	})

	t.Run("Existing Vulns are not inserted", func(t *testing.T) {
		ds.DeleteOutOfDateVulnerabilitiesFuncInvoked = false

		ds.ListSoftwareForVulnDetectionFunc = func(ctx context.Context, filter fleet.VulnSoftwareFilter) ([]fleet.Software, error) {
			return sw, nil
		}

		var insertCount int
		ds.InsertSoftwareVulnerabilityFunc = func(ctx context.Context, vuln fleet.SoftwareVulnerability, source fleet.VulnerabilitySource) (bool, error) {
			insertCount++
			require.Equal(t, fleet.CustomSource, source)
			return false, nil
		}

		ds.DeleteOutOfDateVulnerabilitiesFunc = func(ctx context.Context, source fleet.VulnerabilitySource, duration time.Duration) error {
			require.Equal(t, fleet.CustomSource, source)
			return nil
		}

		ctx := context.Background()
		vulns, err := CheckCustomVulnerabilities(ctx, ds, log.NewNopLogger(), 1*time.Hour)
		require.NoError(t, err)
		require.True(t, ds.DeleteOutOfDateVulnerabilitiesFuncInvoked)
		require.Equal(t, 8, insertCount)
		require.Len(t, vulns, 0)
	})
}
