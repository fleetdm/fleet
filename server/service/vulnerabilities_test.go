package service

import (
	"context"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/stretchr/testify/require"
)

var validSortColumns = []string{
	"cve",
	"host_count",
	"host_count_updated_at",
	"created_at",
}

func TestListVulnerabilities(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)

	ds.ListVulnerabilitiesFunc = func(cxt context.Context, opt fleet.VulnListOptions) ([]fleet.VulnerabilityWithMetadata, error) {
		return []fleet.VulnerabilityWithMetadata{
			{
				CVEMeta: fleet.CVEMeta{
					CVE:         "CVE-2019-1234",
					Description: "A vulnerability",
				},
				CreatedAt:          time.Now(),
				HostCount:          10,
				HostCountUpdatedAt: time.Now(),
			},
		}, nil
	}

	t.Run("no list options", func(t *testing.T) {
		_, err := svc.ListVulnerabilities(ctx, fleet.VulnListOptions{}, validSortColumns)
		require.NoError(t, err)
	})

	t.Run("can only sort by supported columns", func(t *testing.T) {
		// invalid order key
		opts := fleet.VulnListOptions{ListOptions: fleet.ListOptions{
			OrderKey: "invalid",
		}}
		_, err := svc.ListVulnerabilities(ctx, opts, validSortColumns)
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid order key")

		// valid order key
		opts = fleet.VulnListOptions{ListOptions: fleet.ListOptions{
			OrderKey: "cve",
		}}
		_, err = svc.ListVulnerabilities(ctx, opts, validSortColumns)
		require.NoError(t, err)
	})
}
