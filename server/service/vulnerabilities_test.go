package service

import (
	"context"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
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
	ctx = viewer.NewContext(ctx, viewer.Viewer{User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}})

	ds.ListVulnerabilitiesFunc = func(cxt context.Context, opt fleet.VulnListOptions) ([]fleet.VulnerabilityWithMetadata, *fleet.PaginationMetadata, error) {
		return []fleet.VulnerabilityWithMetadata{
			{
				CVEMeta: fleet.CVEMeta{
					CVE:         "CVE-2019-1234",
					Description: "A vulnerability",
				},
				CreatedAt: time.Now(),
				HostCount: 10,
			},
		}, nil, nil
	}

	t.Run("no list options", func(t *testing.T) {
		_, _, err := svc.ListVulnerabilities(ctx, fleet.VulnListOptions{})
		require.NoError(t, err)
	})

	t.Run("can only sort by supported columns", func(t *testing.T) {
		// invalid order key
		opts := fleet.VulnListOptions{ListOptions: fleet.ListOptions{
			OrderKey: "invalid",
		}, ValidSortColumns: freeValidVulnSortColumns}

		_, _, err := svc.ListVulnerabilities(ctx, opts)
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid order key")

		// valid order key
		opts.OrderKey = "cve"
		_, _, err = svc.ListVulnerabilities(ctx, opts)
		require.NoError(t, err)
	})
}
