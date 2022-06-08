package main

import (
	"context"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	kitlog "github.com/go-kit/kit/log"
	"github.com/stretchr/testify/require"
)

func TestFilterRecentVulns(t *testing.T) {
	t.Run("no NVD nor OVAL vulns", func(t *testing.T) {
		ctx := context.Background()
		ds := new(mock.Store)
		logger := kitlog.NewNopLogger()

		result := filterRecentVulns(ctx, ds, logger, nil, nil, 2*time.Hour)
		require.Empty(t, result)
	})

	t.Run("filters both NVD and OVAL vulns based on max age", func(t *testing.T) {
		ctx := context.Background()
		ds := new(mock.Store)
		logger := kitlog.NewNopLogger()

		ds.ListCVEsFunc = func(ctx context.Context, maxAge time.Duration) ([]fleet.CVEMeta, error) {
			return []fleet.CVEMeta{
				{CVE: "cve-recent-1"},
				{CVE: "cve-recent-2"},
				{CVE: "cve-recent-3"},
			}, nil
		}

		ovalVulns := []fleet.SoftwareVulnerability{
			{CVE: "cve-recent-1"},
			{CVE: "cve-recent-2"},
			{CVE: "cve-outdated-1"},
		}

		nvdVulns := []fleet.SoftwareVulnerability{
			{CVE: "cve-recent-1"},
			{CVE: "cve-recent-3"},
			{CVE: "cve-outdated-2"},
			{CVE: "cve-outdated-3"},
		}

		maxAge := 30 * 24 * time.Hour

		expected := []string{
			"cve-recent-1",
			"cve-recent-2",
			"cve-recent-3",
		}

		var actual []string
		result := filterRecentVulns(ctx, ds, logger, nvdVulns, ovalVulns, maxAge)
		for _, r := range result {
			actual = append(actual, r.CVE)
		}

		require.ElementsMatch(t, expected, actual)
	})
}
